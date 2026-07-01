package process

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/peterintech/nosleepp/internal/agent"
)

func parseDarwinPSOutput(output string) ([]agent.Process, error) {
	lines := strings.Split(output, "\n")
	processes := make([]agent.Process, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		parentPID, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		cpuTime, err := parseDarwinCPUTime(fields[2])
		if err != nil {
			continue
		}

		command := strings.Join(fields[3:], " ")
		processes = append(processes, agent.Process{
			PID:       pid,
			ParentPID: parentPID,
			Name:      filepath.Base(command),
			CPUTime:   cpuTime,
		})
	}
	return processes, nil
}

func parseDarwinCPUTime(value string) (time.Duration, error) {
	daySplit := strings.SplitN(value, "-", 2)
	days := 0
	timePart := value
	if len(daySplit) == 2 {
		parsedDays, err := strconv.Atoi(daySplit[0])
		if err != nil {
			return 0, err
		}
		days = parsedDays
		timePart = daySplit[1]
	}

	parts := strings.Split(timePart, ":")
	total := time.Duration(days) * 24 * time.Hour
	switch len(parts) {
	case 2:
		minutes, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
		seconds, err := parseSecondsDuration(parts[1])
		if err != nil {
			return 0, err
		}
		total += time.Duration(minutes)*time.Minute + seconds
	case 3:
		hours, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
		minutes, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, err
		}
		seconds, err := parseSecondsDuration(parts[2])
		if err != nil {
			return 0, err
		}
		total += time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + seconds
	default:
		return 0, strconv.ErrSyntax
	}

	return total, nil
}

func parseSecondsDuration(value string) (time.Duration, error) {
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(seconds * float64(time.Second)), nil
}
