package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guilhermecosta/wpp-gateway/internal/domain"
)

type MessageRepo struct {
	pool *pgxpool.Pool
}

func NewMessageRepo(pool *pgxpool.Pool) *MessageRepo {
	return &MessageRepo{pool: pool}
}

const msgColumns = `id, group_id, instance_id, recipient, message_type, content_hash,
	status, error_code, queued_at, sent_at, delivered_at, read_at`

func scanMessage(row pgx.Row) (*domain.MessageLog, error) {
	var m domain.MessageLog
	err := row.Scan(
		&m.ID, &m.GroupID, &m.InstanceID, &m.Recipient, &m.MessageType, &m.ContentHash,
		&m.Status, &m.ErrorCode, &m.QueuedAt, &m.SentAt, &m.DeliveredAt, &m.ReadAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MessageRepo) Create(ctx context.Context, msg *domain.MessageLog) error {
	query := `
		INSERT INTO message_logs (group_id, instance_id, recipient, message_type, content_hash, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, queued_at`

	return r.pool.QueryRow(ctx, query,
		msg.GroupID, msg.InstanceID, msg.Recipient, msg.MessageType, msg.ContentHash, msg.Status,
	).Scan(&msg.ID, &msg.QueuedAt)
}

func (r *MessageRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.MessageLog, error) {
	query := `SELECT ` + msgColumns + ` FROM message_logs WHERE id = $1`
	return scanMessage(r.pool.QueryRow(ctx, query, id))
}

func (r *MessageRepo) FindByGroupID(ctx context.Context, groupID uuid.UUID, filter domain.MessageFilter) ([]domain.MessageLog, int64, error) {
	where := `WHERE group_id = $1`
	args := []any{groupID}
	argIdx := 2

	if filter.Status != nil {
		where += fmt.Sprintf(` AND status = $%d`, argIdx)
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.Recipient != nil {
		where += fmt.Sprintf(` AND recipient = $%d`, argIdx)
		args = append(args, *filter.Recipient)
		argIdx++
	}
	if filter.InstanceID != nil {
		where += fmt.Sprintf(` AND instance_id = $%d`, argIdx)
		args = append(args, *filter.InstanceID)
		argIdx++
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM message_logs ` + where
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}

	query := fmt.Sprintf(`SELECT %s FROM message_logs %s ORDER BY queued_at DESC LIMIT $%d OFFSET $%d`,
		msgColumns, where, argIdx, argIdx+1)
	args = append(args, limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var messages []domain.MessageLog
	for rows.Next() {
		var m domain.MessageLog
		if err := rows.Scan(
			&m.ID, &m.GroupID, &m.InstanceID, &m.Recipient, &m.MessageType, &m.ContentHash,
			&m.Status, &m.ErrorCode, &m.QueuedAt, &m.SentAt, &m.DeliveredAt, &m.ReadAt,
		); err != nil {
			return nil, 0, err
		}
		messages = append(messages, m)
	}
	return messages, total, rows.Err()
}

func (r *MessageRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.MessageStatus) error {
	_, err := r.pool.Exec(ctx, `UPDATE message_logs SET status = $2 WHERE id = $1`, id, status)
	return err
}

func (r *MessageRepo) UpdateSent(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE message_logs SET status = 'sent', sent_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *MessageRepo) UpdateDelivered(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE message_logs SET status = 'delivered', delivered_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *MessageRepo) UpdateRead(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE message_logs SET status = 'read', read_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *MessageRepo) UpdateFailed(ctx context.Context, id uuid.UUID, errorCode string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE message_logs SET status = 'failed', error_code = $2 WHERE id = $1`, id, errorCode)
	return err
}
