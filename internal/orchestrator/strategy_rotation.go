package orchestrator

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/guilhermecosta/wpp-gateway/internal/domain"
)

// RotationStrategy: distributes messages across all available instances,
// selecting the one with the most remaining daily budget.
type RotationStrategy struct{}

func (s *RotationStrategy) Name() domain.Strategy {
	return domain.StrategyRotation
}

func (s *RotationStrategy) SelectInstance(_ context.Context, instances []domain.Instance) (*domain.Instance, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("no available instances")
	}

	// Select instance with highest remaining budget
	var best *domain.Instance
	bestRemaining := -1

	for i := range instances {
		inst := &instances[i]
		remaining := inst.DailyBudget - inst.MessagesToday

		// Skip instances that exhausted their daily budget
		if remaining <= 0 {
			continue
		}

		// Skip instances that exhausted their hourly budget
		hourRemaining := inst.HourlyBudget - inst.MessagesHour
		if hourRemaining <= 0 {
			continue
		}

		if remaining > bestRemaining {
			bestRemaining = remaining
			best = inst
		}
	}

	if best == nil {
		return nil, fmt.Errorf("all instances exhausted their budget")
	}

	return best, nil
}

func (s *RotationStrategy) OnInstanceFailed(_ context.Context, failedID uuid.UUID, instances []domain.Instance) (*domain.Instance, error) {
	// Simply re-select from remaining instances
	var remaining []domain.Instance
	for _, inst := range instances {
		if inst.ID != failedID && inst.Status == domain.StatusAvailable {
			remaining = append(remaining, inst)
		}
	}

	return (&RotationStrategy{}).SelectInstance(context.Background(), remaining)
}
