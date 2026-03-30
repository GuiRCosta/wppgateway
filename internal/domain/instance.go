package domain

import (
	"time"

	"github.com/google/uuid"
)

type InstanceStatus string

const (
	StatusDisconnected InstanceStatus = "disconnected"
	StatusConnecting   InstanceStatus = "connecting"
	StatusAvailable    InstanceStatus = "available"
	StatusResting      InstanceStatus = "resting"
	StatusWarming      InstanceStatus = "warming"
	StatusSuspect      InstanceStatus = "suspect"
	StatusBanned       InstanceStatus = "banned"
)

type Instance struct {
	ID            uuid.UUID      `json:"id"`
	GroupID       uuid.UUID      `json:"group_id"`
	PhoneNumber   *string        `json:"phone_number,omitempty"`
	DisplayName   *string        `json:"display_name,omitempty"`
	Status        InstanceStatus `json:"status"`
	Priority      int            `json:"priority"`
	DailyBudget   int            `json:"daily_budget"`
	HourlyBudget  int            `json:"hourly_budget"`
	WarmupDays    int            `json:"warmup_days"`
	MessagesToday int            `json:"messages_today"`
	MessagesHour  int            `json:"messages_hour"`
	DeliveryRate  float64        `json:"delivery_rate"`
	LastActiveAt  *time.Time     `json:"last_active_at,omitempty"`
	BannedAt      *time.Time     `json:"banned_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

type CreateInstanceInput struct {
	DisplayName  *string `json:"display_name,omitempty" validate:"omitempty,max=255"`
	Priority     int     `json:"priority"`
	DailyBudget  int     `json:"daily_budget" validate:"min=1"`
	HourlyBudget int     `json:"hourly_budget" validate:"min=1"`
}

type UpdateInstanceInput struct {
	DisplayName  *string `json:"display_name,omitempty" validate:"omitempty,max=255"`
	Priority     *int    `json:"priority,omitempty"`
	DailyBudget  *int    `json:"daily_budget,omitempty" validate:"omitempty,min=1"`
	HourlyBudget *int    `json:"hourly_budget,omitempty" validate:"omitempty,min=1"`
}
