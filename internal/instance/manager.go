package instance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"github.com/guilhermecosta/wpp-gateway/internal/domain"
	"github.com/guilhermecosta/wpp-gateway/internal/webhook"
)

// DeviceMapper persists the mapping between instance IDs and whatsmeow device JIDs.
type DeviceMapper interface {
	GetJID(ctx context.Context, instanceID uuid.UUID) (string, error)
	Upsert(ctx context.Context, instanceID uuid.UUID, jid string) error
	Delete(ctx context.Context, instanceID uuid.UUID) error
	GetAll(ctx context.Context) ([]DeviceMappingEntry, error)
}

type DeviceMappingEntry struct {
	InstanceID uuid.UUID
	JID        string
}

type Manager struct {
	connections  map[uuid.UUID]*Connection
	mu           sync.RWMutex
	container    *sqlstore.Container
	instanceRepo domain.InstanceRepository
	groupRepo    domain.GroupRepository
	deviceMapper DeviceMapper
	webhookEmit  *webhook.Emitter
	log          zerolog.Logger
}

func NewManager(
	container *sqlstore.Container,
	instanceRepo domain.InstanceRepository,
	groupRepo domain.GroupRepository,
	deviceMapper DeviceMapper,
	webhookEmit *webhook.Emitter,
	log zerolog.Logger,
) *Manager {
	return &Manager{
		connections:  make(map[uuid.UUID]*Connection),
		container:    container,
		instanceRepo: instanceRepo,
		groupRepo:    groupRepo,
		deviceMapper: deviceMapper,
		webhookEmit:  webhookEmit,
		log:          log,
	}
}

func (m *Manager) StartInstance(ctx context.Context, instanceID uuid.UUID) (<-chan whatsmeow.QRChannelItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.connections[instanceID]; exists {
		if conn.IsConnected() {
			return nil, fmt.Errorf("instance already connected")
		}
	}

	inst, err := m.instanceRepo.FindByID(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to find instance: %w", err)
	}
	if inst == nil {
		return nil, fmt.Errorf("instance not found")
	}

	device, err := m.getOrCreateDevice(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	conn := NewConnection(instanceID, inst.GroupID, device, m.log, m.handleEvent)
	m.connections[instanceID] = conn

	if err := m.instanceRepo.UpdateStatus(ctx, instanceID, domain.StatusConnecting); err != nil {
		m.log.Error().Err(err).Msg("failed to update instance status to connecting")
	}

	qrChan, err := conn.Connect(ctx)
	if err != nil {
		delete(m.connections, instanceID)
		_ = m.instanceRepo.UpdateStatus(ctx, instanceID, domain.StatusDisconnected)
		return nil, err
	}

	return qrChan, nil
}

func (m *Manager) PairPhone(ctx context.Context, instanceID uuid.UUID, phone string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, err := m.instanceRepo.FindByID(ctx, instanceID)
	if err != nil {
		return "", fmt.Errorf("failed to find instance: %w", err)
	}
	if inst == nil {
		return "", fmt.Errorf("instance not found")
	}

	device, err := m.getOrCreateDevice(ctx, instanceID)
	if err != nil {
		return "", fmt.Errorf("failed to get device: %w", err)
	}

	conn := NewConnection(instanceID, inst.GroupID, device, m.log, m.handleEvent)
	m.connections[instanceID] = conn

	code, err := conn.PairPhone(ctx, phone)
	if err != nil {
		delete(m.connections, instanceID)
		return "", err
	}

	return code, nil
}

func (m *Manager) StopInstance(_ context.Context, instanceID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, exists := m.connections[instanceID]
	if !exists {
		return nil
	}

	conn.Disconnect()
	delete(m.connections, instanceID)
	return nil
}

func (m *Manager) RestartInstance(ctx context.Context, instanceID uuid.UUID) (<-chan whatsmeow.QRChannelItem, error) {
	_ = m.StopInstance(ctx, instanceID)
	return m.StartInstance(ctx, instanceID)
}

func (m *Manager) GetConnection(instanceID uuid.UUID) (*Connection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, ok := m.connections[instanceID]
	return conn, ok
}

func (m *Manager) SendTextMessage(ctx context.Context, instanceID uuid.UUID, to string, text string) (string, error) {
	m.mu.RLock()
	conn, exists := m.connections[instanceID]
	m.mu.RUnlock()

	if !exists || !conn.IsConnected() {
		return "", fmt.Errorf("instance not connected")
	}

	msgID, err := conn.SendTextMessage(ctx, to, text)
	if err != nil {
		return "", err
	}

	return string(msgID), nil
}

func (m *Manager) DisconnectAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, conn := range m.connections {
		conn.Disconnect()
		delete(m.connections, id)
	}
}

