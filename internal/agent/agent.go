package agent

import (
	"sort"
	"strings"
	"time"
)

type Process struct {
	PID       int
	ParentPID int
	Name      string
	CPUTime   time.Duration
}

type Profile struct {
	Name      string
	Processes []string
}

type Match struct {
	Agent      string   `json:"agent"`
	PID        int      `json:"pid"`
	Executable string   `json:"executable"`
	Status     string   `json:"status"`
	CPUDeltaMS int64    `json:"cpu_delta_ms"`
	Children   int      `json:"children"`
	Evidence   []string `json:"evidence"`
}

func DefaultProfiles() []Profile {
	return []Profile{
		{Name: "Codex", Processes: []string{"codex", "Codex"}},
		{Name: "Claude Code", Processes: []string{"claude"}},
		{Name: "OpenCode", Processes: []string{"opencode"}},
		{Name: "Antigravity", Processes: []string{"antigravity"}},
		{Name: "Cursor", Processes: []string{"Cursor", "cursor"}},
	}
}

func UpsertProfile(profiles []Profile, next Profile) []Profile {
	for i, existing := range profiles {
		if strings.EqualFold(existing.Name, next.Name) {
			profiles[i] = next
			return profiles
		}
	}
	return append(profiles, next)
}

func MatchProcesses(profiles []Profile, processes []Process) []Match {
	signatures := compileSignatures(profiles)
	seen := map[int]bool{}
	matches := make([]Match, 0)

	for _, proc := range processes {
		if seen[proc.PID] {
			continue
		}

		agentName, ok := signatures[normalizeProcessName(proc.Name)]
		if !ok {
			continue
		}

		seen[proc.PID] = true
		matches = append(matches, Match{
			Agent:      agentName,
			PID:        proc.PID,
			Executable: proc.Name,
			Status:     "running",
		})
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Agent == matches[j].Agent {
			return matches[i].PID < matches[j].PID
		}
		return matches[i].Agent < matches[j].Agent
	})

	return matches
}

type ActivityOptions struct {
	CPUThreshold time.Duration
	IncludeIdle  bool
}

func DetectActivity(profiles []Profile, before []Process, after []Process, options ActivityOptions) []Match {
	if options.CPUThreshold <= 0 {
		options.CPUThreshold = 250 * time.Millisecond
	}

	signatures := compileSignatures(profiles)
	beforeByPID := indexByPID(before)
	afterByPID := indexByPID(after)
	beforeChildren := childrenByParent(before)
	afterChildren := childrenByParent(after)

	matches := make([]Match, 0)
	seen := map[int]bool{}
	for _, proc := range after {
		if seen[proc.PID] {
			continue
		}

		agentName, ok := signatures[normalizeProcessName(proc.Name)]
		if !ok {
			continue
		}

		seen[proc.PID] = true
		descendantPIDs := collectDescendants(proc.PID, afterChildren)
		beforeDescendantPIDs := collectDescendants(proc.PID, beforeChildren)
		cpuDelta := processDelta(proc.PID, beforeByPID, afterByPID)
		for _, pid := range descendantPIDs {
			cpuDelta += processDelta(pid, beforeByPID, afterByPID)
		}

		evidence := make([]string, 0)
		if cpuDelta >= options.CPUThreshold {
			evidence = append(evidence, "cpu")
		}
		if descendantsChanged(beforeDescendantPIDs, descendantPIDs) {
			evidence = append(evidence, "children")
		}

		status := "idle"
		if len(evidence) > 0 {
			status = "working"
		}
		if status == "idle" && !options.IncludeIdle {
			continue
		}

		matches = append(matches, Match{
			Agent:      agentName,
			PID:        proc.PID,
			Executable: proc.Name,
			Status:     status,
			CPUDeltaMS: cpuDelta.Milliseconds(),
			Children:   len(descendantPIDs),
			Evidence:   evidence,
		})
	}

	sortMatches(matches)
	return matches
}

func compileSignatures(profiles []Profile) map[string]string {
	signatures := map[string]string{}
	for _, profile := range profiles {
		for _, processName := range profile.Processes {
			normalized := normalizeProcessName(processName)
			if normalized != "" {
				signatures[normalized] = profile.Name
			}
		}
	}
	return signatures
}

func normalizeProcessName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.TrimSuffix(name, ".exe")
	return name
}

func indexByPID(processes []Process) map[int]Process {
	index := map[int]Process{}
	for _, proc := range processes {
		index[proc.PID] = proc
	}
	return index
}

func childrenByParent(processes []Process) map[int][]int {
	children := map[int][]int{}
	for _, proc := range processes {
		if proc.ParentPID != 0 {
			children[proc.ParentPID] = append(children[proc.ParentPID], proc.PID)
		}
	}
	return children
}

func collectDescendants(rootPID int, children map[int][]int) []int {
	descendants := make([]int, 0)
	queue := append([]int(nil), children[rootPID]...)
	seen := map[int]bool{}
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		if seen[pid] {
			continue
		}
		seen[pid] = true
		descendants = append(descendants, pid)
		queue = append(queue, children[pid]...)
	}
	sort.Ints(descendants)
	return descendants
}

func processDelta(pid int, before map[int]Process, after map[int]Process) time.Duration {
	next, ok := after[pid]
	if !ok {
		return 0
	}
	prev, ok := before[pid]
	if !ok {
		return 0
	}
	if next.CPUTime <= prev.CPUTime {
		return 0
	}
	return next.CPUTime - prev.CPUTime
}

func descendantsChanged(before []int, after []int) bool {
	if len(before) != len(after) {
		return true
	}
	for i := range before {
		if before[i] != after[i] {
			return true
		}
	}
	return false
}

func sortMatches(matches []Match) {
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Agent == matches[j].Agent {
			return matches[i].PID < matches[j].PID
		}
		return matches[i].Agent < matches[j].Agent
	})
}
