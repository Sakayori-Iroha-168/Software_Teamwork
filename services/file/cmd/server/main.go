package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/file/internal/config"
	filehttp "github.com/Sakayori-Iroha-168/Software_Teamwork/services/file/internal/http"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/file/internal/platform/storage"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/file/internal/repository"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/file/internal/service"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "service", "file", "error", err)
		os.Exit(1)
	}

	repo, closeRepo, err := newRepository(cfg)
	if err != nil {
		logger.Error("repository initialization failed", "service", "file", "error", err)
		os.Exit(1)
	}
	defer closeRepo()
	objectStore, err := newObjectStore(cfg)
	if err != nil {
		logger.Error("storage initialization failed", "service", "file", "error", err)
		os.Exit(1)
	}
	documentService := service.New(repo, objectStore, service.WithStorageBackend(cfg.StorageBackend))
	handler := filehttp.NewServer(documentService, filehttp.Config{
		MaxUploadBytes: cfg.MaxUploadBytes,
		Logger:         logger,
	})

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: handler,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("file service starting", "service", "file", "addr", cfg.HTTPAddr, "storage_backend", cfg.StorageBackend, "metadata_backend", metadataBackend(cfg))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("file service stopped unexpectedly", "service", "file", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	logger.Info("file service shutdown started", "service", "file")
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("file service shutdown failed", "service", "file", "error", err)
		os.Exit(1)
	}
	logger.Info("file service shutdown complete", "service", "file")
}

func newRepository(cfg config.Config) (service.DocumentRepository, func(), error) {
	if cfg.DatabaseURL == "" {
		return repository.NewMemoryRepository(), func() {}, nil
	}
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, nil, err
	}
	return repository.NewPostgresRepository(db), func() { _ = db.Close() }, nil
}

func metadataBackend(cfg config.Config) string {
	if cfg.DatabaseURL != "" {
		return "postgres"
	}
	return "memory"
}

func newObjectStore(cfg config.Config) (service.ObjectStore, error) {
	switch cfg.StorageBackend {
	case "memory":
		return storage.NewMemoryStore(), nil
	case "local":
		return storage.NewLocalStore(cfg.LocalStorageDir)
	case "minio":
		client, err := storage.NewMinIOClient(storage.MinIOClientConfig{
			Endpoint:  cfg.MinIOEndpoint,
			AccessKey: cfg.MinIOAccessKey,
			SecretKey: cfg.MinIOSecretKey,
			UseSSL:    cfg.MinIOUseSSL,
			Region:    cfg.MinIORegion,
			Timeout:   cfg.MinIOTimeout,
		})
		if err != nil {
			return nil, err
		}
		return storage.NewMinIOStore(client, cfg.MinIOBucket)
	default:
		return nil, fmt.Errorf("unsupported storage backend %q", cfg.StorageBackend)
	}
}
