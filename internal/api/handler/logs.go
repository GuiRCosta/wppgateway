package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/guilhermecosta/wpp-gateway/pkg/logger"
	"github.com/guilhermecosta/wpp-gateway/pkg/response"
)

type LogsHandler struct{}

func NewLogsHandler() *LogsHandler {
	return &LogsHandler{}
}

func (h *LogsHandler) List(c *fiber.Ctx) error {
	level := c.Query("level")
	limit := c.QueryInt("limit", 100)

	entries := logger.Buffer.Entries(level, limit)

	return response.OK(c, entries)
}
