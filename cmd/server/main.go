package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/guilhermecosta/wpp-gateway/internal/api"
	"github.com/guilhermecosta/wpp-gateway/internal/config"
	"github.com/guilhermecosta/wpp-gateway/internal/instance"
	"github.com/guilhermecosta/wpp-gateway/internal/metrics"
	"github.com/guilhermecosta/wpp-gateway/internal/orchestrator"
	"github.com/guilhermecosta/wpp-gateway/internal/store/postgres"
	redisStore "github.com/guilhermecosta/wpp-gateway/internal/store/redis"
	"github.com/guilhermecosta/wpp-gateway/internal/webhook"
	"github.com/guilhermecosta/wpp-gateway/pkg/logger"

	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)
	log.Info().Int("port", cfg.Server.Port).Msg("starting wpp-gateway")

	// Register Prometheus metrics
	metrics.Register()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Database
	pool, err := postgres.NewPool(ctx, cfg.Database.URL, cfg.Database.MaxConns, cfg.Database.MinConns)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()
	log.Info().Msg("connected to PostgreSQL")

	// Run migrations
	migrationsPath, err := filepath.Abs("migrations")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to resolve migrations path")
	}
	if err := postgres.RunMigrations(cfg.Database.URL, migrationsPath); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}
	log.Info().Msg("migrations completed")

	// Redis
	redisClient, err := redisStore.NewClient(ctx, cfg.Redis.URL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to Redis")
	}
	defer redisClient.Close()
	log.Info().Msg("connected to Redis")

	// WhatsApp session store (whatsmeow sqlstore using same Postgres)
	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open sql.DB for whatsmeow")
	}
	defer db.Close()

	waLogger := waLog.Stdout("whatsmeow-store", "WARN", true)
	container := sqlstore.NewWithDB(db, "postgres", waLogger)
	if err := container.Upgrade(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to upgrade whatsmeow store")
	}
	log.Info().Msg("whatsmeow store ready")

	// Repositories
	tenantRepo := postgres.NewTenantRepo(pool)
	groupRepo := postgres.NewGroupRepo(pool)
	instanceRepo := postgres.NewInstanceRepo(pool)
	messageRepo := postgres.NewMessageRepo(pool)
	broadcastRepo := postgres.NewBroadcastRepo(pool)
	blacklistRepo := postgres.NewBlacklistRepo(pool)
	auditRepo := postgres.NewAuditRepo(pool)
	deviceMapper := postgres.NewDeviceMappingRepo(pool)

	// Webhook emitter
	webhookEmitter := webhook.NewEmitter(cfg.Webhook.Timeout, cfg.Webhook.MaxRetries, log)
	webhookEmitter.Start(ctx)

	// Instance manager
	manager := instance.NewManager(container, instanceRepo, groupRepo, deviceMapper, webhookEmitter, log)

	// Group orchestrator and dispatcher
	orch := orchestrator.NewGroupOrchestrator(groupRepo, instanceRepo, manager, webhookEmitter, log)
	dispatcher := orchestrator.NewDispatcher(orch, manager, broadcastRepo, log)

	// Health monitor
	healthMon := instance.NewHealthMonitor(manager, instanceRepo, 30*time.Second, log)
	healthMon.OnDegraded(func(instanceID, groupID uuid.UUID) {
		if replacement, err := orch.HandleInstanceFailure(ctx, groupID, instanceID); err == nil {
			log.Info().
				Str("failed", instanceID.String()).
				Str("replacement", replacement.ID.String()).
				Msg("failover executed by health monitor")
		}
	})
	healthMon.Start(ctx)

	// HTTP Router
	app := api.NewRouter(api.Dependencies{
		DB:            pool,
		Redis:         redisClient,
		TenantRepo:    tenantRepo,
		GroupRepo:     groupRepo,
		InstanceRepo:  instanceRepo,
		MessageRepo:   messageRepo,
		BroadcastRepo: broadcastRepo,
		BlacklistRepo: blacklistRepo,
		AuditRepo:     auditRepo,
		Manager:       manager,
		Orchestrator:  orch,
		Dispatcher:    dispatcher,
		Webhook:       webhookEmitter,
		Log:           log,
	})

	// Start server in goroutine
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	go func() {
		if err := app.Listen(addr); err != nil {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()
	log.Info().Str("addr", addr).Msg("server started")

	// Wait for shutdown signal
	<-ctx.Done()
	log.Info().Msg("shutting down gracefully...")

	healthMon.Stop()
	manager.DisconnectAll()
	if err := app.Shutdown(); err != nil {
		log.Error().Err(err).Msg("server shutdown error")
	}

	log.Info().Msg("shutdown complete")
}
