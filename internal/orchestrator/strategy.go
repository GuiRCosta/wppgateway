package orchestrator

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/guilhermecosta/wpp-gateway/internal/domain"
)

// Strategy defines how a group selects instances for sending messages.
type Strategy interface {
	// SelectInstance picks the best instance from available ones.
	SelectInstance(ctx context.Context, instances []domain.Instance) (*domain.Instance, error)

	// OnInstanceFailed handles failover when an instance becomes unavailable.
	OnInstanceFailed(ctx context.Context, failedID uuid.UUID, instances []domain.Instance) (*domain.Instance, error)

	// Name returns the strategy identifier.
	Name() domain.Strategy
}

func NewStrategy(strategy domain.Strategy) (Strategy, error) {
	switch strategy {
	case domain.StrategyFailover:
		return &FailoverStrategy{}, nil
	case domain.StrategyRotation:
		return &RotationStrategy{}, nil
	case domain.StrategyHybrid:
		return &HybridStrategy{}, nil
	default:
		return nil, fmt.Errorf("unknown strategy: %s", strategy)
	}
}
