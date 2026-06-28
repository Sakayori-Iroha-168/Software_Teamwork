package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/config"
	httpx "github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/http"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/repository"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := repository.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	conversations := repository.NewConversationRepository(pool)
	messages := repository.NewMessageRepository(pool)
	responseRuns := repository.NewResponseRunRepository(pool)
	processSteps := repository.NewProcessStepRepository(pool)
	contentBlocks := repository.NewContentBlockRepository(pool)
	streamEvents := repository.NewResponseStreamEventRepository(pool)
	citations := repository.NewCitationRepository(pool)

	chatService := service.NewChatStreamService(conversations, messages, responseRuns, processSteps, contentBlocks, streamEvents, citations)
	conversationService := service.NewConversationService(conversations, messages, responseRuns, processSteps)
	server := httpx.NewServer(chatService, conversationService)

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("qa service listening", "addr", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown server", "error", err)
	}
}
