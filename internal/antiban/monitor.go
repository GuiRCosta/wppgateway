package antiban

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/guilhermecosta/wpp-gateway/internal/domain"
)

// Signal represents a detected anti-ban signal.
type Signal struct {
	InstanceID uuid.UUID
	GroupID    uuid.UUID
	Type       SignalType
	Value      float64
	Threshold  float64
	Action     Action
	DetectedAt time.Time
}

type SignalType string

const (
	SignalDeliveryLow     SignalType = "delivery_rate_low"
	SignalDeliveryCritical SignalType = "delivery_rate_critical"
	SignalShadowBan       SignalType = "shadow_ban_suspected"
	SignalReconnectFlap   SignalType = "reconnect_flapping"
	SignalBudgetExhausted SignalType = "budget_exhausted"
)

type Action string

const (
	ActionThrottle   Action = "throttle"
	ActionPause      Action = "pause"
	ActionFailover   Action = "failover"
	ActionAlert      Action = "alert"
)

// OperatingHours defines when message sending is allowed.
type OperatingHours struct {
	Timezone string           `json:"timezone"`
	Windows  []TimeWindow     `json:"windows"`
	Days     []time.Weekday   `json:"days"`
}

type TimeWindow struct {
	Start string `json:"start"` // "08:00"
	End   string `json:"end"`   // "18:00"
}

// Monitor watches instance health signals and triggers anti-ban actions.
type Monitor struct {
	instanceRepo domain.InstanceRepository
	signals      chan Signal
	onSignal     func(Signal)
	log          zerolog.Logger
	cancel       context.CancelFunc
	wg           sync.WaitGroup

	// Per-instance tracking
	reconnects map[uuid.UUID][]time.Time
	mu         sync.RWMutex
}

func NewMonitor(instanceRepo domain.InstanceRepository, log zerolog.Logger) *Monitor {
	return &Monitor{
		instanceRepo: instanceRepo,
		signals:      make(chan Signal, 100),
		log:          log.With().Str("component", "antiban_monitor").Logger(),
		reconnects:   make(map[uuid.UUID][]time.Time),
	}
}

func (m *Monitor) OnSignal(fn func(Signal)) {
	m.onSignal = fn
}

func (m *Monitor) Start(ctx context.Context) {
	ctx, m.cancel = context.WithCancel(ctx)

	// Signal processor
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case sig := <-m.signals:
				m.processSignal(sig)
			}
		}
	}()

	m.log.Info().Msg("anti-ban monitor started")
}

func (m *Monitor) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
}

// CheckDeliveryRate evaluates delivery rate and emits signals.
func (m *Monitor) CheckDeliveryRate(instanceID, groupID uuid.UUID, rate float64, messagesToday int) {
	if messagesToday < 10 {
		return // Not enough data
	}

	if rate < 0.80 {
		m.emit(Signal{
			InstanceID: instanceID,
			GroupID:    groupID,
			Type:       SignalDeliveryCritical,
			Value:      rate,
			Threshold:  0.80,
			Action:     ActionPause,
			DetectedAt: time.Now(),
		})
	} else if rate < 0.85 {
		m.emit(Signal{
			InstanceID: instanceID,
			GroupID:    groupID,
			Type:       SignalDeliveryLow,
			Value:      rate,
			Threshold:  0.85,
			Action:     ActionThrottle,
			DetectedAt: time.Now(),
		})
	}
}

// TrackReconnect records a reconnection event and detects flapping.
func (m *Monitor) TrackReconnect(instanceID, groupID uuid.UUID) {
	m.mu.Lock()
	now := time.Now()

	// Clean old entries (older than 5 min)
	cutoff := now.Add(-5 * time.Minute)
	var recent []time.Time
	for _, t := range m.reconnects[instanceID] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	recent = append(recent, now)
	m.reconnects[instanceID] = recent
	count := len(recent)
	m.mu.Unlock()

	if count >= 3 {
		m.emit(Signal{
			InstanceID: instanceID,
			GroupID:    groupID,
			Type:       SignalReconnectFlap,
			Value:      float64(count),
			Threshold:  3,
			Action:     ActionFailover,
			DetectedAt: now,
		})
	}
}

// CheckShadowBan detects potential shadow ban:
// messages are "sent" but none become "delivered" within threshold.
func (m *Monitor) CheckShadowBan(instanceID, groupID uuid.UUID, sentCount, deliveredCount int, window time.Duration) {
	if sentCount < 5 {
		return
	}

	deliveryRatio := float64(deliveredCount) / float64(sentCount)
	if deliveryRatio < 0.10 && window > 5*time.Minute {
		m.emit(Signal{
			InstanceID: instanceID,
			GroupID:    groupID,
			Type:       SignalShadowBan,
			Value:      deliveryRatio,
			Threshold:  0.10,
			Action:     ActionPause,
			DetectedAt: time.Now(),
		})
	}
}

// IsWithinOperatingHours checks if current time is within operating hours.
func IsWithinOperatingHours(hours OperatingHours) bool {
	loc, err := time.LoadLocation(hours.Timezone)
	if err != nil {
		return true // Default to allowing if timezone invalid
	}

	now := time.Now().In(loc)

	// Check day
	dayAllowed := false
	for _, d := range hours.Days {
		if now.Weekday() == d {
			dayAllowed = true
			break
		}
	}
	if !dayAllowed && len(hours.Days) > 0 {
		return false
	}

	// Check time windows
	currentMinutes := now.Hour()*60 + now.Minute()
	for _, w := range hours.Windows {
		startMinutes := parseTimeMinutes(w.Start)
		endMinutes := parseTimeMinutes(w.End)
		if currentMinutes >= startMinutes && currentMinutes < endMinutes {
			return true
		}
	}

	return len(hours.Windows) == 0 // If no windows defined, always allow
}

func (m *Monitor) emit(sig Signal) {
	select {
	case m.signals <- sig:
	default:
		m.log.Warn().Str("type", string(sig.Type)).Msg("signal channel full, dropping signal")
	}
}

func (m *Monitor) processSignal(sig Signal) {
	m.log.Warn().
		Str("instance_id", sig.InstanceID.String()).
		Str("type", string(sig.Type)).
		Str("action", string(sig.Action)).
		Float64("value", sig.Value).
		Float64("threshold", sig.Threshold).
		Msg("anti-ban signal detected")

	if m.onSignal != nil {
		m.onSignal(sig)
	}
}

func parseTimeMinutes(t string) int {
	if len(t) < 5 {
		return 0
	}
	h := int(t[0]-'0')*10 + int(t[1]-'0')
	m := int(t[3]-'0')*10 + int(t[4]-'0')
	return h*60 + m
}
