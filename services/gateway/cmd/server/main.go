package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/config"
	gatewayhttp "github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/http"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/platform/authclient"
	redisstore "github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/platform/redis"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "service", "gateway", "error", err)
		os.Exit(1)
	}

	hasher, err := service.NewTokenHasher(cfg.TokenHashSecret, cfg.TokenHashKeyVersion)
	if err != nil {
		logger.Error("token hasher configuration failed", "service", "gateway", "error", err)
		os.Exit(1)
	}

	authClient := authclient.New(cfg.AuthBaseURL, authclient.Config{
		ServiceToken: cfg.AuthServiceToken,
		Timeout:      cfg.AuthTimeout,
	})
	sessionStore := redisstore.New(redisstore.Config{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	handler, err := gatewayhttp.NewServer(authClient, sessionStore, hasher, gatewayhttp.Config{
		Logger:          logger,
		MaxRequestBytes: cfg.MaxRequestBytes,
		Ready:           sessionStore.Ping,
	})
	if err != nil {
		logger.Error("gateway server configuration failed", "service", "gateway", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: handler,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("gateway service starting", "service", "gateway", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("gateway service stopped unexpectedly", "service", "gateway", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	logger.Info("gateway service shutdown started", "service", "gateway")
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("gateway service shutdown failed", "service", "gateway", "error", err)
		os.Exit(1)
	}
	logger.Info("gateway service shutdown complete", "service", "gateway")
}
