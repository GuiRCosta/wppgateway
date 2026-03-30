package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guilhermecosta/wpp-gateway/internal/domain"
)

type InstanceRepo struct {
	pool *pgxpool.Pool
}

func NewInstanceRepo(pool *pgxpool.Pool) *InstanceRepo {
	return &InstanceRepo{pool: pool}
}

const instanceColumns = `id, group_id, phone_number, display_name, status, priority,
	daily_budget, hourly_budget, warmup_days, messages_today, messages_hour,
	delivery_rate, last_active_at, banned_at, created_at`

func scanInstance(row pgx.Row) (*domain.Instance, error) {
	var i domain.Instance
	err := row.Scan(
		&i.ID, &i.GroupID, &i.PhoneNumber, &i.DisplayName, &i.Status, &i.Priority,
		&i.DailyBudget, &i.HourlyBudget, &i.WarmupDays, &i.MessagesToday, &i.MessagesHour,
		&i.DeliveryRate, &i.LastActiveAt, &i.BannedAt, &i.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func (r *InstanceRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Instance, error) {
	query := `SELECT ` + instanceColumns + ` FROM instances WHERE id = $1`
	return scanInstance(r.pool.QueryRow(ctx, query, id))
}

func (r *InstanceRepo) FindByGroupID(ctx context.Context, groupID uuid.UUID) ([]domain.Instance, error) {
	query := `SELECT ` + instanceColumns + ` FROM instances WHERE group_id = $1 ORDER BY priority DESC, created_at`
	rows, err := r.pool.Query(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instances []domain.Instance
	for rows.Next() {
		var i domain.Instance
		if err := rows.Scan(
			&i.ID, &i.GroupID, &i.PhoneNumber, &i.DisplayName, &i.Status, &i.Priority,
			&i.DailyBudget, &i.HourlyBudget, &i.WarmupDays, &i.MessagesToday, &i.MessagesHour,
			&i.DeliveryRate, &i.LastActiveAt, &i.BannedAt, &i.CreatedAt,
		); err != nil {
			return nil, err
		}
		instances = append(instances, i)
	}
	return instances, rows.Err()
}

func (r *InstanceRepo) FindAvailableByGroupID(ctx context.Context, groupID uuid.UUID) ([]domain.Instance, error) {
	query := `SELECT ` + instanceColumns + ` FROM instances WHERE group_id = $1 AND status = 'available' ORDER BY priority DESC`
	rows, err := r.pool.Query(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instances []domain.Instance
	for rows.Next() {
		var i domain.Instance
		if err := rows.Scan(
			&i.ID, &i.GroupID, &i.PhoneNumber, &i.DisplayName, &i.Status, &i.Priority,
			&i.DailyBudget, &i.HourlyBudget, &i.WarmupDays, &i.MessagesToday, &i.MessagesHour,
			&i.DeliveryRate, &i.LastActiveAt, &i.BannedAt, &i.CreatedAt,
		); err != nil {
			return nil, err
		}
		instances = append(instances, i)
	}
	return instances, rows.Err()
}

func (r *InstanceRepo) Create(ctx context.Context, groupID uuid.UUID, input domain.CreateInstanceInput) (*domain.Instance, error) {
	query := `
		INSERT INTO instances (group_id, display_name, priority, daily_budget, hourly_budget)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + instanceColumns

	return scanInstance(r.pool.QueryRow(ctx, query,
		groupID, input.DisplayName, input.Priority, input.DailyBudget, input.HourlyBudget,
	))
}

func (r *InstanceRepo) Update(ctx context.Context, id uuid.UUID, input domain.UpdateInstanceInput) (*domain.Instance, error) {
	query := `
		UPDATE instances SET
			display_name = COALESCE($2, display_name),
			priority = COALESCE($3, priority),
			daily_budget = COALESCE($4, daily_budget),
			hourly_budget = COALESCE($5, hourly_budget)
		WHERE id = $1
		RETURNING ` + instanceColumns

	return scanInstance(r.pool.QueryRow(ctx, query,
		id, input.DisplayName, input.Priority, input.DailyBudget, input.HourlyBudget,
	))
}

func (r *InstanceRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.InstanceStatus) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE instances SET status = $2 WHERE id = $1`, id, status)
	return err
}

func (r *InstanceRepo) UpdatePhone(ctx context.Context, id uuid.UUID, phone string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE instances SET phone_number = $2 WHERE id = $1`, id, phone)
	return err
}

func (r *InstanceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM instances WHERE id = $1`, id)
	return err
}

func (r *InstanceRepo) CountByGroupID(ctx context.Context, groupID uuid.UUID) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM instances WHERE group_id = $1`, groupID).Scan(&count)
	return count, err
}

func (r *InstanceRepo) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM instances i
		JOIN instance_groups g ON i.group_id = g.id
		WHERE g.tenant_id = $1`, tenantID).Scan(&count)
	return count, err
}
