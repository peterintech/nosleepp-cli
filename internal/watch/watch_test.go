package watch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/peterintech/nosleepp/internal/agent"
)

type fakeScanner struct {
	calls int
	sets  [][]agent.Process
	err   error
}

func (s *fakeScanner) Scan(ctx context.Context) ([]agent.Process, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.calls >= len(s.sets) {
		return s.sets[len(s.sets)-1], nil
	}
	next := s.sets[s.calls]
	s.calls++
	return next, nil
}

type fakePower struct {
	acquires int
	releases int
	err      error
}

func (p *fakePower) Acquire() error {
	p.acquires++
	return p.err
}

func (p *fakePower) Release() error {
	p.releases++
	return p.err
}

func TestWatcherAcquireOnceAndReleaseWhenAgentsFinish(t *testing.T) {
	scanner := &fakeScanner{sets: [][]agent.Process{
		{{PID: 1, Name: "codex.exe", CPUTime: time.Second}},
		{{PID: 1, Name: "codex.exe", CPUTime: 1500 * time.Millisecond}},
		{{PID: 1, Name: "codex.exe", CPUTime: 1500 * time.Millisecond}},
		{},
	}}
	power := &fakePower{}

	watcher := NewWatcher(scanner, power, agent.DefaultProfiles(), Options{Interval: time.Millisecond, Sample: time.Nanosecond, Quiet: time.Nanosecond})
	if err := watcher.Run(context.Background()); err != nil {
		t.Fatalf("watcher returned error: %v", err)
	}

	if power.acquires != 1 {
		t.Fatalf("expected one acquire, got %d", power.acquires)
	}
	if power.releases != 1 {
		t.Fatalf("expected one release, got %d", power.releases)
	}
}

func TestWatcherOnceReturnsNoAgents(t *testing.T) {
	scanner := &fakeScanner{sets: [][]agent.Process{{}, {}}}
	power := &fakePower{}

	watcher := NewWatcher(scanner, power, agent.DefaultProfiles(), Options{Once: true, Sample: time.Nanosecond})
	err := watcher.Run(context.Background())

	if !errors.Is(err, ErrNoAgents) {
		t.Fatalf("expected ErrNoAgents, got %v", err)
	}
	if power.acquires != 0 || power.releases != 0 {
		t.Fatalf("expected no power calls, got acquire=%d release=%d", power.acquires, power.releases)
	}
}

func TestWatcherReleasesOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	scanner := &fakeScanner{sets: [][]agent.Process{
		{{PID: 1, Name: "codex.exe", CPUTime: time.Second}},
		{{PID: 1, Name: "codex.exe", CPUTime: 1500 * time.Millisecond}},
	}}
	power := &fakePower{}

	var notifications int
	watcher := NewWatcher(scanner, power, agent.DefaultProfiles(), Options{
		Interval: time.Hour,
		Sample:   time.Nanosecond,
		OnChange: func(matches []agent.Match, state State) {
			notifications++
			cancel()
		},
	})

	err := watcher.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if power.acquires != 1 {
		t.Fatalf("expected one acquire, got %d", power.acquires)
	}
	if power.releases != 1 {
		t.Fatalf("expected release during cleanup, got %d", power.releases)
	}
	if notifications != 1 {
		t.Fatalf("expected one notification, got %d", notifications)
	}
}

func TestWatcherDoesNotAcquireForIdleOpenAgents(t *testing.T) {
	scanner := &fakeScanner{sets: [][]agent.Process{
		{{PID: 1, Name: "codex.exe", CPUTime: time.Second}},
		{{PID: 1, Name: "codex.exe", CPUTime: time.Second}},
	}}
	power := &fakePower{}

	watcher := NewWatcher(scanner, power, agent.DefaultProfiles(), Options{Sample: time.Nanosecond})
	if err := watcher.Run(context.Background()); err != nil {
		t.Fatalf("watcher returned error: %v", err)
	}
	if power.acquires != 0 || power.releases != 0 {
		t.Fatalf("expected no power calls for idle agent, got acquire=%d release=%d", power.acquires, power.releases)
	}
}

func TestWatcherKeepsLockDuringQuietWindow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	scanner := &fakeScanner{sets: [][]agent.Process{
		{{PID: 1, Name: "codex.exe", CPUTime: time.Second}},
		{{PID: 1, Name: "codex.exe", CPUTime: 1500 * time.Millisecond}},
		{{PID: 1, Name: "codex.exe", CPUTime: 1500 * time.Millisecond}},
		{{PID: 1, Name: "codex.exe", CPUTime: 1500 * time.Millisecond}},
	}}
	power := &fakePower{}

	var sawQuiet bool
	watcher := NewWatcher(scanner, power, agent.DefaultProfiles(), Options{
		Interval: time.Nanosecond,
		Sample:   time.Nanosecond,
		Quiet:    time.Hour,
		OnChange: func(matches []agent.Match, state State) {
			if state == StateQuiet {
				sawQuiet = true
				cancel()
			}
		},
	})

	err := watcher.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if !sawQuiet {
		t.Fatalf("expected watcher to enter quiet state")
	}
	if power.acquires != 1 {
		t.Fatalf("expected one acquire, got %d", power.acquires)
	}
	if power.releases != 1 {
		t.Fatalf("expected cleanup release, got %d", power.releases)
	}
}
