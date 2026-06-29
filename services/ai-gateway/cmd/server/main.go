package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/config"
	aihttp "github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/http"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/middleware"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/repository"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "service", "ai-gateway", "error", err)
		os.Exit(1)
	}
	authenticator, err := middleware.NewAuthenticator(cfg.ServiceTokenHashes)
	if err != nil {
		logger.Error("service token configuration failed", "service", "ai-gateway", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	repo, err := repository.NewFileRepository(cfg.ProfileStorePath)
	if err != nil {
		logger.Error("profile store initialization failed", "service", "ai-gateway", "error", err)
		os.Exit(1)
	}
	profileService := service.New(repo,
		service.WithEncryptionKeyVersion(cfg.CredentialEncryptionKeyRef),
		service.WithCredentialEncryptionKey(cfg.CredentialEncryptionKey),
		service.WithDefaultTimeoutMs(int(cfg.DefaultTimeout.Milliseconds())),
	)
	handler := aihttp.NewServer(profileService, aihttp.Config{
		Logger:          logger,
		MaxRequestBytes: cfg.MaxRequestBytes,
		Authenticator:   authenticator,
	})
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("ai-gateway service starting", "service", "ai-gateway", "addr", cfg.HTTPAddr, "default_timeout_ms", cfg.DefaultTimeout.Milliseconds(), "database_configured", cfg.DatabaseURL != "", "profile_store_configured", cfg.ProfileStorePath != "")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("ai-gateway service stopped unexpectedly", "service", "ai-gateway", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	logger.Info("ai-gateway service shutdown started", "service", "ai-gateway")
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("ai-gateway service shutdown failed", "service", "ai-gateway", "error", err)
		os.Exit(1)
	}
	logger.Info("ai-gateway service shutdown complete", "service", "ai-gateway")
}
