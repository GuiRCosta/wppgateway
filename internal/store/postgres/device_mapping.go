package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guilhermecosta/wpp-gateway/internal/instance"
)

type DeviceMappingRepo struct {
	pool *pgxpool.Pool
}

func NewDeviceMappingRepo(pool *pgxpool.Pool) *DeviceMappingRepo {
	return &DeviceMappingRepo{pool: pool}
}

func (r *DeviceMappingRepo) GetJID(ctx context.Context, instanceID uuid.UUID) (string, error) {
	var jid string
	err := r.pool.QueryRow(ctx,
		`SELECT jid FROM device_mapping WHERE instance_id = $1`, instanceID).Scan(&jid)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return jid, err
}

func (r *DeviceMappingRepo) GetInstanceID(ctx context.Context, jid string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx,
		`SELECT instance_id FROM device_mapping WHERE jid = $1`, jid).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, nil
	}
	return id, err
}

func (r *DeviceMappingRepo) Upsert(ctx context.Context, instanceID uuid.UUID, jid string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO device_mapping (instance_id, jid)
		VALUES ($1, $2)
		ON CONFLICT (instance_id) DO UPDATE SET jid = EXCLUDED.jid`,
		instanceID, jid)
	return err
}

func (r *DeviceMappingRepo) Delete(ctx context.Context, instanceID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM device_mapping WHERE instance_id = $1`, instanceID)
	return err
}

func (r *DeviceMappingRepo) GetAll(ctx context.Context) ([]instance.DeviceMappingEntry, error) {
	rows, err := r.pool.Query(ctx, `SELECT instance_id, jid FROM device_mapping`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mappings []instance.DeviceMappingEntry
	for rows.Next() {
		var m instance.DeviceMappingEntry
		if err := rows.Scan(&m.InstanceID, &m.JID); err != nil {
			return nil, err
		}
		mappings = append(mappings, m)
	}
	return mappings, rows.Err()
}
