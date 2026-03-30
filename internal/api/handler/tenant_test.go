package handler_test

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guilhermecosta/wpp-gateway/internal/api/handler"
	"github.com/guilhermecosta/wpp-gateway/internal/api/middleware"
	"github.com/guilhermecosta/wpp-gateway/internal/testutil"
)

func setupTenantApp() (*fiber.App, *testutil.MockTenantRepo, *testutil.MockGroupRepo, *testutil.MockInstanceRepo) {
	tenantRepo := testutil.NewMockTenantRepo()
	groupRepo := testutil.NewMockGroupRepo()
	instanceRepo := testutil.NewMockInstanceRepo()

	h := handler.NewTenantHandler(tenantRepo, groupRepo, instanceRepo)

	app := fiber.New()
	app.Post("/v1/tenants", h.CreateTenant)

	v1 := app.Group("/v1")
	v1.Use(middleware.Auth(tenantRepo))
	v1.Get("/account", h.GetAccount)
	v1.Get("/account/usage", h.GetUsage)

	return app, tenantRepo, groupRepo, instanceRepo
}

func TestCreateTenant(t *testing.T) {
	app, _, _, _ := setupTenantApp()

	req := httptest.NewRequest("POST", "/v1/tenants", strings.NewReader(`{"name":"Test Tenant"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "api_key")
	assert.Contains(t, string(body), "Test Tenant")
}

func TestCreateTenantMissingName(t *testing.T) {
	app, _, _, _ := setupTenantApp()

	req := httptest.NewRequest("POST", "/v1/tenants", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestGetAccountUnauthorized(t *testing.T) {
	app, _, _, _ := setupTenantApp()

	req := httptest.NewRequest("GET", "/v1/account", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)
}

func TestGetAccountWithValidKey(t *testing.T) {
	app, tenantRepo, _, _ := setupTenantApp()

	tenant, _ := tenantRepo.Create(nil, testutil.MockTenantInput("My Company"), "wpp_test123")

	req := httptest.NewRequest("GET", "/v1/account", nil)
	req.Header.Set("X-API-Key", "wpp_test123")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), tenant.Name)
}

func TestGetUsage(t *testing.T) {
	app, tenantRepo, _, _ := setupTenantApp()

	tenantRepo.Create(nil, testutil.MockTenantInput("Company"), "wpp_usage_key")

	req := httptest.NewRequest("GET", "/v1/account/usage", nil)
	req.Header.Set("X-API-Key", "wpp_usage_key")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "max_groups")
}
