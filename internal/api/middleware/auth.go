package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/guilhermecosta/wpp-gateway/internal/domain"
	"github.com/guilhermecosta/wpp-gateway/pkg/response"
)

const TenantKey = "tenant"

func Auth(tenantRepo domain.TenantRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			return response.ErrUnauthorized(c, "Missing X-API-Key header")
		}

		tenant, err := tenantRepo.FindByAPIKey(c.Context(), apiKey)
		if err != nil {
			return response.ErrInternal(c, "Failed to validate API key")
		}
		if tenant == nil {
			return response.ErrUnauthorized(c, "Invalid API key")
		}
		if !tenant.IsActive {
			return response.ErrForbidden(c, "Account is deactivated")
		}

		c.Locals(TenantKey, tenant)
		return c.Next()
	}
}

func GetTenant(c *fiber.Ctx) *domain.Tenant {
	t, ok := c.Locals(TenantKey).(*domain.Tenant)
	if !ok {
		return nil
	}
	return t
}
