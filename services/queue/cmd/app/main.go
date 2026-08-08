package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/avito-hack/queue/services/queue/config"
	avitoclient "github.com/avito-hack/queue/services/queue/gen/clients/avito"
	ticketsclient "github.com/avito-hack/queue/services/queue/gen/clients/tickets"
	"github.com/avito-hack/queue/services/queue/infrastructure/auth"
	"github.com/avito-hack/queue/services/queue/infrastructure/repository/postgres"
	transporthttp "github.com/avito-hack/queue/services/queue/infrastructure/transport/http"
	"github.com/avito-hack/queue/services/queue/internal/usecase"
	"github.com/avito-hack/queue/services/queue/postgresql/sqlc"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx := context.Background()

	if cfg.Postgres.AutoMigrate {
		if err := postgres.ApplyMigrations(ctx, cfg.Postgres); err != nil {
			return fmt.Errorf("apply postgres migrations: %w", err)
		}
	}

	pool, err := postgres.NewPool(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("create postgres pool: %w", err)
	}

	defer pool.Close()

	queries := sqlc.New(pool)

	queueRepository := postgres.NewItemQueueRepository(queries)
	memberRepository := postgres.NewItemQueueMemberRepository(queries)

	transactionManager := postgres.NewTransactionManager(pool)

	avitoClient, err := avitoclient.NewClient(cfg.Services.AvitoBaseURL)
	if err != nil {
		return fmt.Errorf("create avito client: %w", err)
	}

	ticketsClient, err := ticketsclient.NewClient(cfg.Services.TicketsBaseURL)
	if err != nil {
		return fmt.Errorf("create tickets client: %w", err)
	}

	queueService := usecase.NewItemQueueService(
		queueRepository,
		memberRepository,
		transactionManager,
		avitoClient,
		ticketsClient,
	)

	healthChecker := usecase.NewHealth()

	handler := transporthttp.NewHandler(
		healthChecker,
		queueService,
	)

	jwtService := auth.NewJWTService(
		cfg.Auth.JWTSecret,
	)

	router, err := transporthttp.NewRouter(
		handler,
		jwtService,
	)
	if err != nil {
		return fmt.Errorf("create router: %w", err)
	}

	httpServer := &http.Server{
		Addr: fmt.Sprintf(
			"%s:%d",
			cfg.HTTP.Host,
			cfg.HTTP.Port,
		),
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	serverError := make(chan error, 1)

	go func() {
		slog.Info(
			"HTTP server started",
			"address",
			httpServer.Addr,
		)

		serverError <- httpServer.ListenAndServe()
	}()

	signalContext, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	defer stop()

	select {
	case <-signalContext.Done():
		shutdownContext, cancel := context.WithTimeout(
			context.Background(),
			cfg.HTTP.ShutdownTimeout,
		)

		defer cancel()

		if err := httpServer.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}

		return nil

	case err := <-serverError:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return fmt.Errorf("serve HTTP: %w", err)
	}
}
