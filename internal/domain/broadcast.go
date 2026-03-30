package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type BroadcastStatus string

const (
	BcastPending    BroadcastStatus = "pending"
	BcastProcessing BroadcastStatus = "processing"
	BcastPaused     BroadcastStatus = "paused"
	BcastCompleted  BroadcastStatus = "completed"
	BcastCancelled  BroadcastStatus = "cancelled"
	BcastFailed     BroadcastStatus = "failed"
)

type Broadcast struct {
	ID          uuid.UUID       `json:"id"`
	GroupID     uuid.UUID       `json:"group_id"`
	Status      BroadcastStatus `json:"status"`
	Total       int             `json:"total"`
	Sent        int             `json:"sent"`
	Delivered   int             `json:"delivered"`
	Read        int             `json:"read"`
	Failed      int             `json:"failed"`
	MessageType string          `json:"message_type"`
	Content     json.RawMessage `json:"content"`
	Variables   json.RawMessage `json:"variables,omitempty"`
	Options     json.RawMessage `json:"options,omitempty"`
	ScheduleAt  *time.Time      `json:"schedule_at,omitempty"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

type BroadcastRecipient struct {
	ID          uuid.UUID  `json:"id"`
	BroadcastID uuid.UUID  `json:"broadcast_id"`
	Recipient   string     `json:"recipient"`
	InstanceID  *uuid.UUID `json:"instance_id,omitempty"`
	Status      string     `json:"status"`
	ErrorCode   *string    `json:"error_code,omitempty"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
}

type CreateBroadcastInput struct {
	Recipients []string        `json:"recipients" validate:"required,min=1"`
	Type       string          `json:"type" validate:"required"`
	Content    json.RawMessage `json:"content" validate:"required"`
	Variables  json.RawMessage `json:"variables,omitempty"`
	Options    BroadcastOptions `json:"options,omitempty"`
}

type BroadcastOptions struct {
	ShuffleRecipients    bool       `json:"shuffle_recipients"`
	VaryContent          bool       `json:"vary_content"`
	Spintax              bool       `json:"spintax"`
	ScheduleAt           *time.Time `json:"schedule_at,omitempty"`
	RespectOperatingHours bool      `json:"respect_operating_hours"`
	SkipInvalidNumbers   bool       `json:"skip_invalid_numbers"`
	SkipNonWhatsapp      bool       `json:"skip_non_whatsapp"`
}

type BroadcastProgress struct {
	BroadcastID         uuid.UUID       `json:"broadcast_id"`
	Status              BroadcastStatus `json:"status"`
	Total               int             `json:"total"`
	Sent                int             `json:"sent"`
	Delivered           int             `json:"delivered"`
	Read                int             `json:"read"`
	Failed              int             `json:"failed"`
	Pending             int             `json:"pending"`
	EstimatedCompletion *time.Time      `json:"estimated_completion,omitempty"`
}

type BroadcastRepository interface {
	Create(ctx context.Context, groupID uuid.UUID, input CreateBroadcastInput) (*Broadcast, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Broadcast, error)
	FindByGroupID(ctx context.Context, groupID uuid.UUID, limit, offset int) ([]Broadcast, int64, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status BroadcastStatus) error
	UpdateProgress(ctx context.Context, id uuid.UUID, sent, delivered, failed int) error
	MarkStarted(ctx context.Context, id uuid.UUID) error
	MarkCompleted(ctx context.Context, id uuid.UUID) error

	CreateRecipients(ctx context.Context, broadcastID uuid.UUID, recipients []string) error
	GetPendingRecipients(ctx context.Context, broadcastID uuid.UUID, limit int) ([]BroadcastRecipient, error)
	UpdateRecipientSent(ctx context.Context, id uuid.UUID, instanceID uuid.UUID) error
	UpdateRecipientFailed(ctx context.Context, id uuid.UUID, errorCode string) error
	GetProgress(ctx context.Context, broadcastID uuid.UUID) (*BroadcastProgress, error)
}
