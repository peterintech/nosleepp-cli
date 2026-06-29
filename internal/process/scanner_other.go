//go:build !windows

package process

import (
	"context"
	"errors"

	"nosleep/internal/agent"
)

type unsupportedScanner struct{}

func NewScanner() Scanner {
	return unsupportedScanner{}
}

func (unsupportedScanner) Scan(ctx context.Context) ([]agent.Process, error) {
	return nil, errors.New("process scanning is only supported on Windows in this MVP")
}
