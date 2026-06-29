package agent

import (
	"testing"
	"time"
)

func TestMatchProcessesMatchesCaseInsensitiveAndTrimExe(t *testing.T) {
	matches := MatchProcesses(DefaultProfiles(), []Process{
		{PID: 10, Name: "CODEX.EXE"},
		{PID: 11, Name: "cursor"},
		{PID: 12, Name: "notepad.exe"},
	})

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d: %#v", len(matches), matches)
	}
	if matches[0].Agent != "Codex" || matches[0].PID != 10 {
		t.Fatalf("expected Codex pid 10 first, got %#v", matches[0])
	}
	if matches[1].Agent != "Cursor" || matches[1].PID != 11 {
		t.Fatalf("expected Cursor pid 11 second, got %#v", matches[1])
	}
}

func TestUpsertProfileOverridesExistingAgent(t *testing.T) {
	profiles := UpsertProfile(DefaultProfiles(), Profile{Name: "codex", Processes: []string{"custom-codex"}})
	matches := MatchProcesses(profiles, []Process{
		{PID: 10, Name: "codex.exe"},
		{PID: 11, Name: "custom-codex.exe"},
	})

	if len(matches) != 1 {
		t.Fatalf("expected one custom match, got %#v", matches)
	}
	if matches[0].PID != 11 {
		t.Fatalf("expected custom process to match after override, got %#v", matches[0])
	}
}

func TestMatchProcessesDeduplicatesPID(t *testing.T) {
	profiles := []Profile{
		{Name: "First", Processes: []string{"worker"}},
		{Name: "Second", Processes: []string{"worker.exe"}},
	}
	matches := MatchProcesses(profiles, []Process{
		{PID: 10, Name: "worker.exe"},
		{PID: 10, Name: "worker.exe"},
	})

	if len(matches) != 1 {
		t.Fatalf("expected duplicate PID to be returned once, got %#v", matches)
	}
}

func TestDetectActivitySkipsIdleAgentsByDefault(t *testing.T) {
	matches := DetectActivity(DefaultProfiles(),
		[]Process{{PID: 10, Name: "codex.exe", CPUTime: time.Second}},
		[]Process{{PID: 10, Name: "codex.exe", CPUTime: time.Second}},
		ActivityOptions{CPUThreshold: 250 * time.Millisecond},
	)

	if len(matches) != 0 {
		t.Fatalf("expected idle agent to be skipped, got %#v", matches)
	}
}

func TestDetectActivityMarksRootCPUDeltaWorking(t *testing.T) {
	matches := DetectActivity(DefaultProfiles(),
		[]Process{{PID: 10, Name: "codex.exe", CPUTime: time.Second}},
		[]Process{{PID: 10, Name: "codex.exe", CPUTime: 1500 * time.Millisecond}},
		ActivityOptions{CPUThreshold: 250 * time.Millisecond},
	)

	if len(matches) != 1 {
		t.Fatalf("expected working agent, got %#v", matches)
	}
	if matches[0].Status != "working" || matches[0].CPUDeltaMS != 500 {
		t.Fatalf("unexpected working match: %#v", matches[0])
	}
}

func TestDetectActivityMarksDescendantCPUDeltaWorking(t *testing.T) {
	matches := DetectActivity(DefaultProfiles(),
		[]Process{
			{PID: 10, Name: "codex.exe", CPUTime: time.Second},
			{PID: 11, ParentPID: 10, Name: "node.exe", CPUTime: time.Second},
		},
		[]Process{
			{PID: 10, Name: "codex.exe", CPUTime: time.Second},
			{PID: 11, ParentPID: 10, Name: "node.exe", CPUTime: 1400 * time.Millisecond},
		},
		ActivityOptions{CPUThreshold: 250 * time.Millisecond},
	)

	if len(matches) != 1 {
		t.Fatalf("expected descendant CPU activity, got %#v", matches)
	}
	if matches[0].Children != 1 || matches[0].CPUDeltaMS != 400 {
		t.Fatalf("unexpected descendant activity match: %#v", matches[0])
	}
}

func TestDetectActivityMarksDescendantChangesWorking(t *testing.T) {
	matches := DetectActivity(DefaultProfiles(),
		[]Process{{PID: 10, Name: "codex.exe", CPUTime: time.Second}},
		[]Process{
			{PID: 10, Name: "codex.exe", CPUTime: time.Second},
			{PID: 11, ParentPID: 10, Name: "node.exe", CPUTime: 0},
		},
		ActivityOptions{CPUThreshold: 250 * time.Millisecond},
	)

	if len(matches) != 1 {
		t.Fatalf("expected child process activity, got %#v", matches)
	}
	if matches[0].Evidence[0] != "children" {
		t.Fatalf("expected child evidence, got %#v", matches[0])
	}
}

func TestDetectActivityIncludesIdleAgentsWhenRequested(t *testing.T) {
	matches := DetectActivity(DefaultProfiles(),
		[]Process{{PID: 10, Name: "codex.exe", CPUTime: time.Second}},
		[]Process{{PID: 10, Name: "codex.exe", CPUTime: time.Second}},
		ActivityOptions{CPUThreshold: 250 * time.Millisecond, IncludeIdle: true},
	)

	if len(matches) != 1 {
		t.Fatalf("expected idle agent to be included, got %#v", matches)
	}
	if matches[0].Status != "idle" {
		t.Fatalf("expected idle status, got %#v", matches[0])
	}
}
