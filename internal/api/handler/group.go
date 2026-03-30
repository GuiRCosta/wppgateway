package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/guilhermecosta/wpp-gateway/internal/api/middleware"
	"github.com/guilhermecosta/wpp-gateway/internal/domain"
	"github.com/guilhermecosta/wpp-gateway/internal/orchestrator"
	"github.com/guilhermecosta/wpp-gateway/pkg/response"
	"github.com/guilhermecosta/wpp-gateway/pkg/validator"
)

type GroupHandler struct {
	groupRepo    domain.GroupRepository
	orchestrator *orchestrator.GroupOrchestrator
}

func NewGroupHandler(gr domain.GroupRepository, orch *orchestrator.GroupOrchestrator) *GroupHandler {
	return &GroupHandler{groupRepo: gr, orchestrator: orch}
}

func (h *GroupHandler) Create(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	count, err := h.groupRepo.CountByTenantID(c.Context(), tenant.ID)
	if err != nil {
		return response.ErrInternal(c, "Failed to check group limit")
	}
	if count >= int64(tenant.MaxGroups) {
		return response.Err(c, fiber.StatusForbidden, "limit_exceeded", "Maximum number of groups reached")
	}

	var input domain.CreateGroupInput
	if err := c.BodyParser(&input); err != nil {
		return response.ErrBadRequest(c, "Invalid request body")
	}
	if input.Name == "" {
		return response.ErrBadRequest(c, "Name is required")
	}
	if input.Strategy == "" {
		input.Strategy = domain.StrategyFailover
	}

	group, err := h.groupRepo.Create(c.Context(), tenant.ID, input)
	if err != nil {
		return response.ErrInternal(c, "Failed to create group")
	}

	return response.Created(c, group)
}

func (h *GroupHandler) List(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	groups, err := h.groupRepo.FindByTenantID(c.Context(), tenant.ID)
	if err != nil {
		return response.ErrInternal(c, "Failed to list groups")
	}

	return response.OK(c, groups)
}

func (h *GroupHandler) Get(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	id, err := validator.ValidateUUID(c.Params("groupId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	group, err := h.groupRepo.FindByID(c.Context(), id)
	if err != nil {
		return response.ErrInternal(c, "Failed to get group")
	}
	if group == nil || group.TenantID != tenant.ID {
		return response.ErrNotFound(c, "Group not found")
	}

	return response.OK(c, group)
}

func (h *GroupHandler) Update(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	id, err := validator.ValidateUUID(c.Params("groupId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	group, err := h.groupRepo.FindByID(c.Context(), id)
	if err != nil {
		return response.ErrInternal(c, "Failed to get group")
	}
	if group == nil || group.TenantID != tenant.ID {
		return response.ErrNotFound(c, "Group not found")
	}

	var input domain.UpdateGroupInput
	if err := c.BodyParser(&input); err != nil {
		return response.ErrBadRequest(c, "Invalid request body")
	}

	updated, err := h.groupRepo.Update(c.Context(), id, input)
	if err != nil {
		return response.ErrInternal(c, "Failed to update group")
	}

	return response.OK(c, updated)
}

func (h *GroupHandler) Delete(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	id, err := validator.ValidateUUID(c.Params("groupId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	group, err := h.groupRepo.FindByID(c.Context(), id)
	if err != nil {
		return response.ErrInternal(c, "Failed to get group")
	}
	if group == nil || group.TenantID != tenant.ID {
		return response.ErrNotFound(c, "Group not found")
	}

	if err := h.groupRepo.Delete(c.Context(), id); err != nil {
		return response.ErrInternal(c, "Failed to delete group")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *GroupHandler) GetStatus(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	id, err := validator.ValidateUUID(c.Params("groupId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	group, err := h.groupRepo.FindByID(c.Context(), id)
	if err != nil {
		return response.ErrInternal(c, "Failed to get group")
	}
	if group == nil || group.TenantID != tenant.ID {
		return response.ErrNotFound(c, "Group not found")
	}

	status, err := h.orchestrator.GetGroupStatus(c.Context(), id)
	if err != nil {
		return response.ErrInternal(c, "Failed to get group status")
	}

	return response.OK(c, status)
}

func (h *GroupHandler) Pause(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	id, err := validator.ValidateUUID(c.Params("groupId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	group, err := h.groupRepo.FindByID(c.Context(), id)
	if err != nil {
		return response.ErrInternal(c, "Failed to get group")
	}
	if group == nil || group.TenantID != tenant.ID {
		return response.ErrNotFound(c, "Group not found")
	}

	h.orchestrator.PauseGroup(id)

	return response.OK(c, fiber.Map{"status": "paused"})
}

func (h *GroupHandler) Resume(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	id, err := validator.ValidateUUID(c.Params("groupId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	group, err := h.groupRepo.FindByID(c.Context(), id)
	if err != nil {
		return response.ErrInternal(c, "Failed to get group")
	}
	if group == nil || group.TenantID != tenant.ID {
		return response.ErrNotFound(c, "Group not found")
	}

	h.orchestrator.ResumeGroup(id)

	return response.OK(c, fiber.Map{"status": "resumed"})
}
