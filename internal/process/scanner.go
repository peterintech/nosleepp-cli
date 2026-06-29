package process

import (
	"context"

	"nosleep/internal/agent"
)

type Scanner interface {
	Scan(ctx context.Context) ([]agent.Process, error)
}
