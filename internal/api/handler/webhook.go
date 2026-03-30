package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/guilhermecosta/wpp-gateway/internal/api/middleware"
	"github.com/guilhermecosta/wpp-gateway/internal/domain"
	"github.com/guilhermecosta/wpp-gateway/internal/webhook"
	"github.com/guilhermecosta/wpp-gateway/pkg/response"
	"github.com/guilhermecosta/wpp-gateway/pkg/validator"
)

type WebhookHandler struct {
	groupRepo   domain.GroupRepository
	webhookEmit *webhook.Emitter
}

func NewWebhookHandler(gr domain.GroupRepository, we *webhook.Emitter) *WebhookHandler {
	return &WebhookHandler{groupRepo: gr, webhookEmit: we}
}

func (h *WebhookHandler) Configure(c *fiber.Ctx) error {
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
		URL    string   `json:"url"`
		Events []string `json:"events"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.ErrBadRequest(c, "Invalid request body")
	}
	if body.URL != "" {
		if err := validator.ValidateWebhookURL(body.URL); err != nil {
			return response.ErrBadRequest(c, err.Error())
		}
	}

	input := domain.UpdateGroupInput{
		WebhookURL:    &body.URL,
		WebhookEvents: &body.Events,
	}

	updated, err := h.groupRepo.Update(c.Context(), groupID, input)
	if err != nil {
		return response.ErrInternal(c, "Failed to update webhook")
	}

	return response.OK(c, fiber.Map{
		"webhook_url":    updated.WebhookURL,
		"webhook_events": updated.WebhookEvents,
	})
}

func (h *WebhookHandler) Get(c *fiber.Ctx) error {
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

	return response.OK(c, fiber.Map{
		"webhook_url":    group.WebhookURL,
		"webhook_events": group.WebhookEvents,
	})
}

func (h *WebhookHandler) Test(c *fiber.Ctx) error {
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

	if group.WebhookURL == nil || *group.WebhookURL == "" {
		return response.ErrBadRequest(c, "No webhook URL configured for this group")
	}

	secret := ""
	if group.WebhookSecret != nil {
		secret = *group.WebhookSecret
	}

	h.webhookEmit.Emit(*group.WebhookURL, secret, webhook.Event{
		Name:      "webhook.test",
		GroupID:   groupID.String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data: fiber.Map{
			"message": "This is a test webhook event",
		},
	})

	return response.OK(c, fiber.Map{
		"status":  "sent",
		"message": "Test webhook event dispatched",
	})
}
