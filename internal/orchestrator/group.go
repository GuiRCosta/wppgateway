package orchestrator

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/guilhermecosta/wpp-gateway/internal/domain"
	"github.com/guilhermecosta/wpp-gateway/internal/instance"
	"github.com/guilhermecosta/wpp-gateway/internal/webhook"
)

// GroupOrchestrator manages instance selection and failover for a group.
type GroupOrchestrator struct {
	groupRepo    domain.GroupRepository
	instanceRepo domain.InstanceRepository
	manager      *instance.Manager
	webhookEmit  *webhook.Emitter
	strategies   map[uuid.UUID]Strategy
	paused       map[uuid.UUID]bool
	mu           sync.RWMutex
	log          zerolog.Logger
}

func NewGroupOrchestrator(
	groupRepo domain.GroupRepository,
	instanceRepo domain.InstanceRepository,
	manager *instance.Manager,
	webhookEmit *webhook.Emitter,
	log zerolog.Logger,
) *GroupOrchestrator {
	return &GroupOrchestrator{
		groupRepo:    groupRepo,
		instanceRepo: instanceRepo,
		manager:      manager,
		webhookEmit:  webhookEmit,
		strategies:   make(map[uuid.UUID]Strategy),
		paused:       make(map[uuid.UUID]bool),
		log:          log,
	}
}

// SelectInstance picks the best instance for sending in a given group.
func (o *GroupOrchestrator) SelectInstance(ctx context.Context, groupID uuid.UUID) (*domain.Instance, error) {
	o.mu.RLock()
	if o.paused[groupID] {
		o.mu.RUnlock()
		return nil, fmt.Errorf("group is paused")
	}
	o.mu.RUnlock()

	strategy, err := o.getStrategy(ctx, groupID)
	if err != nil {
		return nil, err
	}

	instances, err := o.instanceRepo.FindAvailableByGroupID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to find available instances: %w", err)
	}

	// Also filter by actual connection status
	var connected []domain.Instance
	for _, inst := range instances {
		if conn, ok := o.manager.GetConnection(inst.ID); ok && conn.IsConnected() {
			connected = append(connected, inst)
		}
	}

	if len(connected) == 0 {
		return nil, fmt.Errorf("no connected instances available in group")
	}

	return strategy.SelectInstance(ctx, connected)
}

// HandleInstanceFailure triggers failover logic when an instance fails.
func (o *GroupOrchestrator) HandleInstanceFailure(ctx context.Context, groupID uuid.UUID, failedID uuid.UUID) (*domain.Instance, error) {
	strategy, err := o.getStrategy(ctx, groupID)
	if err != nil {
		return nil, err
	}

	instances, err := o.instanceRepo.FindByGroupID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to find instances: %w", err)
	}

	replacement, err := strategy.OnInstanceFailed(ctx, failedID, instances)
	if err != nil {
		o.log.Error().
			Str("group_id", groupID.String()).
			Str("failed_id", failedID.String()).
			Msg("no failover candidate found")
		return nil, err
	}

	o.log.Info().
		Str("group_id", groupID.String()).
		Str("failed_id", failedID.String()).
		Str("replacement_id", replacement.ID.String()).
		Msg("failover executed")

	return replacement, nil
}

