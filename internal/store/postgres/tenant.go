package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guilhermecosta/wpp-gateway/internal/domain"
)

type TenantRepo struct {
	pool *pgxpool.Pool
}

func NewTenantRepo(pool *pgxpool.Pool) *TenantRepo {
	return &TenantRepo{pool: pool}
}

func (r *TenantRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	query := `
		SELECT id, name, api_key, plan, max_groups, max_instances, is_active, created_at
		FROM tenants WHERE id = $1`

	var t domain.Tenant
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Name, &t.APIKey, &t.Plan,
		&t.MaxGroups, &t.MaxInstances, &t.IsActive, &t.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TenantRepo) FindByAPIKey(ctx context.Context, apiKey string) (*domain.Tenant, error) {
	query := `
		SELECT id, name, api_key, plan, max_groups, max_instances, is_active, created_at
		FROM tenants WHERE api_key = $1`

	var t domain.Tenant
	err := r.pool.QueryRow(ctx, query, apiKey).Scan(
		&t.ID, &t.Name, &t.APIKey, &t.Plan,
		&t.MaxGroups, &t.MaxInstances, &t.IsActive, &t.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TenantRepo) Create(ctx context.Context, input domain.CreateTenantInput, apiKey string) (*domain.Tenant, error) {
	query := `
		INSERT INTO tenants (name, api_key)
		VALUES ($1, $2)
		RETURNING id, name, api_key, plan, max_groups, max_instances, is_active, created_at`

	var t domain.Tenant
	err := r.pool.QueryRow(ctx, query, input.Name, apiKey).Scan(
		&t.ID, &t.Name, &t.APIKey, &t.Plan,
		&t.MaxGroups, &t.MaxInstances, &t.IsActive, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TenantRepo) Update(ctx context.Context, id uuid.UUID, input domain.UpdateTenantInput) (*domain.Tenant, error) {
	query := `
		UPDATE tenants SET
			name = COALESCE($2, name)
		WHERE id = $1
		RETURNING id, name, api_key, plan, max_groups, max_instances, is_active, created_at`

	var t domain.Tenant
	err := r.pool.QueryRow(ctx, query, id, input.Name).Scan(
		&t.ID, &t.Name, &t.APIKey, &t.Plan,
		&t.MaxGroups, &t.MaxInstances, &t.IsActive, &t.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TenantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, id)
	return err
}
