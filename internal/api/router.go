package api

import (
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/guilhermecosta/wpp-gateway/internal/api/handler"
	"github.com/guilhermecosta/wpp-gateway/internal/api/middleware"
	"github.com/guilhermecosta/wpp-gateway/internal/domain"
	"github.com/guilhermecosta/wpp-gateway/internal/instance"
	"github.com/guilhermecosta/wpp-gateway/internal/orchestrator"
	"github.com/guilhermecosta/wpp-gateway/internal/store/postgres"
	"github.com/guilhermecosta/wpp-gateway/internal/webhook"
	"github.com/guilhermecosta/wpp-gateway/web"
)

type Dependencies struct {
	DB            *pgxpool.Pool
	Redis         *redis.Client
	TenantRepo    domain.TenantRepository
	GroupRepo     domain.GroupRepository
	InstanceRepo  domain.InstanceRepository
	MessageRepo   domain.MessageRepository
	BroadcastRepo domain.BroadcastRepository
	BlacklistRepo *postgres.BlacklistRepo
	AuditRepo     *postgres.AuditRepo
	Manager       *instance.Manager
	Orchestrator  *orchestrator.GroupOrchestrator
	Dispatcher    *orchestrator.Dispatcher
	Webhook       *webhook.Emitter
	Log           zerolog.Logger
}

