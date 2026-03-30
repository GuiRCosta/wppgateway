package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type BudgetManager struct {
	client *redis.Client
}

func NewBudgetManager(client *redis.Client) *BudgetManager {
	return &BudgetManager{client: client}
}

func (b *BudgetManager) IncrementDaily(ctx context.Context, instanceID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("budget:daily:%s:%s", instanceID.String(), time.Now().UTC().Format("2006-01-02"))
	val, err := b.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	// Set TTL to 25 hours (auto-cleanup after day ends)
	b.client.Expire(ctx, key, 25*time.Hour)
	return val, nil
}

func (b *BudgetManager) GetDaily(ctx context.Context, instanceID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("budget:daily:%s:%s", instanceID.String(), time.Now().UTC().Format("2006-01-02"))
	val, err := b.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

func (b *BudgetManager) IncrementHourly(ctx context.Context, instanceID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("budget:hourly:%s:%s", instanceID.String(), time.Now().UTC().Format("2006-01-02T15"))
	val, err := b.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	// Set TTL to 2 hours
	b.client.Expire(ctx, key, 2*time.Hour)
	return val, nil
}

func (b *BudgetManager) GetHourly(ctx context.Context, instanceID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("budget:hourly:%s:%s", instanceID.String(), time.Now().UTC().Format("2006-01-02T15"))
	val, err := b.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// RemainingDaily returns how many messages can still be sent today.
func (b *BudgetManager) RemainingDaily(ctx context.Context, instanceID uuid.UUID, budget int) (int, error) {
	used, err := b.GetDaily(ctx, instanceID)
	if err != nil {
		return 0, err
	}
	remaining := budget - int(used)
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

// RemainingHourly returns how many messages can still be sent this hour.
func (b *BudgetManager) RemainingHourly(ctx context.Context, instanceID uuid.UUID, budget int) (int, error) {
	used, err := b.GetHourly(ctx, instanceID)
	if err != nil {
		return 0, err
	}
	remaining := budget - int(used)
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}
