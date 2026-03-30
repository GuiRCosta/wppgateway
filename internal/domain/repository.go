package domain

import (
	"context"

	"github.com/google/uuid"
)

type TenantRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
	FindByAPIKey(ctx context.Context, apiKey string) (*Tenant, error)
	Create(ctx context.Context, input CreateTenantInput, apiKey string) (*Tenant, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateTenantInput) (*Tenant, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type GroupRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Group, error)
	FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]Group, error)
	Create(ctx context.Context, tenantID uuid.UUID, input CreateGroupInput) (*Group, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateGroupInput) (*Group, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error)
}

type InstanceRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Instance, error)
	FindByGroupID(ctx context.Context, groupID uuid.UUID) ([]Instance, error)
	FindAvailableByGroupID(ctx context.Context, groupID uuid.UUID) ([]Instance, error)
	Create(ctx context.Context, groupID uuid.UUID, input CreateInstanceInput) (*Instance, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateInstanceInput) (*Instance, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status InstanceStatus) error
	UpdatePhone(ctx context.Context, id uuid.UUID, phone string) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountByGroupID(ctx context.Context, groupID uuid.UUID) (int64, error)
	CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error)
}

type SessionRepository interface {
	Find(ctx context.Context, instanceID uuid.UUID) (creds []byte, iv []byte, err error)
	Upsert(ctx context.Context, instanceID uuid.UUID, creds []byte, iv []byte) error
	Delete(ctx context.Context, instanceID uuid.UUID) error
}

type MessageRepository interface {
	Create(ctx context.Context, msg *MessageLog) error
	FindByID(ctx context.Context, id uuid.UUID) (*MessageLog, error)
	FindByGroupID(ctx context.Context, groupID uuid.UUID, filter MessageFilter) ([]MessageLog, int64, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status MessageStatus) error
	UpdateSent(ctx context.Context, id uuid.UUID) error
	UpdateDelivered(ctx context.Context, id uuid.UUID) error
	UpdateRead(ctx context.Context, id uuid.UUID) error
	UpdateFailed(ctx context.Context, id uuid.UUID, errorCode string) error
}
