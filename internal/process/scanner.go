package process

import (
	"context"

	"github.com/peterintech/nosleepp/internal/agent"
)

type Scanner interface {
	Scan(ctx context.Context) ([]agent.Process, error)
}
