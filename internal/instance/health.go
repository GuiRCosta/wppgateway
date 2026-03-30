package instance

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/guilhermecosta/wpp-gateway/internal/domain"
)

// HealthMonitor checks instance health periodically and triggers failover.
type HealthMonitor struct {
	manager      *Manager
	instanceRepo domain.InstanceRepository
	onDegraded   func(instanceID, groupID uuid.UUID)
	onBanned     func(instanceID, groupID uuid.UUID)
	interval     time.Duration
	log          zerolog.Logger
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

type HealthConfig struct {
	Interval              time.Duration
	DeliveryRateThreshold float64
	ReconnectTimeout      time.Duration
}

func DefaultHealthConfig() HealthConfig {
	return HealthConfig{
		Interval:              30 * time.Second,
		DeliveryRateThreshold: 0.85,
		ReconnectTimeout:      10 * time.Second,
	}
}

func NewHealthMonitor(
	manager *Manager,
	instanceRepo domain.InstanceRepository,
	interval time.Duration,
	log zerolog.Logger,
) *HealthMonitor {
	return &HealthMonitor{
		manager:      manager,
		instanceRepo: instanceRepo,
		interval:     interval,
		log:          log.With().Str("component", "health_monitor").Logger(),
	}
}

// OnDegraded sets callback when an instance shows degradation signals.
func (h *HealthMonitor) OnDegraded(fn func(instanceID, groupID uuid.UUID)) {
	h.onDegraded = fn
}

// OnBanned sets callback when an instance is detected as banned/disconnected permanently.
func (h *HealthMonitor) OnBanned(fn func(instanceID, groupID uuid.UUID)) {
	h.onBanned = fn
}

// Start begins periodic health checking.
func (h *HealthMonitor) Start(ctx context.Context) {
	ctx, h.cancel = context.WithCancel(ctx)
	h.wg.Add(1)

	go func() {
		defer h.wg.Done()
		ticker := time.NewTicker(h.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.checkAll(ctx)
			}
		}
	}()

	h.log.Info().Dur("interval", h.interval).Msg("health monitor started")
}

// Stop gracefully stops the health monitor.
func (h *HealthMonitor) Stop() {
	if h.cancel != nil {
		h.cancel()
	}
	h.wg.Wait()
	h.log.Info().Msg("health monitor stopped")
}

func (h *HealthMonitor) checkAll(ctx context.Context) {
	h.manager.mu.RLock()
	connections := make(map[uuid.UUID]*Connection, len(h.manager.connections))
	for id, conn := range h.manager.connections {
		connections[id] = conn
	}
	h.manager.mu.RUnlock()

	for id, conn := range connections {
		h.checkInstance(ctx, id, conn)
	}
}

func (h *HealthMonitor) checkInstance(ctx context.Context, id uuid.UUID, conn *Connection) {
	inst, err := h.instanceRepo.FindByID(ctx, id)
	if err != nil || inst == nil {
		return
	}

	// Skip instances that are already banned or disconnected intentionally
	if inst.Status == domain.StatusBanned || inst.Status == domain.StatusDisconnected {
		return
	}

	// Check WebSocket connection
	if !conn.IsConnected() {
		h.log.Warn().
			Str("instance_id", id.String()).
			Msg("instance WebSocket disconnected")

		if err := h.instanceRepo.UpdateStatus(ctx, id, domain.StatusSuspect); err != nil {
			h.log.Error().Err(err).Msg("failed to update instance status")
		}

		if h.onDegraded != nil {
			h.onDegraded(id, inst.GroupID)
		}
		return
	}

	// Check delivery rate degradation
	if inst.DeliveryRate < 0.85 && inst.MessagesToday > 10 {
		h.log.Warn().
			Str("instance_id", id.String()).
			Float64("delivery_rate", inst.DeliveryRate).
			Msg("delivery rate below threshold")

		if inst.DeliveryRate < 0.80 {
			// Critical: mark as suspect
			if err := h.instanceRepo.UpdateStatus(ctx, id, domain.StatusSuspect); err != nil {
				h.log.Error().Err(err).Msg("failed to update instance status")
			}

			if h.onDegraded != nil {
				h.onDegraded(id, inst.GroupID)
			}
		}
		return
	}

	// Check budget exhaustion
	if inst.MessagesToday >= inst.DailyBudget && inst.Status == domain.StatusAvailable {
		h.log.Info().
			Str("instance_id", id.String()).
			Int("messages_today", inst.MessagesToday).
			Int("daily_budget", inst.DailyBudget).
			Msg("instance daily budget exhausted, setting to resting")

		if err := h.instanceRepo.UpdateStatus(ctx, id, domain.StatusResting); err != nil {
			h.log.Error().Err(err).Msg("failed to update instance status")
		}
	}
}
