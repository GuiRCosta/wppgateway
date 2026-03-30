package orchestrator

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/guilhermecosta/wpp-gateway/internal/domain"
)

// HybridStrategy: uses rotation for normal operation but with failover
// behavior when instances are banned or degraded.
// Combines budget-based rotation with automatic failover on failure.
type HybridStrategy struct {
	rotation RotationStrategy
}

func (s *HybridStrategy) Name() domain.Strategy {
	return domain.StrategyHybrid
}

func (s *HybridStrategy) SelectInstance(ctx context.Context, instances []domain.Instance) (*domain.Instance, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("no available instances")
	}

	// Use rotation logic for selection
	return s.rotation.SelectInstance(ctx, instances)
}

func (s *HybridStrategy) OnInstanceFailed(ctx context.Context, failedID uuid.UUID, instances []domain.Instance) (*domain.Instance, error) {
	// Filter out failed instance and redistribute
	var candidates []domain.Instance
	for _, inst := range instances {
		if inst.ID != failedID && inst.Status == domain.StatusAvailable {
			candidates = append(candidates, inst)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no instances available after failover")
	}

	return s.rotation.SelectInstance(ctx, candidates)
}
