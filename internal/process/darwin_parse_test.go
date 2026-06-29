package process

import (
	"testing"
	"time"
)

func TestParseDarwinCPUTime(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Duration
	}{
		{name: "minutes seconds", input: "03:14", want: 3*time.Minute + 14*time.Second},
		{name: "minutes fractional seconds", input: "03:14.25", want: 3*time.Minute + 14250*time.Millisecond},
		{name: "hours minutes seconds", input: "02:03:14", want: 2*time.Hour + 3*time.Minute + 14*time.Second},
		{name: "days hours minutes seconds", input: "1-02:03:14", want: 26*time.Hour + 3*time.Minute + 14*time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDarwinCPUTime(tt.input)
			if err != nil {
				t.Fatalf("parseDarwinCPUTime returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

func TestParseDarwinPSOutput(t *testing.T) {
	output := `
	  101     1   03:14 /Applications/Codex.app/Contents/MacOS/Codex
	  202   101 1-02:03:14 /usr/local/bin/node
	  bad line
	`

	processes, err := parseDarwinPSOutput(output)
	if err != nil {
		t.Fatalf("parseDarwinPSOutput returned error: %v", err)
	}
	if len(processes) != 2 {
		t.Fatalf("expected 2 processes, got %#v", processes)
	}
	if processes[0].PID != 101 || processes[0].ParentPID != 1 || processes[0].Name != "Codex" {
		t.Fatalf("unexpected first process: %#v", processes[0])
	}
	if processes[1].PID != 202 || processes[1].ParentPID != 101 || processes[1].Name != "node" {
		t.Fatalf("unexpected second process: %#v", processes[1])
	}
	if processes[1].CPUTime != 26*time.Hour+3*time.Minute+14*time.Second {
		t.Fatalf("unexpected CPU time: %s", processes[1].CPUTime)
	}
}
