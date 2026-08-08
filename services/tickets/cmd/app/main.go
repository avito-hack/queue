package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/avito-hack/queue/services/tickets/config"
	"github.com/avito-hack/queue/services/tickets/infrastructure/client/avitoadapter"
	brokerrabbit "github.com/avito-hack/queue/services/tickets/infrastructure/messaging/rabbitmq"
	"github.com/avito-hack/queue/services/tickets/infrastructure/repository/postgresql"
	transporthttp "github.com/avito-hack/queue/services/tickets/infrastructure/transport/http"
	"github.com/avito-hack/queue/services/tickets/internal/usecase"
	"github.com/avito-hack/queue/services/tickets/internal/worker"
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
	activationRepository := postgresql.NewActivationRepository(database)
	declineRepository := postgresql.NewDeclineRepository(database)
	issueRepository := postgresql.NewIssueRepository(database)
	lifecycleRepository := postgresql.NewLifecycleRepository(database)
	listingEventRepository := postgresql.NewListingEventRepository(database)
	outboxRepository := postgresql.NewOutboxRepository(database)
	listTickets := usecase.NewListTickets(ticketRepository, time.Now)
	getTicket := usecase.NewGetTicket(ticketRepository, time.Now)
	health := usecase.NewHealth(database)
	avitoHTTPClient := &http.Client{Timeout: cfg.AvitoAdapter.Timeout}
	tokenResolver, err := avitoadapter.NewUserTokenResolver(
		cfg.AvitoAdapter.URL,
		avitoHTTPClient,
	)
	if err != nil {
		return fmt.Errorf("create Avito adapter user token resolver: %w", err)
	}
	orderCreator, err := avitoadapter.NewOrderCreator(cfg.AvitoAdapter.URL, avitoHTTPClient)
	if err != nil {
		return fmt.Errorf("create Avito adapter order creator: %w", err)
	}
	activateTicket := usecase.NewActivateTicket(activationRepository, orderCreator, time.Now)
	declineTicket := usecase.NewDeclineTicket(declineRepository, time.Now)
	issueTicket := usecase.NewIssueTicket(issueRepository, cfg.Ticket.ActivationTTL, time.Now)
	maintainTickets := usecase.NewMaintainTickets(
		lifecycleRepository,
		cfg.Workers.BatchSize,
		cfg.Workers.ActivationRecoveryTimeout,
		time.Now,
	)
	handleListingEvents := usecase.NewHandleListingEvents(listingEventRepository, time.Now)
	rabbitConnection, err := amqp.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		return fmt.Errorf("connect to RabbitMQ: %w", err)
	}
	defer func() { _ = rabbitConnection.Close() }()
	consumer, err := brokerrabbit.NewConsumer(
		rabbitConnection,
		cfg.RabbitMQ.Exchange,
		cfg.RabbitMQ.Queue,
		cfg.Workers.BatchSize,
		handleListingEvents,
	)
	if err != nil {
		return fmt.Errorf("create listing events consumer: %w", err)
	}
	defer func() { _ = consumer.Close() }()
	publisher, err := brokerrabbit.NewPublisher(rabbitConnection, cfg.RabbitMQ.Exchange)
	if err != nil {
		return fmt.Errorf("create outbox publisher: %w", err)
	}
	defer func() { _ = publisher.Close() }()
	maintenanceWorker := worker.NewMaintenance(maintainTickets, cfg.Workers.MaintenanceInterval)
	outboxWorker := worker.NewOutbox(
		outboxRepository,
		publisher,
		cfg.Workers.OutboxInterval,
		cfg.Workers.OutboxLease,
		cfg.Workers.OutboxRetryDelay,
		cfg.Workers.OutboxConcurrency,
		cfg.Workers.BatchSize,
		time.Now,
	)
	handler := transporthttp.NewHandler(health, listTickets, getTicket, activateTicket, declineTicket, issueTicket)
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

	signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverError := make(chan error, 1)
	go func() {
		slog.Info("HTTP server started", "address", httpServer.Addr)
		serverError <- httpServer.ListenAndServe()
	}()

	workerError := make(chan error, 3)
	var workerWaitGroup sync.WaitGroup
	startWorker := func(name string, run func(context.Context) error) {
		workerWaitGroup.Add(1)
		go func() {
			defer workerWaitGroup.Done()
			if err := run(signalContext); err != nil {
				workerError <- fmt.Errorf("%s worker: %w", name, err)
				return
			}
			if signalContext.Err() == nil {
				workerError <- fmt.Errorf("%s worker stopped unexpectedly", name)
			}
		}()
	}
	startWorker("maintenance", maintenanceWorker.Run)
	startWorker("outbox", outboxWorker.Run)
	startWorker("listing events", consumer.Run)

	var runError error
	select {
	case <-signalContext.Done():
	case err := <-serverError:
		if !errors.Is(err, http.ErrServerClosed) {
			runError = fmt.Errorf("serve HTTP: %w", err)
		}
	case err := <-workerError:
		runError = err
	}
	stop()

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancelShutdown()
	if err := httpServer.Shutdown(shutdownContext); err != nil {
		runError = errors.Join(runError, fmt.Errorf("shutdown HTTP server: %w", err))
	}
	workersStopped := make(chan struct{})
	go func() {
		workerWaitGroup.Wait()
		close(workersStopped)
	}()
	select {
	case <-workersStopped:
	case <-shutdownContext.Done():
		runError = errors.Join(runError, fmt.Errorf("shutdown workers: %w", shutdownContext.Err()))
	}

	return runError
}
