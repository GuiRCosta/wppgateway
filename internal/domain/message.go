package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type MessageStatus string

const (
	MsgStatusQueued    MessageStatus = "queued"
	MsgStatusSent      MessageStatus = "sent"
	MsgStatusDelivered MessageStatus = "delivered"
	MsgStatusRead      MessageStatus = "read"
	MsgStatusFailed    MessageStatus = "failed"
)

type MessageLog struct {
	ID          uuid.UUID     `json:"id"`
	GroupID     *uuid.UUID    `json:"group_id,omitempty"`
	InstanceID  *uuid.UUID    `json:"instance_id,omitempty"`
	Recipient   string        `json:"recipient"`
	MessageType string        `json:"message_type"`
	ContentHash *string       `json:"content_hash,omitempty"`
	Status      MessageStatus `json:"status"`
	ErrorCode   *string       `json:"error_code,omitempty"`
	QueuedAt    time.Time     `json:"queued_at"`
	SentAt      *time.Time    `json:"sent_at,omitempty"`
	DeliveredAt *time.Time    `json:"delivered_at,omitempty"`
	ReadAt      *time.Time    `json:"read_at,omitempty"`
}

type SendMessageInput struct {
	To      string          `json:"to" validate:"required"`
	Type    string          `json:"type" validate:"required,oneof=text image video audio document sticker location contact reaction poll"`
	Content json.RawMessage `json:"content" validate:"required"`
}

type MessageFilter struct {
	Status     *MessageStatus
	Recipient  *string
	InstanceID *uuid.UUID
	Limit      int
	Offset     int
}