func (m *Manager) RestoreAll(ctx context.Context) error {
	mappings, err := m.deviceMapper.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get device mappings: %w", err)
	}

	restored := 0
	for _, mapping := range mappings {
		inst, findErr := m.instanceRepo.FindByID(ctx, mapping.InstanceID)
		if findErr != nil || inst == nil {
			m.log.Warn().Str("instance_id", mapping.InstanceID.String()).Msg("mapped instance not found, skipping")
			continue
		}

		// Only restore instances that were previously available
		if inst.Status == domain.StatusBanned || inst.Status == domain.StatusDisconnected {
			continue
		}

		m.log.Info().
			Str("instance_id", mapping.InstanceID.String()).
			Str("jid", mapping.JID).
			Msg("restoring instance session")

		if _, startErr := m.StartInstance(ctx, mapping.InstanceID); startErr != nil {
			m.log.Error().Err(startErr).
				Str("instance_id", mapping.InstanceID.String()).
				Msg("failed to restore instance")
			continue
		}
		restored++
	}

	m.log.Info().Int("restored", restored).Int("total", len(mappings)).Msg("instance restore complete")
	return nil
}

func (m *Manager) getOrCreateDevice(ctx context.Context, instanceID uuid.UUID) (*store.Device, error) {
	// Check if this instance already has a mapped device JID
	jidStr, err := m.deviceMapper.GetJID(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device mapping: %w", err)
	}

	if jidStr != "" {
		// Try to load existing device by JID
		jid, parseErr := types.ParseJID(jidStr)
		if parseErr == nil {
			device, devErr := m.container.GetDevice(ctx, jid)
			if devErr == nil && device != nil {
				return device, nil
			}
		}
	}

	// No existing device, create a new one
	return m.container.NewDevice(), nil
}

func (m *Manager) handleEvent(instanceID uuid.UUID, evt any) {
	ctx := context.Background()

	switch v := evt.(type) {
	case *events.Connected:
		if err := m.instanceRepo.UpdateStatus(ctx, instanceID, domain.StatusAvailable); err != nil {
			m.log.Error().Err(err).Msg("failed to update instance status to available")
		}

		m.mu.RLock()
		conn := m.connections[instanceID]
		m.mu.RUnlock()

		if conn != nil && conn.Phone != "" {
			if err := m.instanceRepo.UpdatePhone(ctx, instanceID, conn.Phone); err != nil {
				m.log.Error().Err(err).Msg("failed to update instance phone")
			}
		}

		// Persist device-to-instance mapping for restore on restart
		if conn != nil && conn.Client.Store.ID != nil {
			jid := conn.Client.Store.ID.String()
			if err := m.deviceMapper.Upsert(ctx, instanceID, jid); err != nil {
				m.log.Error().Err(err).Msg("failed to persist device mapping")
			}
		}

		m.emitWebhook(ctx, instanceID, "instance.connected", map[string]any{
			"instance_id": instanceID.String(),
		})

	case *events.Disconnected:
		if err := m.instanceRepo.UpdateStatus(ctx, instanceID, domain.StatusDisconnected); err != nil {
			m.log.Error().Err(err).Msg("failed to update instance status to disconnected")
		}

		m.emitWebhook(ctx, instanceID, "instance.disconnected", map[string]any{
			"instance_id": instanceID.String(),
		})

	case *events.LoggedOut:
		if err := m.instanceRepo.UpdateStatus(ctx, instanceID, domain.StatusBanned); err != nil {
			m.log.Error().Err(err).Msg("failed to update instance status to banned")
		}

		m.emitWebhook(ctx, instanceID, "instance.banned", map[string]any{
			"instance_id": instanceID.String(),
			"reason":      "logged_out",
		})

	case *events.Message:
		m.emitWebhook(ctx, instanceID, "message.received", map[string]any{
			"message_id": v.Info.ID,
			"from":       v.Info.Sender.String(),
			"chat_id":    v.Info.Chat.String(),
			"type":       v.Info.Type,
			"timestamp":  v.Info.Timestamp.Unix(),
			"is_group":   v.Info.IsGroup,
		})

	case *events.Receipt:
		eventName := "message.delivered"
		if v.Type == events.ReceiptTypeRead {
			eventName = "message.read"
		}
		m.emitWebhook(ctx, instanceID, eventName, map[string]any{
			"message_ids": v.MessageIDs,
			"from":        v.MessageSource.Sender.String(),
			"timestamp":   v.Timestamp.Unix(),
		})
	}
}

func (m *Manager) emitWebhook(ctx context.Context, instanceID uuid.UUID, eventName string, data any) {
	m.mu.RLock()
	conn, exists := m.connections[instanceID]
	m.mu.RUnlock()

	if !exists {
		return
	}

	group, err := m.groupRepo.FindByID(ctx, conn.GroupID)
	if err != nil || group == nil {
		return
	}

	if group.WebhookURL == nil || group.WebhookSecret == nil {
		return
	}

	m.webhookEmit.Emit(*group.WebhookURL, *group.WebhookSecret, webhook.Event{
		Name:       eventName,
		GroupID:    conn.GroupID.String(),
		InstanceID: instanceID.String(),
		Phone:      conn.Phone,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Data:       data,
	})
}
