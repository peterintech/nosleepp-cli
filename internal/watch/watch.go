package watch

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/peterintech/nosleepp/internal/agent"
)

var ErrNoAgents = errors.New("no running agents found")

type ProcessScanner interface {
	Scan(ctx context.Context) ([]agent.Process, error)
}

type PowerManager interface {
	Acquire() error
	Release() error
}

type Options struct {
	Interval     time.Duration
	Sample       time.Duration
	CPUThreshold time.Duration
	Quiet        time.Duration
	Once         bool
	OnChange     func(matches []agent.Match, state State)
}

type State string

const (
	StateWorking  State = "working"
	StateQuiet    State = "quiet"
	StateReleased State = "released"
)

type Watcher struct {
	scanner  ProcessScanner
	power    PowerManager
	profiles []agent.Profile
	options  Options
}

func NewWatcher(scanner ProcessScanner, powerManager PowerManager, profiles []agent.Profile, options Options) Watcher {
	if options.Interval <= 0 {
		options.Interval = 10 * time.Second
	}
	if options.Sample <= 0 {
		options.Sample = 2 * time.Second
	}
	if options.CPUThreshold <= 0 {
		options.CPUThreshold = 250 * time.Millisecond
	}
	if options.Quiet <= 0 {
		options.Quiet = 30 * time.Second
	}
	return Watcher{
		scanner:  scanner,
		power:    powerManager,
		profiles: profiles,
		options:  options,
	}
}

func (w Watcher) Run(ctx context.Context) (runErr error) {
	locked := false
	var lastActivity time.Time
	defer func() {
		if locked {
			if err := w.power.Release(); err != nil && runErr == nil {
				runErr = err
			}
		}
	}()

	var lastState string
	for {
		matches, err := w.sampleActivity(ctx, false)
		if err != nil {
			return err
		}

		if len(matches) > 0 {
			lastActivity = time.Now()
			if !locked {
				if err := w.power.Acquire(); err != nil {
					return err
				}
				locked = true
			}
			w.notify(matches, StateWorking, &lastState)

			if w.options.Once {
				return nil
			}
		} else {
			if w.options.Once {
				return ErrNoAgents
			}

			if locked && time.Since(lastActivity) < w.options.Quiet {
				w.notify(matches, StateQuiet, &lastState)
			} else {
				if locked {
					if err := w.power.Release(); err != nil {
						return err
					}
					locked = false
				}
				w.notify(matches, StateReleased, &lastState)
				return nil
			}
		}

		timer := time.NewTimer(w.options.Interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (w Watcher) sampleActivity(ctx context.Context, includeIdle bool) ([]agent.Match, error) {
	before, err := w.scanner.Scan(ctx)
	if err != nil {
		return nil, err
	}

	timer := time.NewTimer(w.options.Sample)
	select {
	case <-ctx.Done():
		timer.Stop()
		return nil, ctx.Err()
	case <-timer.C:
	}

	after, err := w.scanner.Scan(ctx)
	if err != nil {
		return nil, err
	}

	return agent.DetectActivity(w.profiles, before, after, agent.ActivityOptions{
		CPUThreshold: w.options.CPUThreshold,
		IncludeIdle:  includeIdle,
	}), nil
}

func (w Watcher) notify(matches []agent.Match, state State, lastState *string) {
	if w.options.OnChange == nil {
		return
	}

	key := stateKey(matches, state)
	if key == *lastState {
		return
	}
	*lastState = key
	w.options.OnChange(matches, state)
}

func stateKey(matches []agent.Match, state State) string {
	key := string(state)
	for _, match := range matches {
		key += "|" + match.Agent + ":" + match.Executable + ":" + strconv.Itoa(match.PID) + ":" + match.Status + ":" + strconv.FormatInt(match.CPUDeltaMS, 10)
	}
	return key
}
