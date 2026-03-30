package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/guilhermecosta/wpp-gateway/internal/api/middleware"
	"github.com/guilhermecosta/wpp-gateway/internal/domain"
	"github.com/guilhermecosta/wpp-gateway/internal/instance"
	"github.com/guilhermecosta/wpp-gateway/internal/orchestrator"
	"github.com/guilhermecosta/wpp-gateway/pkg/response"
	"github.com/guilhermecosta/wpp-gateway/pkg/validator"
)

type MessageHandler struct {
	messageRepo  domain.MessageRepository
	instanceRepo domain.InstanceRepository
	groupRepo    domain.GroupRepository
	manager      *instance.Manager
	orchestrator *orchestrator.GroupOrchestrator
}

func NewMessageHandler(
	mr domain.MessageRepository,
	ir domain.InstanceRepository,
	gr domain.GroupRepository,
	mgr *instance.Manager,
	orch *orchestrator.GroupOrchestrator,
) *MessageHandler {
	return &MessageHandler{
		messageRepo:  mr,
		instanceRepo: ir,
		groupRepo:    gr,
		manager:      mgr,
		orchestrator: orch,
	}
}

func (h *MessageHandler) Send(c *fiber.Ctx) error {
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

	var input domain.SendMessageInput
	if err := c.BodyParser(&input); err != nil {
		return response.ErrBadRequest(c, "Invalid request body")
	}
	if input.To == "" || input.Type == "" {
		return response.ErrBadRequest(c, "Fields 'to' and 'type' are required")
	}

	if err := validator.ValidatePhone(input.To); err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	// Use orchestrator to select best instance based on group strategy
	selected, err := h.orchestrator.SelectInstance(c.Context(), groupID)
	if err != nil {
		return response.Err(c, fiber.StatusServiceUnavailable, "no_instance_available", err.Error())
	}
	selectedInstance := *selected

	// Create message log
	msgLog := &domain.MessageLog{
		GroupID:     &groupID,
		InstanceID:  &selectedInstance.ID,
		Recipient:   input.To,
		MessageType: input.Type,
		Status:      domain.MsgStatusQueued,
	}
	if err := h.messageRepo.Create(c.Context(), msgLog); err != nil {
		return response.ErrInternal(c, "Failed to create message log")
	}

	// Send message based on type (Phase 1: text only)
	if input.Type == "text" {
		var content struct {
			Body string `json:"body"`
		}
		if err := json.Unmarshal(input.Content, &content); err != nil {
			return response.ErrBadRequest(c, "Invalid content for text message")
		}

		_, err := h.manager.SendTextMessage(c.Context(), selectedInstance.ID, input.To, content.Body)
		if err != nil {
			_ = h.messageRepo.UpdateFailed(c.Context(), msgLog.ID, "send_failed")
			return response.ErrInternal(c, "Failed to send message: "+err.Error())
		}

		_ = h.messageRepo.UpdateSent(c.Context(), msgLog.ID)
	} else {
		return response.ErrBadRequest(c, "Only 'text' type is supported in Phase 1")
	}

	return response.OK(c, fiber.Map{
		"message_id":  msgLog.ID,
		"status":      "sent",
		"instance_id": selectedInstance.ID,
	})
}

func (h *MessageHandler) GetStatus(c *fiber.Ctx) error {
	msgID, err := validator.ValidateUUID(c.Params("messageId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	msg, err := h.messageRepo.FindByID(c.Context(), msgID)
	if err != nil {
		return response.ErrInternal(c, "Failed to get message")
	}
	if msg == nil {
		return response.ErrNotFound(c, "Message not found")
	}

	return response.OK(c, msg)
}

func (h *MessageHandler) ListByGroup(c *fiber.Ctx) error {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return response.ErrUnauthorized(c, "Tenant not found")
	}

	groupID, err := validator.ValidateUUID(c.Params("groupId"))
	if err != nil {
		return response.ErrBadRequest(c, err.Error())
	}

	filter := domain.MessageFilter{
		Limit:  c.QueryInt("limit", 50),
		Offset: c.QueryInt("offset", 0),
	}

	if status := c.Query("status"); status != "" {
		s := domain.MessageStatus(status)
		filter.Status = &s
	}
	if recipient := c.Query("recipient"); recipient != "" {
		filter.Recipient = &recipient
	}
	if instID := c.Query("instance_id"); instID != "" {
		parsed, parseErr := uuid.Parse(instID)
		if parseErr == nil {
			filter.InstanceID = &parsed
		}
	}

	messages, total, err := h.messageRepo.FindByGroupID(c.Context(), groupID, filter)
	if err != nil {
		return response.ErrInternal(c, "Failed to list messages")
	}

	return response.List(c, messages, response.Meta{
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	})
}
