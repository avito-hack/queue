package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/avito-hack/queue/services/queue/config"

	avitogen "github.com/avito-hack/queue/services/queue/gen/clients/avito"
	ticketsgen "github.com/avito-hack/queue/services/queue/gen/clients/tickets"

	authinfra "github.com/avito-hack/queue/services/queue/infrastructure/auth"
	avitoclient "github.com/avito-hack/queue/services/queue/infrastructure/client/avitoadapter"
	ticketsclient "github.com/avito-hack/queue/services/queue/infrastructure/client/tickets"
	"github.com/avito-hack/queue/services/queue/infrastructure/repository/postgres"
	transporthttp "github.com/avito-hack/queue/services/queue/infrastructure/transport/http"

	"github.com/avito-hack/queue/services/queue/internal/usecase"

	"github.com/avito-hack/queue/services/queue/postgresql/sqlc"
)

func main() {
	logger := slog.New(
		slog.NewJSONHandler(
			os.Stdout,
			nil,
		),
	)

	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}


	ctx := context.Background()


	pool, err := postgres.NewPool(
		ctx,
		cfg.Postgres,
		logger,
	)

	if err != nil {
		return fmt.Errorf("create postgres pool: %w", err)
	}

	defer pool.Close()


	queries := sqlc.New(pool)


	queueRepository := postgres.NewItemQueueRepository(
		queries,
		logger,
	)

	memberRepository := postgres.NewItemQueueMemberRepository(
		queries,
		logger,
	)

	txManager := postgres.NewTransactionManager(
		pool,
		logger,
	)


	avitoAPI, err := avitogen.NewClient(
		cfg.Services.AvitoBaseURL,
	)

	if err != nil {
		return fmt.Errorf("create avito client: %w", err)
	}

	avito := avitoclient.NewClient(
		avitoAPI,
		logger,
	)


	ticketsAPI, err := ticketsgen.NewClient(
		cfg.Services.TicketsBaseURL,
	)

	if err != nil {
		return fmt.Errorf("create tickets client: %w", err)
	}

	tickets := ticketsclient.NewClient(
		ticketsAPI,
		logger,
	)


	service := usecase.NewItemQueueService(
		queueRepository,
		memberRepository,
		txManager,
		avito,
		tickets,
		logger,
	)


	introspectionURL := cfg.Auth.IntrospectionURL
	if introspectionURL == "" {
		introspectionURL = "http://avito-adapter:8080/v1/users/validate"
	}

	sessionService := authinfra.NewSessionService(
		introspectionURL,
		cfg.Auth.Timeout,
		cfg.Auth.CacheTTL,
		logger,
	)

	resolver := authinfra.NewSessionResolver(
		sessionService,
		logger,
	)


	health := usecase.NewHealth()

	handler := transporthttp.NewHandler(
		health,
		service,
		logger,
	)

	router, err := transporthttp.NewRouter(handler, resolver, logger)
	if err != nil {
		return fmt.Errorf("create router: %w", err)
	}

	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("HTTP server started", "address", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen error", "error", err)
		}
	}()

	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	<-signalContext.Done()

	shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}

	slog.Info("HTTP server stopped gracefully")

	return nil
}