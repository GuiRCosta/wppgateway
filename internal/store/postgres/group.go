package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guilhermecosta/wpp-gateway/internal/domain"
)

type GroupRepo struct {
	pool *pgxpool.Pool
}

func NewGroupRepo(pool *pgxpool.Pool) *GroupRepo {
	return &GroupRepo{pool: pool}
}

func (r *GroupRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Group, error) {
	query := `
		SELECT id, tenant_id, name, strategy, config, webhook_url, webhook_secret, webhook_events, is_active, created_at
		FROM instance_groups WHERE id = $1`

	var g domain.Group
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&g.ID, &g.TenantID, &g.Name, &g.Strategy, &g.Config,
		&g.WebhookURL, &g.WebhookSecret, &g.WebhookEvents, &g.IsActive, &g.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *GroupRepo) FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]domain.Group, error) {
	query := `
		SELECT id, tenant_id, name, strategy, config, webhook_url, webhook_secret, webhook_events, is_active, created_at
		FROM instance_groups WHERE tenant_id = $1 ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []domain.Group
	for rows.Next() {
		var g domain.Group
		if err := rows.Scan(
			&g.ID, &g.TenantID, &g.Name, &g.Strategy, &g.Config,
			&g.WebhookURL, &g.WebhookSecret, &g.WebhookEvents, &g.IsActive, &g.CreatedAt,
		); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (r *GroupRepo) Create(ctx context.Context, tenantID uuid.UUID, input domain.CreateGroupInput) (*domain.Group, error) {
	secret, err := generateWebhookSecret()
	if err != nil {
		return nil, err
	}

	cfg := input.Config
	if cfg == nil {
		cfg = []byte("{}")
	}

	query := `
		INSERT INTO instance_groups (tenant_id, name, strategy, config, webhook_url, webhook_secret, webhook_events)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, tenant_id, name, strategy, config, webhook_url, webhook_secret, webhook_events, is_active, created_at`

	var g domain.Group
	err = r.pool.QueryRow(ctx, query,
		tenantID, input.Name, input.Strategy, cfg,
		input.WebhookURL, secret, input.WebhookEvents,
	).Scan(
		&g.ID, &g.TenantID, &g.Name, &g.Strategy, &g.Config,
		&g.WebhookURL, &g.WebhookSecret, &g.WebhookEvents, &g.IsActive, &g.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *GroupRepo) Update(ctx context.Context, id uuid.UUID, input domain.UpdateGroupInput) (*domain.Group, error) {
	query := `
		UPDATE instance_groups SET
			name = COALESCE($2, name),
			strategy = COALESCE($3, strategy),
			config = COALESCE($4, config),
			webhook_url = COALESCE($5, webhook_url),
			webhook_events = COALESCE($6, webhook_events)
		WHERE id = $1
		RETURNING id, tenant_id, name, strategy, config, webhook_url, webhook_secret, webhook_events, is_active, created_at`

	var g domain.Group
	err := r.pool.QueryRow(ctx, query,
		id, input.Name, input.Strategy, input.Config,
		input.WebhookURL, input.WebhookEvents,
	).Scan(
		&g.ID, &g.TenantID, &g.Name, &g.Strategy, &g.Config,
		&g.WebhookURL, &g.WebhookSecret, &g.WebhookEvents, &g.IsActive, &g.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *GroupRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM instance_groups WHERE id = $1`, id)
	return err
}

func (r *GroupRepo) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM instance_groups WHERE tenant_id = $1`, tenantID).Scan(&count)
	return count, err
}

func generateWebhookSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
