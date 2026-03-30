package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guilhermecosta/wpp-gateway/internal/domain"
)

type BroadcastRepo struct {
	pool *pgxpool.Pool
}

func NewBroadcastRepo(pool *pgxpool.Pool) *BroadcastRepo {
	return &BroadcastRepo{pool: pool}
}

func (r *BroadcastRepo) Create(ctx context.Context, groupID uuid.UUID, input domain.CreateBroadcastInput) (*domain.Broadcast, error) {
	opts, _ := json.Marshal(input.Options)
	vars := input.Variables
	if vars == nil {
		vars = []byte("{}")
	}

	query := `
		INSERT INTO broadcasts (group_id, total, message_type, content, variables, options, schedule_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, group_id, status, total, sent, delivered, read, failed,
		          message_type, content, variables, options, schedule_at, started_at, completed_at, created_at`

	var b domain.Broadcast
	err := r.pool.QueryRow(ctx, query,
		groupID, len(input.Recipients), input.Type, input.Content, vars, opts, input.Options.ScheduleAt,
	).Scan(
		&b.ID, &b.GroupID, &b.Status, &b.Total, &b.Sent, &b.Delivered, &b.Read, &b.Failed,
		&b.MessageType, &b.Content, &b.Variables, &b.Options, &b.ScheduleAt, &b.StartedAt, &b.CompletedAt, &b.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *BroadcastRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Broadcast, error) {
	query := `
		SELECT id, group_id, status, total, sent, delivered, read, failed,
		       message_type, content, variables, options, schedule_at, started_at, completed_at, created_at
		FROM broadcasts WHERE id = $1`

	var b domain.Broadcast
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&b.ID, &b.GroupID, &b.Status, &b.Total, &b.Sent, &b.Delivered, &b.Read, &b.Failed,
		&b.MessageType, &b.Content, &b.Variables, &b.Options, &b.ScheduleAt, &b.StartedAt, &b.CompletedAt, &b.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *BroadcastRepo) FindByGroupID(ctx context.Context, groupID uuid.UUID, limit, offset int) ([]domain.Broadcast, int64, error) {
	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM broadcasts WHERE group_id = $1`, groupID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, group_id, status, total, sent, delivered, read, failed,
		       message_type, content, variables, options, schedule_at, started_at, completed_at, created_at
		FROM broadcasts WHERE group_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, groupID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var broadcasts []domain.Broadcast
	for rows.Next() {
		var b domain.Broadcast
		if err := rows.Scan(
			&b.ID, &b.GroupID, &b.Status, &b.Total, &b.Sent, &b.Delivered, &b.Read, &b.Failed,
			&b.MessageType, &b.Content, &b.Variables, &b.Options, &b.ScheduleAt, &b.StartedAt, &b.CompletedAt, &b.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		broadcasts = append(broadcasts, b)
	}
	return broadcasts, total, rows.Err()
}

func (r *BroadcastRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.BroadcastStatus) error {
	_, err := r.pool.Exec(ctx, `UPDATE broadcasts SET status = $2 WHERE id = $1`, id, status)
	return err
}

func (r *BroadcastRepo) UpdateProgress(ctx context.Context, id uuid.UUID, sent, delivered, failed int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE broadcasts SET sent = $2, delivered = $3, failed = $4 WHERE id = $1`,
		id, sent, delivered, failed)
	return err
}

func (r *BroadcastRepo) MarkStarted(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE broadcasts SET status = 'processing', started_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *BroadcastRepo) MarkCompleted(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE broadcasts SET status = 'completed', completed_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *BroadcastRepo) CreateRecipients(ctx context.Context, broadcastID uuid.UUID, recipients []string) error {
	batch := &pgx.Batch{}
	for _, recipient := range recipients {
		batch.Queue(`INSERT INTO broadcast_recipients (broadcast_id, recipient) VALUES ($1, $2)`,
			broadcastID, recipient)
	}
	results := r.pool.SendBatch(ctx, batch)
	defer results.Close()

	for range recipients {
		if _, err := results.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (r *BroadcastRepo) GetPendingRecipients(ctx context.Context, broadcastID uuid.UUID, limit int) ([]domain.BroadcastRecipient, error) {
	query := `
		SELECT id, broadcast_id, recipient, instance_id, status, error_code, sent_at, delivered_at
		FROM broadcast_recipients
		WHERE broadcast_id = $1 AND status = 'pending'
		ORDER BY id LIMIT $2`

	rows, err := r.pool.Query(ctx, query, broadcastID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipients []domain.BroadcastRecipient
	for rows.Next() {
		var br domain.BroadcastRecipient
		if err := rows.Scan(&br.ID, &br.BroadcastID, &br.Recipient, &br.InstanceID,
			&br.Status, &br.ErrorCode, &br.SentAt, &br.DeliveredAt); err != nil {
			return nil, err
		}
		recipients = append(recipients, br)
	}
	return recipients, rows.Err()
}

func (r *BroadcastRepo) UpdateRecipientSent(ctx context.Context, id uuid.UUID, instanceID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE broadcast_recipients SET status = 'sent', instance_id = $2, sent_at = NOW() WHERE id = $1`,
		id, instanceID)
	return err
}

func (r *BroadcastRepo) UpdateRecipientFailed(ctx context.Context, id uuid.UUID, errorCode string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE broadcast_recipients SET status = 'failed', error_code = $2 WHERE id = $1`,
		id, errorCode)
	return err
}

func (r *BroadcastRepo) GetProgress(ctx context.Context, broadcastID uuid.UUID) (*domain.BroadcastProgress, error) {
	b, err := r.FindByID(ctx, broadcastID)
	if err != nil || b == nil {
		return nil, err
	}

	var pending int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM broadcast_recipients WHERE broadcast_id = $1 AND status = 'pending'`,
		broadcastID).Scan(&pending); err != nil {
		return nil, err
	}

	return &domain.BroadcastProgress{
		BroadcastID: broadcastID,
		Status:      b.Status,
		Total:       b.Total,
		Sent:        b.Sent,
		Delivered:   b.Delivered,
		Read:        b.Read,
		Failed:      b.Failed,
		Pending:     pending,
	}, nil
}
