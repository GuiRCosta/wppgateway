package handler

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gofiber/fiber/v2"

	"github.com/guilhermecosta/wpp-gateway/internal/api/middleware"
	"github.com/guilhermecosta/wpp-gateway/internal/domain"
	"github.com/guilhermecosta/wpp-gateway/pkg/response"
)

type TenantHandler struct {
	tenantRepo   domain.TenantRepository
	groupRepo    domain.GroupRepository
	instanceRepo domain.InstanceRepository
}

func NewTenantHandler(tr domain.TenantRepository, gr domain.GroupRepository, ir domain.InstanceRepository) *TenantHandler {
	return &TenantHandler{tenantRepo: tr, groupRepo: gr, instanceRepo: ir}
}

func (h *TenantHandler) CreateTenant(c *fiber.Ctx) error {
	var input domain.CreateTenantInput
	if err := c.BodyParser(&input); err != nil {
		return response.ErrBadRequest(c, "Invalid request body")
	}
	if input.Name == "" {
		return response.ErrBadRequest(c, "Name is required")
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		return response.ErrInternal(c, "Failed to generate API key")
	}

	tenant, err := h.tenantRepo.Create(c.Context(), input, apiKey)
	if err != nil {
		return response.ErrInternal(c, "Failed to create tenant")
	}

	return response.Created(c, fiber.Map{
		"tenant":  tenant,
		"api_key": apiKey,
	})
}

func (h *TenantHandler) GetAccount(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}
	return response.OK(c, tenant)
}

func (h *TenantHandler) UpdateAccount(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	var input domain.UpdateTenantInput
	if err := c.BodyParser(&input); err != nil {
		return response.ErrBadRequest(c, "Invalid request body")
	}

	updated, err := h.tenantRepo.Update(c.Context(), tenant.ID, input)
	if err != nil {
		return response.ErrInternal(c, "Failed to update account")
	}

	return response.OK(c, updated)
}

func (h *TenantHandler) GetUsage(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	groups, err := h.groupRepo.CountByTenantID(c.Context(), tenant.ID)
	if err != nil {
		return response.ErrInternal(c, "Failed to count groups")
	}

	instances, err := h.instanceRepo.CountByTenantID(c.Context(), tenant.ID)
	if err != nil {
		return response.ErrInternal(c, "Failed to count instances")
	}

	return response.OK(c, fiber.Map{
		"groups":         groups,
		"max_groups":     tenant.MaxGroups,
		"instances":      instances,
		"max_instances":  tenant.MaxInstances,
	})
}

func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "wpp_" + hex.EncodeToString(b), nil
}
