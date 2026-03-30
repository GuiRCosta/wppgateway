package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/guilhermecosta/wpp-gateway/internal/api/middleware"
	"github.com/guilhermecosta/wpp-gateway/internal/store/postgres"
	"github.com/guilhermecosta/wpp-gateway/pkg/response"
)

type AuditHandler struct {
	auditRepo *postgres.AuditRepo
}

func NewAuditHandler(ar *postgres.AuditRepo) *AuditHandler {
	return &AuditHandler{auditRepo: ar}
}

func (h *AuditHandler) List(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	action := c.Query("action")
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	entries, total, err := h.auditRepo.List(c.Context(), tenant.ID, action, limit, offset)
	if err != nil {
		return response.ErrInternal(c, "Failed to list audit log")
	}

	return response.List(c, entries, response.Meta{Total: total, Limit: limit, Offset: offset})
}
