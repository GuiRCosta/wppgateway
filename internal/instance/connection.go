package instance

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type ConnectionStatus string

const (
	ConnStatusDisconnected ConnectionStatus = "disconnected"
	ConnStatusConnecting   ConnectionStatus = "connecting"
	ConnStatusConnected    ConnectionStatus = "connected"
)

type Connection struct {
	ID         uuid.UUID
	GroupID    uuid.UUID
	Client     *whatsmeow.Client
	Device     *store.Device
	Status     ConnectionStatus
	Phone      string
	mu         sync.RWMutex
	log        zerolog.Logger
	onEvent    func(instanceID uuid.UUID, evt any)
}

func NewConnection(id, groupID uuid.UUID, device *store.Device, log zerolog.Logger, onEvent func(uuid.UUID, any)) *Connection {
	waLogger := waLog.Stdout("whatsmeow", "INFO", true)

	client := whatsmeow.NewClient(device, waLogger)

	conn := &Connection{
		ID:      id,
		GroupID: groupID,
		Client:  client,
		Device:  device,
		Status:  ConnStatusDisconnected,
		log:     log.With().Str("instance_id", id.String()).Logger(),
		onEvent: onEvent,
	}

	client.AddEventHandler(conn.handleEvent)

	return conn
}

func (c *Connection) Connect(ctx context.Context) (<-chan whatsmeow.QRChannelItem, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Status = ConnStatusConnecting

	if c.Client.Store.ID == nil {
		qrChan, err := c.Client.GetQRChannel(ctx)
		if err != nil {
			c.Status = ConnStatusDisconnected
			return nil, fmt.Errorf("failed to get QR channel: %w", err)
		}
		if err := c.Client.Connect(); err != nil {
			c.Status = ConnStatusDisconnected
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
		return qrChan, nil
	}

	if err := c.Client.Connect(); err != nil {
		c.Status = ConnStatusDisconnected
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	c.Status = ConnStatusConnected
	return nil, nil
}

func (c *Connection) PairPhone(ctx context.Context, phone string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Status = ConnStatusConnecting

	if err := c.Client.Connect(); err != nil {
		c.Status = ConnStatusDisconnected
		return "", fmt.Errorf("failed to connect: %w", err)
	}

	code, err := c.Client.PairPhone(ctx, phone, true, whatsmeow.PairClientChrome, "WPP Gateway")
	if err != nil {
		c.Status = ConnStatusDisconnected
		return "", fmt.Errorf("failed to pair phone: %w", err)
	}

	return code, nil
}

func (c *Connection) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Client.Disconnect()
	c.Status = ConnStatusDisconnected
}

func (c *Connection) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Client.IsConnected()
}

func (c *Connection) GetStatus() ConnectionStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Status
}

func (c *Connection) SendTextMessage(ctx context.Context, to string, text string) (types.MessageID, error) {
	jid, err := types.ParseJID(to + "@s.whatsapp.net")
	if err != nil {
		return "", fmt.Errorf("invalid JID: %w", err)
	}

	msg := &waE2E.Message{
		Conversation: proto.String(text),
	}

	resp, err := c.Client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	return resp.ID, nil
}

func (c *Connection) handleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Connected:
		c.mu.Lock()
		c.Status = ConnStatusConnected
		if c.Client.Store.ID != nil {
			c.Phone = c.Client.Store.ID.User
		}
		c.mu.Unlock()
		c.log.Info().Str("phone", c.Phone).Msg("WhatsApp connected")

	case *events.Disconnected:
		c.mu.Lock()
		c.Status = ConnStatusDisconnected
		c.mu.Unlock()
		c.log.Warn().Msg("WhatsApp disconnected")

	case *events.LoggedOut:
		c.mu.Lock()
		c.Status = ConnStatusDisconnected
		c.mu.Unlock()
		c.log.Warn().Msg("WhatsApp logged out")

	case *events.Message:
		c.log.Debug().
			Str("from", v.Info.Sender.String()).
			Str("type", v.Info.Type).
			Msg("message received")
	}

	if c.onEvent != nil {
		c.onEvent(c.ID, evt)
	}
}
