package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/guilhermecosta/wpp-gateway/pkg/response"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(db *pgxpool.Pool, redis *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redis}
}

func (h *HealthHandler) Health(c *fiber.Ctx) error {
	return response.OK(c, fiber.Map{"status": "ok"})
}

func (h *HealthHandler) HealthDetailed(c *fiber.Ctx) error {
	ctx := c.Context()

	pgStatus := "up"
	if err := h.db.Ping(ctx); err != nil {
		pgStatus = "down"
	}

	redisStatus := "up"
	if err := h.redis.Ping(ctx).Err(); err != nil {
		redisStatus = "down"
	}

	status := "healthy"
	if pgStatus == "down" || redisStatus == "down" {
		status = "degraded"
	}

	return response.OK(c, fiber.Map{
		"status": status,
		"dependencies": fiber.Map{
			"postgresql": fiber.Map{"status": pgStatus},
			"redis":      fiber.Map{"status": redisStatus},
		},
	})
}
