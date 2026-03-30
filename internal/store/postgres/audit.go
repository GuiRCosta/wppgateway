package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditEntry struct {
	ID         uuid.UUID       `json:"id"`
	TenantID   uuid.UUID       `json:"tenant_id"`
	Action     string          `json:"action"`
	ResourceID *uuid.UUID      `json:"resource_id,omitempty"`
	Details    json.RawMessage `json:"details"`
	IPAddress  *string         `json:"ip_address,omitempty"`
	APIKeyID   *string         `json:"api_key_id,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

type AuditRepo struct {
	pool *pgxpool.Pool
}

func NewAuditRepo(pool *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{pool: pool}
}

func (r *AuditRepo) Log(ctx context.Context, tenantID uuid.UUID, action string, resourceID *uuid.UUID, details any, ip string) error {
	detailsJSON, _ := json.Marshal(details)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO audit_log (tenant_id, action, resource_id, details, ip_address) VALUES ($1, $2, $3, $4, $5)`,
		tenantID, action, resourceID, detailsJSON, ip)
	return err
}

func (r *AuditRepo) List(ctx context.Context, tenantID uuid.UUID, action string, limit, offset int) ([]AuditEntry, int64, error) {
	where := `WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2

	if action != "" {
		where += ` AND action = $2`
		args = append(args, action)
		argIdx++
	}

	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_log `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT id, tenant_id, action, resource_id, details, ip_address, api_key_id, created_at
		FROM audit_log ` + where + ` ORDER BY created_at DESC LIMIT $` + itoa(argIdx) + ` OFFSET $` + itoa(argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.TenantID, &e.Action, &e.ResourceID, &e.Details,
			&e.IPAddress, &e.APIKeyID, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}

func itoa(i int) string {
	return string(rune('0' + i))
}
