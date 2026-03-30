package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepo struct {
	pool *pgxpool.Pool
}

func NewSessionRepo(pool *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{pool: pool}
}

func (r *SessionRepo) Find(ctx context.Context, instanceID uuid.UUID) ([]byte, []byte, error) {
	query := `SELECT creds_encrypted, iv FROM session_credentials WHERE instance_id = $1`

	var creds, iv []byte
	err := r.pool.QueryRow(ctx, query, instanceID).Scan(&creds, &iv)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	return creds, iv, nil
}

func (r *SessionRepo) Upsert(ctx context.Context, instanceID uuid.UUID, creds []byte, iv []byte) error {
	query := `
		INSERT INTO session_credentials (instance_id, creds_encrypted, iv)
		VALUES ($1, $2, $3)
		ON CONFLICT (instance_id) DO UPDATE SET
			creds_encrypted = EXCLUDED.creds_encrypted,
			iv = EXCLUDED.iv,
			updated_at = NOW()`

	_, err := r.pool.Exec(ctx, query, instanceID, creds, iv)
	return err
}

func (r *SessionRepo) Delete(ctx context.Context, instanceID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM session_credentials WHERE instance_id = $1`, instanceID)
	return err
}
