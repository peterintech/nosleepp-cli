package process

import (
	"context"

	"nosleepp/internal/agent"
)

type Scanner interface {
	Scan(ctx context.Context) ([]agent.Process, error)
}
