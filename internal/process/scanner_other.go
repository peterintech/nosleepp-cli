//go:build !windows && !darwin

package process

import (
	"context"
	"errors"

	"github.com/peterintech/nosleepp/internal/agent"
)

type unsupportedScanner struct{}

func NewScanner() Scanner {
	return unsupportedScanner{}
}

func (unsupportedScanner) Scan(ctx context.Context) ([]agent.Process, error) {
	return nil, errors.New("process scanning is only supported on Windows and macOS")
}