// GetGroupStatus returns consolidated status for a group.
func (o *GroupOrchestrator) GetGroupStatus(ctx context.Context, groupID uuid.UUID) (*GroupStatus, error) {
	group, err := o.groupRepo.FindByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, fmt.Errorf("group not found")
	}

	instances, err := o.instanceRepo.FindByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}

	status := &GroupStatus{
		GroupID:   groupID,
		Name:      group.Name,
		Strategy:  group.Strategy,
		IsActive:  group.IsActive,
		Instances: make([]InstanceInfo, 0, len(instances)),
	}

	o.mu.RLock()
	status.IsPaused = o.paused[groupID]
	o.mu.RUnlock()

	for _, inst := range instances {
		info := InstanceInfo{
			ID:            inst.ID,
			PhoneNumber:   inst.PhoneNumber,
			DisplayName:   inst.DisplayName,
			Status:        inst.Status,
			Priority:      inst.Priority,
			DailyBudget:   inst.DailyBudget,
			MessagesToday: inst.MessagesToday,
			HourlyBudget:  inst.HourlyBudget,
			MessagesHour:  inst.MessagesHour,
			DeliveryRate:  inst.DeliveryRate,
		}

		if conn, ok := o.manager.GetConnection(inst.ID); ok {
			info.WSConnected = conn.IsConnected()
		}

		switch inst.Status {
		case domain.StatusAvailable:
			status.Available++
		case domain.StatusResting:
			status.Resting++
		case domain.StatusWarming:
			status.Warming++
		case domain.StatusBanned:
			status.Banned++
		case domain.StatusDisconnected, domain.StatusConnecting:
			status.Disconnected++
		}

		status.TotalBudget += inst.DailyBudget
		status.TotalSentToday += inst.MessagesToday
		status.Instances = append(status.Instances, info)
	}

	return status, nil
}

// PauseGroup pauses all dispatching for a group.
func (o *GroupOrchestrator) PauseGroup(groupID uuid.UUID) {
	o.mu.Lock()
	o.paused[groupID] = true
	o.mu.Unlock()
}

// ResumeGroup resumes dispatching for a group.
func (o *GroupOrchestrator) ResumeGroup(groupID uuid.UUID) {
	o.mu.Lock()
	delete(o.paused, groupID)
	o.mu.Unlock()
}

// IsPaused checks if a group is paused.
func (o *GroupOrchestrator) IsPaused(groupID uuid.UUID) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.paused[groupID]
}

func (o *GroupOrchestrator) getStrategy(ctx context.Context, groupID uuid.UUID) (Strategy, error) {
	o.mu.RLock()
	s, exists := o.strategies[groupID]
	o.mu.RUnlock()
	if exists {
		return s, nil
	}

	group, err := o.groupRepo.FindByID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to find group: %w", err)
	}
	if group == nil {
		return nil, fmt.Errorf("group not found")
	}

	strategy, err := NewStrategy(group.Strategy)
	if err != nil {
		return nil, err
	}

	o.mu.Lock()
	o.strategies[groupID] = strategy
	o.mu.Unlock()

	return strategy, nil
}

// InvalidateStrategy removes cached strategy (call after group strategy update).
func (o *GroupOrchestrator) InvalidateStrategy(groupID uuid.UUID) {
	o.mu.Lock()
	delete(o.strategies, groupID)
	o.mu.Unlock()
}

// GroupStatus holds consolidated status info for a group.
type GroupStatus struct {
	GroupID        uuid.UUID      `json:"group_id"`
	Name           string         `json:"name"`
	Strategy       domain.Strategy `json:"strategy"`
	IsActive       bool           `json:"is_active"`
	IsPaused       bool           `json:"is_paused"`
	Available      int            `json:"available"`
	Resting        int            `json:"resting"`
	Warming        int            `json:"warming"`
	Banned         int            `json:"banned"`
	Disconnected   int            `json:"disconnected"`
	TotalBudget    int            `json:"total_budget"`
	TotalSentToday int            `json:"total_sent_today"`
	Instances      []InstanceInfo `json:"instances"`
}

type InstanceInfo struct {
	ID            uuid.UUID            `json:"id"`
	PhoneNumber   *string              `json:"phone_number,omitempty"`
	DisplayName   *string              `json:"display_name,omitempty"`
	Status        domain.InstanceStatus `json:"status"`
	WSConnected   bool                 `json:"ws_connected"`
	Priority      int                  `json:"priority"`
	DailyBudget   int                  `json:"daily_budget"`
	MessagesToday int                  `json:"messages_today"`
	HourlyBudget  int                  `json:"hourly_budget"`
	MessagesHour  int                  `json:"messages_hour"`
	DeliveryRate  float64              `json:"delivery_rate"`
}
