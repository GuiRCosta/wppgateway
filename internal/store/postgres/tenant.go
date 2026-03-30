package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guilhermecosta/wpp-gateway/internal/domain"
)

func hashAPIKey(apiKey string) string {
	h := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(h[:])
}

type TenantRepo struct {
	pool *pgxpool.Pool
}

func NewTenantRepo(pool *pgxpool.Pool) *TenantRepo {
	return &TenantRepo{pool: pool}
}

func (r *TenantRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	query := `
		SELECT id, name, api_key_hash, plan, max_groups, max_instances, is_active, created_at
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
	hash := hashAPIKey(apiKey)
	query := `
		SELECT id, name, api_key_hash, plan, max_groups, max_instances, is_active, created_at
		FROM tenants WHERE api_key_hash = $1`

	var t domain.Tenant
	err := r.pool.QueryRow(ctx, query, hash).Scan(
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
	hash := hashAPIKey(apiKey)
	query := `
		INSERT INTO tenants (name, api_key_hash)
		VALUES ($1, $2)
		RETURNING id, name, api_key_hash, plan, max_groups, max_instances, is_active, created_at`

	var t domain.Tenant
	err := r.pool.QueryRow(ctx, query, input.Name, hash).Scan(
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
		RETURNING id, name, api_key_hash, plan, max_groups, max_instances, is_active, created_at`

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
