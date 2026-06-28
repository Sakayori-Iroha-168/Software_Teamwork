package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/config"
	httpapi "github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/http"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/repository"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "service", "qa", "operation", "load_config", "error", err)
		os.Exit(1)
	}

	repo := repository.NewMemoryRepository()
	sessions := service.New(repo)
	handler := httpapi.NewServer(sessions, httpapi.Config{Logger: logger})

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: handler,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("qa service starting", "service", "qa", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("qa service stopped unexpectedly", "service", "qa", "operation", "listen", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	logger.Info("qa service shutdown started", "service", "qa")
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("qa service shutdown failed", "service", "qa", "operation", "shutdown", "error", err)
		os.Exit(1)
	}
	logger.Info("qa service shutdown complete", "service", "qa")
}
