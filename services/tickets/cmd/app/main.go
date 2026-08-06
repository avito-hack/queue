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
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/avito-hack/queue/services/tickets/config"
	"github.com/avito-hack/queue/services/tickets/infrastructure/client/avitoadapter"
	"github.com/avito-hack/queue/services/tickets/infrastructure/repository/postgresql"
	transporthttp "github.com/avito-hack/queue/services/tickets/infrastructure/transport/http"
	"github.com/avito-hack/queue/services/tickets/internal/usecase"
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

	databaseConfig, err := pgxpool.ParseConfig(cfg.PostgreSQL.URL)
	if err != nil {
		return fmt.Errorf("parse database config: %w", err)
	}
	database, err := pgxpool.NewWithConfig(databaseContext, databaseConfig)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer database.Close()
	if err := database.Ping(databaseContext); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	cancelDatabase()

	ticketRepository := postgresql.NewTicketRepository(database)
	listTickets := usecase.NewListTickets(ticketRepository, time.Now)
	getTicket := usecase.NewGetTicket(ticketRepository, time.Now)
	health := usecase.NewHealth(database)
	handler := transporthttp.NewHandler(health, listTickets, getTicket)
	tokenResolver, err := avitoadapter.NewUserTokenResolver(
		cfg.AvitoAdapter.URL,
		&http.Client{Timeout: cfg.AvitoAdapter.Timeout},
	)
	if err != nil {
		return fmt.Errorf("create Avito adapter user token resolver: %w", err)
	}
	router, err := transporthttp.NewRouter(handler, tokenResolver)
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
