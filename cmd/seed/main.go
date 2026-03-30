package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/guilhermecosta/wpp-gateway/internal/config"
	"github.com/guilhermecosta/wpp-gateway/internal/domain"
	"github.com/guilhermecosta/wpp-gateway/internal/store/postgres"
	"github.com/guilhermecosta/wpp-gateway/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)
	ctx := context.Background()

	pool, err := postgres.NewPool(ctx, cfg.Database.URL, 5, 1)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()

	tenantRepo := postgres.NewTenantRepo(pool)

	apiKey, err := generateAPIKey()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to generate API key")
	}

	tenant, err := tenantRepo.Create(ctx, domain.CreateTenantInput{
		Name: "Default Tenant",
	}, apiKey)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create tenant")
	}

	fmt.Println("=== Tenant Created ===")
	fmt.Printf("ID:      %s\n", tenant.ID)
	fmt.Printf("Name:    %s\n", tenant.Name)
	fmt.Printf("API Key: %s\n", apiKey)
	fmt.Println("======================")
	fmt.Println("\nUse this API key in the X-API-Key header for all requests.")
}

func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "wpp_" + hex.EncodeToString(b), nil
}
