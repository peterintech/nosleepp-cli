package cmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"nosleepp/internal/agent"
)

type commandFakeScanner struct {
	calls int
	sets  [][]agent.Process
}

func (s *commandFakeScanner) Scan(ctx context.Context) ([]agent.Process, error) {
	if s.calls >= len(s.sets) {
		return s.sets[len(s.sets)-1], nil
	}
	next := s.sets[s.calls]
	s.calls++
	return next, nil
}

type commandFakePower struct {
	acquires int
	releases int
}

func (p *commandFakePower) Acquire() error {
	p.acquires++
	return nil
}

func (p *commandFakePower) Release() error {
	p.releases++
	return nil
}

func TestParseAgentFlag(t *testing.T) {
	profile, err := parseAgentFlag("Codex=codex,Codex.exe")
	if err != nil {
		t.Fatalf("parseAgentFlag returned error: %v", err)
	}
	if profile.Name != "Codex" || len(profile.Processes) != 2 {
		t.Fatalf("unexpected profile: %#v", profile)
	}
}

func TestListCommandJSON(t *testing.T) {
	var out bytes.Buffer
	opts := &options{interval: time.Millisecond, sample: 0, output: &out, errorOutput: &bytes.Buffer{}}
	opts.processScan = &commandFakeScanner{sets: [][]agent.Process{
		{{PID: 7, Name: "codex.exe", CPUTime: 100 * time.Millisecond}},
		{{PID: 7, Name: "codex.exe", CPUTime: 500 * time.Millisecond}},
	}}
	root := newRootCommand(opts)
	root.SetArgs([]string{"list", "--json", "--sample", "0s"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(out.String(), `"agent": "Codex"`) {
		t.Fatalf("expected JSON output to include Codex, got %s", out.String())
	}
}

func TestWatchOnceNoAgentsExitCode(t *testing.T) {
	var out bytes.Buffer
	opts := &options{interval: time.Millisecond, sample: 0, output: &out, errorOutput: &bytes.Buffer{}}
	opts.processScan = &commandFakeScanner{sets: [][]agent.Process{{}, {}}}
	opts.powerManager = &commandFakePower{}
	root := newRootCommand(opts)
	root.SetArgs([]string{"watch", "--once", "--sample", "0s"})

	err := root.Execute()
	var exitErr ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %v", err)
	}
	if exitErr.Code != 1 {
		t.Fatalf("expected exit code 1, got %d", exitErr.Code)
	}
}
