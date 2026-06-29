package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"nosleep/internal/agent"
	"nosleep/internal/power"
	"nosleep/internal/process"
	"nosleep/internal/watch"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type ExitError struct {
	Code    int
	Message string
}

func (e ExitError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("exit code %d", e.Code)
}

type options struct {
	interval     time.Duration
	sample       time.Duration
	cpuThreshold time.Duration
	quiet        time.Duration
	agentFlags   []string
	configPath   string
	jsonOutput   bool
	includeAll   bool
	once         bool
	output       io.Writer
	errorOutput  io.Writer
	processScan  watch.ProcessScanner
	powerManager watch.PowerManager
}

func Execute() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	return NewRootCommand(os.Stdout, os.Stderr).ExecuteContext(ctx)
}

func NewRootCommand(stdout, stderr io.Writer) *cobra.Command {
	opts := &options{
		interval:     10 * time.Second,
		sample:       2 * time.Second,
		cpuThreshold: 250 * time.Millisecond,
		quiet:        30 * time.Second,
		output:       stdout,
		errorOutput:  stderr,
	}
	return newRootCommand(opts)
}

func newRootCommand(opts *options) *cobra.Command {
	root := &cobra.Command{
		Use:           "nosleep",
		Short:         "Keep your PC awake while AI agents are working",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringArrayVar(&opts.agentFlags, "agent", nil, "agent signature in name=process1,process2 format")
	root.PersistentFlags().StringVar(&opts.configPath, "config", "", "optional config path reserved for future use")

	root.AddCommand(newListCommand(opts))
	root.AddCommand(newWatchCommand(opts))
	root.AddCommand(newVersionCommand(opts.output))

	return root
}

func newListCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List working agent processes",
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := buildProfiles(opts.agentFlags)
			if err != nil {
				return ExitError{Code: 2, Message: err.Error()}
			}

			scanner := opts.processScan
			if scanner == nil {
				scanner = process.NewScanner()
			}

			before, err := scanner.Scan(cmd.Context())
			if err != nil {
				return err
			}

			if err := sleepContext(cmd.Context(), opts.sample); err != nil {
				return err
			}

			after, err := scanner.Scan(cmd.Context())
			if err != nil {
				return err
			}

			matches := agent.DetectActivity(profiles, before, after, agent.ActivityOptions{
				CPUThreshold: opts.cpuThreshold,
				IncludeIdle:  opts.includeAll,
			})
			return printMatches(opts.output, matches, opts.jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "print machine-readable JSON")
	cmd.Flags().DurationVar(&opts.sample, "sample", 2*time.Second, "activity sample window")
	cmd.Flags().DurationVar(&opts.cpuThreshold, "cpu-threshold", 250*time.Millisecond, "minimum CPU delta for working status")
	cmd.Flags().BoolVar(&opts.includeAll, "all", false, "include idle matching agent processes")
	return cmd
}

func newWatchCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Prevent system sleep while agents are working",
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := buildProfiles(opts.agentFlags)
			if err != nil {
				return ExitError{Code: 2, Message: err.Error()}
			}

			scanner := opts.processScan
			if scanner == nil {
				scanner = process.NewScanner()
			}

			powerManager := opts.powerManager
			if powerManager == nil {
				powerManager = power.NewManager()
			}

			watcher := watch.NewWatcher(scanner, powerManager, profiles, watch.Options{
				Interval:     opts.interval,
				Sample:       opts.sample,
				CPUThreshold: opts.cpuThreshold,
				Quiet:        opts.quiet,
				Once:         opts.once,
				OnChange: func(matches []agent.Match, state watch.State) {
					switch state {
					case watch.StateWorking:
						fmt.Fprintln(opts.output, "Agents running; preventing system sleep.")
					case watch.StateQuiet:
						fmt.Fprintf(opts.output, "No current agent activity; keeping PC awake for the %s quiet window.\n", opts.quiet)
					case watch.StateReleased:
						fmt.Fprintln(opts.output, "No agents running; normal sleep behavior restored.")
					}
					_ = printMatches(opts.output, matches, opts.jsonOutput)
				},
			})

			err = watcher.Run(cmd.Context())
			if errors.Is(err, watch.ErrNoAgents) {
				if opts.once {
					return ExitError{Code: 1, Message: "no working agents found"}
				}
				return nil
			}
			if err != nil {
				return ExitError{Code: 3, Message: err.Error()}
			}
			return nil
		},
	}

	cmd.Flags().DurationVar(&opts.interval, "interval", 10*time.Second, "polling interval")
	cmd.Flags().DurationVar(&opts.sample, "sample", 2*time.Second, "activity sample window")
	cmd.Flags().DurationVar(&opts.cpuThreshold, "cpu-threshold", 250*time.Millisecond, "minimum CPU delta for working status")
	cmd.Flags().DurationVar(&opts.quiet, "quiet", 30*time.Second, "quiet grace period before releasing sleep prevention")
	cmd.Flags().BoolVar(&opts.once, "once", false, "check once and exit")
	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "print machine-readable JSON")
	return cmd
}

func newVersionCommand(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(stdout, "nosleep %s (commit %s, built %s)\n", version, commit, date)
			return err
		},
	}
}

func buildProfiles(agentFlags []string) ([]agent.Profile, error) {
	profiles := agent.DefaultProfiles()
	for _, flag := range agentFlags {
		profile, err := parseAgentFlag(flag)
		if err != nil {
			return nil, err
		}
		profiles = agent.UpsertProfile(profiles, profile)
	}
	return profiles, nil
}

func parseAgentFlag(value string) (agent.Profile, error) {
	name, processes, ok := strings.Cut(value, "=")
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return agent.Profile{}, fmt.Errorf("invalid --agent %q: expected name=process1,process2", value)
	}

	var names []string
	for _, processName := range strings.Split(processes, ",") {
		processName = strings.TrimSpace(processName)
		if processName != "" {
			names = append(names, processName)
		}
	}
	if len(names) == 0 {
		return agent.Profile{}, fmt.Errorf("invalid --agent %q: at least one process name is required", value)
	}

	return agent.Profile{Name: name, Processes: names}, nil
}

func printMatches(w io.Writer, matches []agent.Match, asJSON bool) error {
	if asJSON {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(matches)
	}

	if len(matches) == 0 {
		_, err := fmt.Fprintln(w, "No working agents found.")
		return err
	}

	_, err := fmt.Fprintln(w, "AGENT\tPID\tPROCESS\tSTATUS\tCPU_DELTA\tCHILDREN\tEVIDENCE")
	if err != nil {
		return err
	}
	for _, match := range matches {
		evidence := strings.Join(match.Evidence, ",")
		if evidence == "" {
			evidence = "-"
		}
		if _, err := fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%dms\t%d\t%s\n", match.Agent, match.PID, match.Executable, match.Status, match.CPUDeltaMS, match.Children, evidence); err != nil {
			return err
		}
	}
	return nil
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return ctx.Err()
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
