package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/guilhermecosta/wpp-gateway/internal/api/middleware"
	"github.com/guilhermecosta/wpp-gateway/internal/domain"
	"github.com/guilhermecosta/wpp-gateway/internal/store/postgres"
	"github.com/guilhermecosta/wpp-gateway/pkg/response"
	"github.com/guilhermecosta/wpp-gateway/pkg/validator"
)

type BlacklistHandler struct {
	blacklistRepo *postgres.BlacklistRepo
	groupRepo     domain.GroupRepository
}

func NewBlacklistHandler(br *postgres.BlacklistRepo, gr domain.GroupRepository) *BlacklistHandler {
	return &BlacklistHandler{blacklistRepo: br, groupRepo: gr}
}

func (h *BlacklistHandler) List(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	groupID, err := validator.ValidateUUID(c.Params("groupId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	group, err := h.groupRepo.FindByID(c.Context(), groupID)
	if err != nil {
		return response.ErrInternal(c, "Failed to get group")
	}
	if group == nil || group.TenantID != tenant.ID {
		return response.ErrNotFound(c, "Group not found")
	}

	limit := c.QueryInt("limit", 50)
	if limit > 200 {
		limit = 200
	}
	offset := c.QueryInt("offset", 0)

	entries, total, err := h.blacklistRepo.List(c.Context(), groupID, limit, offset)
	if err != nil {
		return response.ErrInternal(c, "Failed to list blacklist")
	}

	return response.List(c, entries, response.Meta{Total: total, Limit: limit, Offset: offset})
}

func (h *BlacklistHandler) Add(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	groupID, err := validator.ValidateUUID(c.Params("groupId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	group, err := h.groupRepo.FindByID(c.Context(), groupID)
	if err != nil {
		return response.ErrInternal(c, "Failed to get group")
	}
	if group == nil || group.TenantID != tenant.ID {
		return response.ErrNotFound(c, "Group not found")
	}

	var body struct {
		Numbers []string `json:"numbers"`
		Reason  string   `json:"reason"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.ErrBadRequest(c, "Invalid request body")
	}
	if len(body.Numbers) == 0 {
		return response.ErrBadRequest(c, "Numbers list cannot be empty")
	}
	if body.Reason == "" {
		body.Reason = "opt_out"
	}

	added, err := h.blacklistRepo.Add(c.Context(), groupID, body.Numbers, body.Reason)
	if err != nil {
		return response.ErrInternal(c, "Failed to add to blacklist")
	}

	return response.OK(c, fiber.Map{"added": added})
}

func (h *BlacklistHandler) Remove(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	groupID, err := validator.ValidateUUID(c.Params("groupId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	group, err := h.groupRepo.FindByID(c.Context(), groupID)
	if err != nil {
		return response.ErrInternal(c, "Failed to get group")
	}
	if group == nil || group.TenantID != tenant.ID {
		return response.ErrNotFound(c, "Group not found")
	}

	phone := c.Params("number")
	if phone == "" {
		return response.ErrBadRequest(c, "Phone number is required")
	}

	if err := h.blacklistRepo.Remove(c.Context(), groupID, phone); err != nil {
		return response.ErrInternal(c, "Failed to remove from blacklist")
	}

	return c.SendStatus(fiber.StatusNoContent)
}
