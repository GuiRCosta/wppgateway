package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BlacklistEntry struct {
	ID        uuid.UUID `json:"id"`
	GroupID   uuid.UUID `json:"group_id"`
	Phone     string    `json:"phone"`
	Reason    string    `json:"reason"`
	CreatedAt string    `json:"created_at"`
}

type BlacklistRepo struct {
	pool *pgxpool.Pool
}

func NewBlacklistRepo(pool *pgxpool.Pool) *BlacklistRepo {
	return &BlacklistRepo{pool: pool}
}

func (r *BlacklistRepo) IsBlacklisted(ctx context.Context, groupID uuid.UUID, phone string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM blacklist WHERE group_id = $1 AND phone = $2)`,
		groupID, phone).Scan(&exists)
	return exists, err
}

func (r *BlacklistRepo) Add(ctx context.Context, groupID uuid.UUID, phones []string, reason string) (int, error) {
	added := 0
	for _, phone := range phones {
		tag, err := r.pool.Exec(ctx,
			`INSERT INTO blacklist (group_id, phone, reason) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
			groupID, phone, reason)
		if err != nil {
			return added, err
		}
		added += int(tag.RowsAffected())
	}
	return added, nil
}

func (r *BlacklistRepo) Remove(ctx context.Context, groupID uuid.UUID, phone string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM blacklist WHERE group_id = $1 AND phone = $2`,
		groupID, phone)
	return err
}

func (r *BlacklistRepo) List(ctx context.Context, groupID uuid.UUID, limit, offset int) ([]BlacklistEntry, int64, error) {
	var total int64
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM blacklist WHERE group_id = $1`, groupID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, group_id, phone, reason, created_at FROM blacklist WHERE group_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		groupID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []BlacklistEntry
	for rows.Next() {
		var e BlacklistEntry
		if err := rows.Scan(&e.ID, &e.GroupID, &e.Phone, &e.Reason, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}
