package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Strategy string

const (
	StrategyFailover Strategy = "failover"
	StrategyRotation Strategy = "rotation"
	StrategyHybrid   Strategy = "hybrid"
)

type Group struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	Name          string          `json:"name"`
	Strategy      Strategy        `json:"strategy"`
	Config        json.RawMessage `json:"config"`
	WebhookURL    *string         `json:"webhook_url,omitempty"`
	WebhookSecret *string         `json:"-"`
	WebhookEvents []string        `json:"webhook_events"`
	IsActive      bool            `json:"is_active"`
	CreatedAt     time.Time       `json:"created_at"`
}

type CreateGroupInput struct {
	Name          string          `json:"name" validate:"required,min=2,max=255"`
	Strategy      Strategy        `json:"strategy" validate:"required,oneof=failover rotation hybrid"`
	Config        json.RawMessage `json:"config,omitempty"`
	WebhookURL    *string         `json:"webhook_url,omitempty" validate:"omitempty,url"`
	WebhookEvents []string        `json:"webhook_events,omitempty"`
}

type UpdateGroupInput struct {
	Name          *string          `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	Strategy      *Strategy        `json:"strategy,omitempty" validate:"omitempty,oneof=failover rotation hybrid"`
	Config        *json.RawMessage `json:"config,omitempty"`
	WebhookURL    *string          `json:"webhook_url,omitempty" validate:"omitempty,url"`
	WebhookEvents *[]string        `json:"webhook_events,omitempty"`
}
