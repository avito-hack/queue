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

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/avito-hack/queue/services/avito-adapter/config"
	"github.com/avito-hack/queue/services/avito-adapter/infrastructure/repository/postgresql"
	transporthttp "github.com/avito-hack/queue/services/avito-adapter/infrastructure/transport/http"
	"github.com/avito-hack/queue/services/avito-adapter/internal/usecase"
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

	databaseContext, cancelDatabase := context.WithTimeout(context.Background(), cfg.PostgreSQL.ConnectTimeout)
	defer cancelDatabase()
	database, err := pgxpool.New(databaseContext, cfg.PostgreSQL.URL)
	if err != nil {
		return fmt.Errorf("create postgres pool: %w", err)
	}
	defer database.Close()
	if err := database.Ping(databaseContext); err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	cancelDatabase()

	health := usecase.NewHealth(database)
	service := usecase.NewService(postgresql.NewReader(database))
	handler := transporthttp.NewHandler(health, service)
	router, err := transporthttp.NewRouter(handler)
	if err != nil {
		return fmt.Errorf("create router: %w", err)
	}

	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	serverError := make(chan error, 1)
	go func() {
		slog.Info("HTTP server started", "address", httpServer.Addr)
		serverError <- httpServer.ListenAndServe()
	}()

	signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-signalContext.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
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
