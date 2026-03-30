package orchestrator

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/guilhermecosta/wpp-gateway/internal/domain"
)

// FailoverStrategy: only the highest-priority available instance is active.
// If it fails, promote the next by priority.
type FailoverStrategy struct{}

func (s *FailoverStrategy) Name() domain.Strategy {
	return domain.StrategyFailover
}

func (s *FailoverStrategy) SelectInstance(_ context.Context, instances []domain.Instance) (*domain.Instance, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("no available instances")
	}

	// Instances are expected sorted by priority DESC.
	// The first available instance is the active one.
	best := instances[0]
	for _, inst := range instances {
		if inst.Priority > best.Priority {
			best = inst
		}
	}

	return &best, nil
}

func (s *FailoverStrategy) OnInstanceFailed(_ context.Context, failedID uuid.UUID, instances []domain.Instance) (*domain.Instance, error) {
	// Filter out the failed instance and pick the next highest priority
	var candidates []domain.Instance
	for _, inst := range instances {
		if inst.ID != failedID && inst.Status == domain.StatusAvailable {
			candidates = append(candidates, inst)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no failover candidate available")
	}

	best := candidates[0]
	for _, inst := range candidates[1:] {
		if inst.Priority > best.Priority {
			best = inst
		}
	}

	return &best, nil
}