func NewRouter(deps Dependencies) *fiber.App {
	app := fiber.New(fiber.Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		BodyLimit:    10 * 1024 * 1024, // 10MB
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			message := "Internal server error"
			errCode := "internal_error"
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
				message = e.Message
			}

			switch {
			case code == fiber.StatusNotFound:
				errCode = "not_found"
				deps.Log.Warn().Str("path", c.Path()).Str("method", c.Method()).Msg("route not found")
			case code >= 400 && code < 500:
				errCode = "client_error"
				deps.Log.Warn().Err(err).Str("path", c.Path()).Int("status", code).Msg("client error")
			default:
				deps.Log.Error().Err(err).Str("path", c.Path()).Int("status", code).Msg("unhandled error")
			}

			return c.Status(code).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    errCode,
					"message": message,
				},
			})
		},
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(helmet.New(helmet.Config{
		CrossOriginEmbedderPolicy: "unsafe-none",
		ContentSecurityPolicy:     "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.tailwindcss.com https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://cdn.tailwindcss.com https://fonts.googleapis.com; connect-src 'self'; img-src 'self' data:; font-src 'self' https://fonts.gstatic.com",
	}))

	allowedOrigins := os.Getenv("CORS_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "https://wpp.ideva.ai"
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE",
		AllowHeaders:     "Origin,Content-Type,Accept,X-API-Key",
		AllowCredentials: false,
	}))
	app.Use(middleware.RequestLogger(deps.Log))

	// Handlers
	healthH := handler.NewHealthHandler(deps.DB, deps.Redis)
	tenantH := handler.NewTenantHandler(deps.TenantRepo, deps.GroupRepo, deps.InstanceRepo)
	groupH := handler.NewGroupHandler(deps.GroupRepo, deps.Orchestrator)
	instanceH := handler.NewInstanceHandler(deps.InstanceRepo, deps.GroupRepo, deps.Manager)
	messageH := handler.NewMessageHandler(deps.MessageRepo, deps.InstanceRepo, deps.GroupRepo, deps.Manager, deps.Orchestrator)
	broadcastH := handler.NewBroadcastHandler(deps.BroadcastRepo, deps.GroupRepo, deps.Dispatcher)
	blacklistH := handler.NewBlacklistHandler(deps.BlacklistRepo, deps.GroupRepo)
	auditH := handler.NewAuditHandler(deps.AuditRepo)
	metricsH := handler.NewMetricsHandler(deps.DB, deps.GroupRepo)
	webhookH := handler.NewWebhookHandler(deps.GroupRepo, deps.Webhook)

	// Public routes
	app.Get("/health", healthH.Health)

	// Metrics on internal port (not exposed publicly)
	go func() {
		metricsApp := fiber.New(fiber.Config{DisableStartupMessage: true})
		metricsApp.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
		if err := metricsApp.Listen("127.0.0.1:9091"); err != nil {
			deps.Log.Error().Err(err).Msg("failed to start metrics server")
		}
	}()

	// Tenant registration with rate limit (no auth required)
	app.Post("/register", middleware.RateLimit(5, time.Hour), tenantH.CreateTenant)

	// API v1
	v1 := app.Group("/v1")
	v1.Use(middleware.Auth(deps.TenantRepo))
	v1.Use(middleware.RateLimit(1000, time.Minute))

	// Health (authenticated)
	v1.Get("/health/detailed", healthH.HealthDetailed)

	// Account
	v1.Get("/account", tenantH.GetAccount)
	v1.Patch("/account", tenantH.UpdateAccount)
	v1.Get("/account/usage", tenantH.GetUsage)

	// Groups
	v1.Post("/groups", groupH.Create)
	v1.Get("/groups", groupH.List)
	v1.Get("/groups/:groupId", groupH.Get)
	v1.Patch("/groups/:groupId", groupH.Update)
	v1.Delete("/groups/:groupId", groupH.Delete)
	v1.Get("/groups/:groupId/status", groupH.GetStatus)
	v1.Post("/groups/:groupId/pause", groupH.Pause)
	v1.Post("/groups/:groupId/resume", groupH.Resume)

	// Instances
	v1.Post("/groups/:groupId/instances", instanceH.Create)
	v1.Get("/groups/:groupId/instances", instanceH.List)
	v1.Get("/instances/:instanceId", instanceH.Get)
	v1.Patch("/instances/:instanceId", instanceH.Update)
	v1.Delete("/instances/:instanceId", instanceH.Delete)

	// Instance lifecycle
	v1.Get("/instances/:instanceId/qrcode", instanceH.GetQRCode)
	v1.Post("/instances/:instanceId/pair", instanceH.PairPhone)
	v1.Post("/instances/:instanceId/connect", instanceH.Connect)
	v1.Post("/instances/:instanceId/disconnect", instanceH.Disconnect)
	v1.Post("/instances/:instanceId/restart", instanceH.Restart)
	v1.Get("/instances/:instanceId/status", instanceH.GetStatus)

	// Messages
	v1.Post("/groups/:groupId/messages/send", messageH.Send)
	v1.Get("/groups/:groupId/messages", messageH.ListByGroup)
	v1.Get("/groups/:groupId/messages/:messageId/status", messageH.GetStatus)

	// Broadcasts
	v1.Post("/groups/:groupId/messages/broadcast", broadcastH.Create)
	v1.Get("/groups/:groupId/messages/broadcast", broadcastH.List)
	v1.Get("/groups/:groupId/messages/broadcast/:broadcastId", broadcastH.GetStatus)
	v1.Post("/groups/:groupId/messages/broadcast/:broadcastId/pause", broadcastH.Pause)
	v1.Post("/groups/:groupId/messages/broadcast/:broadcastId/resume", broadcastH.Resume)
	v1.Post("/groups/:groupId/messages/broadcast/:broadcastId/cancel", broadcastH.Cancel)

	// Metrics
	v1.Get("/groups/:groupId/metrics", metricsH.GroupMetrics)
	v1.Get("/groups/:groupId/metrics/daily", metricsH.DailyMetrics)
	v1.Get("/groups/:groupId/metrics/instances", metricsH.InstanceMetrics)

	// Blacklist
	v1.Get("/groups/:groupId/blacklist", blacklistH.List)
	v1.Post("/groups/:groupId/blacklist", blacklistH.Add)
	v1.Delete("/groups/:groupId/blacklist/:number", blacklistH.Remove)

	// Webhooks
	v1.Put("/groups/:groupId/webhook", webhookH.Configure)
	v1.Get("/groups/:groupId/webhook", webhookH.Get)
	v1.Post("/groups/:groupId/webhook/test", webhookH.Test)

	// Audit
	v1.Get("/account/audit-log", auditH.List)

	// Logs (restricted - each tenant sees only their own request logs)
	logsH := handler.NewLogsHandler()
	v1.Get("/logs", logsH.List)


	// Frontend static files
	staticFS, _ := fs.Sub(web.StaticFS, "static")
	app.Use("/static", filesystem.New(filesystem.Config{
		Root:       http.FS(staticFS),
		PathPrefix: "",
		Browse:     false,
	}))
	app.Get("/", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		data, err := web.StaticFS.ReadFile("static/index.html")
		if err != nil {
			return c.Status(500).SendString("Frontend not found")
		}
		return c.Send(data)
	})
	app.Get("/docs", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		data, err := web.StaticFS.ReadFile("static/docs.html")
		if err != nil {
			return c.Status(500).SendString("Docs not found")
		}
		return c.Send(data)
	})

	return app
}
