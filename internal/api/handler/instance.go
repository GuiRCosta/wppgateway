package handler

import (
	"encoding/base64"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/skip2/go-qrcode"

	"github.com/guilhermecosta/wpp-gateway/internal/api/middleware"
	"github.com/guilhermecosta/wpp-gateway/internal/domain"
	"github.com/guilhermecosta/wpp-gateway/internal/instance"
	"github.com/guilhermecosta/wpp-gateway/pkg/response"
	"github.com/guilhermecosta/wpp-gateway/pkg/validator"
)

type InstanceHandler struct {
	instanceRepo domain.InstanceRepository
	groupRepo    domain.GroupRepository
	manager      *instance.Manager
}

func NewInstanceHandler(ir domain.InstanceRepository, gr domain.GroupRepository, mgr *instance.Manager) *InstanceHandler {
	return &InstanceHandler{instanceRepo: ir, groupRepo: gr, manager: mgr}
}

func (h *InstanceHandler) Create(c *fiber.Ctx) error {
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

	count, err := h.instanceRepo.CountByTenantID(c.Context(), tenant.ID)
	if err != nil {
		return response.ErrInternal(c, "Failed to check instance limit")
	}
	if count >= int64(tenant.MaxInstances) {
		return response.Err(c, fiber.StatusForbidden, "limit_exceeded", "Maximum number of instances reached")
	}

	var input domain.CreateInstanceInput
	if err := c.BodyParser(&input); err != nil {
		return response.ErrBadRequest(c, "Invalid request body")
	}
	if input.DailyBudget <= 0 {
		input.DailyBudget = 200
	}
	if input.HourlyBudget <= 0 {
		input.HourlyBudget = 30
	}

	inst, err := h.instanceRepo.Create(c.Context(), groupID, input)
	if err != nil {
		return response.ErrInternal(c, "Failed to create instance")
	}

	return response.Created(c, inst)
}

func (h *InstanceHandler) List(c *fiber.Ctx) error {
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

	instances, err := h.instanceRepo.FindByGroupID(c.Context(), groupID)
	if err != nil {
		return response.ErrInternal(c, "Failed to list instances")
	}

	return response.OK(c, instances)
}

func (h *InstanceHandler) Get(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	id, err := validator.ValidateUUID(c.Params("instanceId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	inst, err := h.instanceRepo.FindByID(c.Context(), id)
	if err != nil {
		return response.ErrInternal(c, "Failed to get instance")
	}
	if inst == nil {
		return response.ErrNotFound(c, "Instance not found")
	}

	return response.OK(c, inst)
}

func (h *InstanceHandler) Update(c *fiber.Ctx) error {
	id, err := validator.ValidateUUID(c.Params("instanceId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	var input domain.UpdateInstanceInput
	if err := c.BodyParser(&input); err != nil {
		return response.ErrBadRequest(c, "Invalid request body")
	}

	updated, err := h.instanceRepo.Update(c.Context(), id, input)
	if err != nil {
		return response.ErrInternal(c, "Failed to update instance")
	}
	if updated == nil {
		return response.ErrNotFound(c, "Instance not found")
	}

	return response.OK(c, updated)
}

func (h *InstanceHandler) Delete(c *fiber.Ctx) error {
	id, err := validator.ValidateUUID(c.Params("instanceId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	_ = h.manager.StopInstance(c.Context(), id)

	if err := h.instanceRepo.Delete(c.Context(), id); err != nil {
		return response.ErrInternal(c, "Failed to delete instance")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *InstanceHandler) GetQRCode(c *fiber.Ctx) error {
	id, err := validator.ValidateUUID(c.Params("instanceId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	qrChan, err := h.manager.StartInstance(c.Context(), id)
	if err != nil {
		return response.ErrInternal(c, err.Error())
	}

	if qrChan == nil {
		return response.OK(c, fiber.Map{
			"status":  "already_connected",
			"message": "Instance reconnected with existing session",
		})
	}

	// Wait for first QR code with timeout
	select {
	case item := <-qrChan:
		if item.Event == "code" {
			png, qrErr := qrcode.Encode(item.Code, qrcode.Medium, 512)
			if qrErr != nil {
				return response.ErrInternal(c, "Failed to generate QR image")
			}
			return response.OK(c, fiber.Map{
				"qr_code":    item.Code,
				"qr_base64":  base64.StdEncoding.EncodeToString(png),
				"expires_in": 60,
			})
		}
		return response.OK(c, fiber.Map{"status": item.Event})
	case <-time.After(30 * time.Second):
		return response.Err(c, fiber.StatusGatewayTimeout, "timeout", "QR code generation timed out")
	}
}

func (h *InstanceHandler) PairPhone(c *fiber.Ctx) error {
	id, err := validator.ValidateUUID(c.Params("instanceId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	var body struct {
		Phone string `json:"phone"`
	}
	if err := c.BodyParser(&body); err != nil || body.Phone == "" {
		return response.ErrBadRequest(c, "Phone number is required")
	}

	code, err := h.manager.PairPhone(c.Context(), id, body.Phone)
	if err != nil {
		return response.ErrInternal(c, err.Error())
	}

	return response.OK(c, fiber.Map{
		"pairing_code": code,
		"message":      "Enter this code on your phone to pair",
	})
}

func (h *InstanceHandler) Connect(c *fiber.Ctx) error {
	id, err := validator.ValidateUUID(c.Params("instanceId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	qrChan, err := h.manager.StartInstance(c.Context(), id)
	if err != nil {
		return response.ErrInternal(c, err.Error())
	}

	if qrChan != nil {
		return response.OK(c, fiber.Map{
			"status":  "needs_qr",
			"message": "No existing session. Use /qrcode or /pair to authenticate.",
		})
	}

	return response.OK(c, fiber.Map{
		"status":  "connected",
		"message": "Instance reconnected with existing session",
	})
}

func (h *InstanceHandler) Disconnect(c *fiber.Ctx) error {
	id, err := validator.ValidateUUID(c.Params("instanceId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	if err := h.manager.StopInstance(c.Context(), id); err != nil {
		return response.ErrInternal(c, "Failed to disconnect instance")
	}

	_ = h.instanceRepo.UpdateStatus(c.Context(), id, domain.StatusDisconnected)

	return response.OK(c, fiber.Map{"status": "disconnected"})
}

func (h *InstanceHandler) Restart(c *fiber.Ctx) error {
	id, err := validator.ValidateUUID(c.Params("instanceId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	_, err = h.manager.RestartInstance(c.Context(), id)
	if err != nil {
		return response.ErrInternal(c, err.Error())
	}

	return response.OK(c, fiber.Map{"status": "restarting"})
}

func (h *InstanceHandler) GetStatus(c *fiber.Ctx) error {
	id, err := validator.ValidateUUID(c.Params("instanceId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	inst, err := h.instanceRepo.FindByID(c.Context(), id)
	if err != nil {
		return response.ErrInternal(c, "Failed to get instance")
	}
	if inst == nil {
		return response.ErrNotFound(c, "Instance not found")
	}

	conn, connected := h.manager.GetConnection(id)
	wsConnected := false
	if connected {
		wsConnected = conn.IsConnected()
	}

	return response.OK(c, fiber.Map{
		"id":            inst.ID,
		"status":        inst.Status,
		"phone_number":  inst.PhoneNumber,
		"ws_connected":  wsConnected,
		"delivery_rate": inst.DeliveryRate,
	})
}
