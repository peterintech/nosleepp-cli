//go:build darwin

package process

import (
	"context"
	"os/exec"

	"nosleepp/internal/agent"
)

type darwinScanner struct{}

func NewScanner() Scanner {
	return darwinScanner{}
}

func (darwinScanner) Scan(ctx context.Context) ([]agent.Process, error) {
	cmd := exec.CommandContext(ctx, "ps", "-axo", "pid=,ppid=,time=,comm=")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseDarwinPSOutput(string(output))
}
