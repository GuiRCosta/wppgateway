package domain

import (
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	APIKey       string    `json:"-"`
	Plan         string    `json:"plan"`
	MaxGroups    int       `json:"max_groups"`
	MaxInstances int       `json:"max_instances"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreateTenantInput struct {
	Name string `json:"name" validate:"required,min=2,max=255"`
}

type UpdateTenantInput struct {
	Name *string `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
}
