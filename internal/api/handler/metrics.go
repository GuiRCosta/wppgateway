package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guilhermecosta/wpp-gateway/internal/api/middleware"
	"github.com/guilhermecosta/wpp-gateway/pkg/response"
	"github.com/guilhermecosta/wpp-gateway/pkg/validator"
)

type MetricsHandler struct {
	db *pgxpool.Pool
}

func NewMetricsHandler(db *pgxpool.Pool) *MetricsHandler {
	return &MetricsHandler{db: db}
}

func (h *MetricsHandler) GroupMetrics(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	groupID, err := validator.ValidateUUID(c.Params("groupId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	ctx := c.Context()

	var sent, delivered, failed int64
	err = h.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN status IN ('sent','delivered','read') THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN status IN ('delivered','read') THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0)
		FROM message_logs WHERE group_id = $1`, groupID).Scan(&sent, &delivered, &failed)
	if err != nil {
		return response.ErrInternal(c, "Failed to get metrics")
	}

	var deliveryRate float64
	if sent > 0 {
		deliveryRate = float64(delivered) / float64(sent)
	}

	var activeInstances, bannedInstances int64
	_ = h.db.QueryRow(ctx,
		`SELECT COUNT(*) FILTER (WHERE status = 'available'), COUNT(*) FILTER (WHERE status = 'banned')
		 FROM instances WHERE group_id = $1`, groupID).Scan(&activeInstances, &bannedInstances)

	return response.OK(c, fiber.Map{
		"sent":              sent,
		"delivered":         delivered,
		"failed":            failed,
		"delivery_rate":     deliveryRate,
		"active_instances":  activeInstances,
		"banned_instances":  bannedInstances,
	})
}

func (h *MetricsHandler) DailyMetrics(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	groupID, err := validator.ValidateUUID(c.Params("groupId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	from := c.Query("from", time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
	to := c.Query("to", time.Now().Format("2006-01-02"))

	rows, err := h.db.Query(c.Context(), `
		SELECT DATE(queued_at) as date,
		       COUNT(*) FILTER (WHERE status IN ('sent','delivered','read')) as sent,
		       COUNT(*) FILTER (WHERE status IN ('delivered','read')) as delivered,
		       COUNT(*) FILTER (WHERE status = 'read') as read,
		       COUNT(*) FILTER (WHERE status = 'failed') as failed
		FROM message_logs
		WHERE group_id = $1 AND queued_at >= $2::date AND queued_at < ($3::date + interval '1 day')
		GROUP BY DATE(queued_at) ORDER BY date`, groupID, from, to)
	if err != nil {
		return response.ErrInternal(c, "Failed to get daily metrics")
	}
	defer rows.Close()

	type dailyRow struct {
		Date         string  `json:"date"`
		Sent         int     `json:"sent"`
		Delivered    int     `json:"delivered"`
		Read         int     `json:"read"`
		Failed       int     `json:"failed"`
		DeliveryRate float64 `json:"delivery_rate"`
	}

	var daily []dailyRow
	for rows.Next() {
		var r dailyRow
		var d time.Time
		if err := rows.Scan(&d, &r.Sent, &r.Delivered, &r.Read, &r.Failed); err != nil {
			return response.ErrInternal(c, "Failed to scan metrics")
		}
		r.Date = d.Format("2006-01-02")
		if r.Sent > 0 {
			r.DeliveryRate = float64(r.Delivered) / float64(r.Sent)
		}
		daily = append(daily, r)
	}

	return response.OK(c, fiber.Map{
		"period": fiber.Map{"from": from, "to": to},
		"daily":  daily,
	})
}

func (h *MetricsHandler) InstanceMetrics(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	groupID, err := validator.ValidateUUID(c.Params("groupId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	rows, err := h.db.Query(c.Context(), `
		SELECT i.id, i.phone_number, i.status, i.daily_budget, i.messages_today,
		       i.delivery_rate,
		       COUNT(m.id) FILTER (WHERE m.status IN ('sent','delivered','read')) as total_sent,
		       COUNT(m.id) FILTER (WHERE m.status = 'failed') as total_failed
		FROM instances i
		LEFT JOIN message_logs m ON m.instance_id = i.id
		WHERE i.group_id = $1
		GROUP BY i.id ORDER BY i.priority DESC`, groupID)
	if err != nil {
		return response.ErrInternal(c, "Failed to get instance metrics")
	}
	defer rows.Close()

	type instMetric struct {
		ID            string  `json:"id"`
		Phone         *string `json:"phone_number"`
		Status        string  `json:"status"`
		DailyBudget   int     `json:"daily_budget"`
		MessagesToday int     `json:"messages_today"`
		DeliveryRate  float64 `json:"delivery_rate"`
		TotalSent     int     `json:"total_sent"`
		TotalFailed   int     `json:"total_failed"`
	}

	var metrics []instMetric
	for rows.Next() {
		var m instMetric
		if err := rows.Scan(&m.ID, &m.Phone, &m.Status, &m.DailyBudget, &m.MessagesToday,
			&m.DeliveryRate, &m.TotalSent, &m.TotalFailed); err != nil {
			return response.ErrInternal(c, "Failed to scan instance metrics")
		}
		metrics = append(metrics, m)
	}

	return response.OK(c, metrics)
}
