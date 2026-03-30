package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/guilhermecosta/wpp-gateway/internal/api/middleware"
	"github.com/guilhermecosta/wpp-gateway/internal/domain"
	"github.com/guilhermecosta/wpp-gateway/internal/orchestrator"
	"github.com/guilhermecosta/wpp-gateway/pkg/response"
	"github.com/guilhermecosta/wpp-gateway/pkg/validator"
)

type BroadcastHandler struct {
	broadcastRepo domain.BroadcastRepository
	groupRepo     domain.GroupRepository
	dispatcher    *orchestrator.Dispatcher
}

func NewBroadcastHandler(
	br domain.BroadcastRepository,
	gr domain.GroupRepository,
	disp *orchestrator.Dispatcher,
) *BroadcastHandler {
	return &BroadcastHandler{
		broadcastRepo: br,
		groupRepo:     gr,
		dispatcher:    disp,
	}
}

func (h *BroadcastHandler) Create(c *fiber.Ctx) error {
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

	var input domain.CreateBroadcastInput
	if err := c.BodyParser(&input); err != nil {
		return response.ErrBadRequest(c, "Invalid request body")
	}
	if len(input.Recipients) == 0 {
		return response.ErrBadRequest(c, "Recipients list cannot be empty")
	}
	if input.Type == "" {
		return response.ErrBadRequest(c, "Message type is required")
	}

	broadcast, err := h.broadcastRepo.Create(c.Context(), groupID, input)
	if err != nil {
		return response.ErrInternal(c, "Failed to create broadcast")
	}

	// Create recipient records
	if err := h.broadcastRepo.CreateRecipients(c.Context(), broadcast.ID, input.Recipients); err != nil {
		return response.ErrInternal(c, "Failed to create recipient records")
	}

	// Start processing if not scheduled
	if input.Options.ScheduleAt == nil {
		h.dispatcher.StartBroadcast(c.Context(), broadcast)
	}

	return response.Created(c, fiber.Map{
		"broadcast_id":              broadcast.ID,
		"status":                    broadcast.Status,
		"total_recipients":          broadcast.Total,
		"estimated_duration_minutes": estimateDuration(broadcast.Total),
		"created_at":                broadcast.CreatedAt,
	})
}

func (h *BroadcastHandler) GetStatus(c *fiber.Ctx) error {
	broadcastID, err := validator.ValidateUUID(c.Params("broadcastId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	progress, err := h.broadcastRepo.GetProgress(c.Context(), broadcastID)
	if err != nil {
		return response.ErrInternal(c, "Failed to get broadcast progress")
	}
	if progress == nil {
		return response.ErrNotFound(c, "Broadcast not found")
	}

	return response.OK(c, progress)
}

func (h *BroadcastHandler) List(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	groupID, err := validator.ValidateUUID(c.Params("groupId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	broadcasts, total, err := h.broadcastRepo.FindByGroupID(c.Context(), groupID, limit, offset)
	if err != nil {
		return response.ErrInternal(c, "Failed to list broadcasts")
	}

	return response.List(c, broadcasts, response.Meta{
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (h *BroadcastHandler) Pause(c *fiber.Ctx) error {
	broadcastID, err := validator.ValidateUUID(c.Params("broadcastId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	if err := h.dispatcher.PauseBroadcast(broadcastID); err != nil {
		return response.ErrInternal(c, "Failed to pause broadcast")
	}

	return response.OK(c, fiber.Map{"status": "paused"})
}

func (h *BroadcastHandler) Resume(c *fiber.Ctx) error {
	broadcastID, err := validator.ValidateUUID(c.Params("broadcastId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	if err := h.dispatcher.ResumeBroadcast(c.Context(), broadcastID); err != nil {
		return response.ErrInternal(c, "Failed to resume broadcast")
	}

	return response.OK(c, fiber.Map{"status": "resumed"})
}

func (h *BroadcastHandler) Cancel(c *fiber.Ctx) error {
	broadcastID, err := validator.ValidateUUID(c.Params("broadcastId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	if err := h.dispatcher.CancelBroadcast(broadcastID); err != nil {
		return response.ErrInternal(c, "Failed to cancel broadcast")
	}

	return response.OK(c, fiber.Map{"status": "cancelled"})
}

// estimateDuration estimates broadcast duration in minutes based on recipient count.
// Assumes ~3.5s average per message (humanized delay).
func estimateDuration(totalRecipients int) int {
	seconds := float64(totalRecipients) * 3.5
	// Add chunk pauses (every 20 messages, ~60s pause)
	chunks := totalRecipients / 20
	seconds += float64(chunks) * 60

	minutes := int(seconds / 60)
	if minutes < 1 {
		minutes = 1
	}
	return minutes
}
