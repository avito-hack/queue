package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTP             HTTPConfig
	PostgreSQL       PostgreSQLConfig
	AvitoAdapter     AvitoAdapterConfig
	RabbitMQ         RabbitMQConfig
	Ticket           TicketConfig
	Workers          WorkerConfig
	ServiceAuthToken string
}

type HTTPConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type PostgreSQLConfig struct {
	URL            string
	ConnectTimeout time.Duration
}

type AvitoAdapterConfig struct {
	URL     string
	Timeout time.Duration
}

type TicketConfig struct {
	ActivationTTL time.Duration
}

type RabbitMQConfig struct {
	URL      string
	Exchange string
}

type WorkerConfig struct {
	MaintenanceInterval       time.Duration
	BatchSize                 int
	ActivationRecoveryTimeout time.Duration
	OutboxInterval            time.Duration
	OutboxLease               time.Duration
	OutboxRetryDelay          time.Duration
	OutboxConcurrency         int
}

func Load() (Config, error) {
	port, err := intValue("HTTP_PORT", 8080)
	if err != nil {
		return Config{}, err
	}

	readTimeout, err := durationValue("HTTP_READ_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}

	writeTimeout, err := durationValue("HTTP_WRITE_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := durationValue("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	connectTimeout, err := durationValue("DATABASE_CONNECT_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}
	if connectTimeout <= 0 {
		return Config{}, fmt.Errorf("DATABASE_CONNECT_TIMEOUT must be positive")
	}
	avitoAdapterTimeout, err := durationValue("AVITO_ADAPTER_TIMEOUT", 3*time.Second)
	if err != nil {
		return Config{}, err
	}
	if avitoAdapterTimeout <= 0 {
		return Config{}, fmt.Errorf("AVITO_ADAPTER_TIMEOUT must be positive")
	}
	activationTTL, err := durationValue("TICKET_ACTIVATION_TTL", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}
	if activationTTL <= 0 {
		return Config{}, fmt.Errorf("TICKET_ACTIVATION_TTL must be positive")
	}
	maintenanceInterval, err := positiveDuration("TICKET_MAINTENANCE_INTERVAL", time.Second)
	if err != nil {
		return Config{}, err
	}
	activationRecoveryTimeout, err := positiveDuration("ACTIVATION_RECOVERY_TIMEOUT", time.Minute)
	if err != nil {
		return Config{}, err
	}
	outboxInterval, err := positiveDuration("OUTBOX_POLL_INTERVAL", 500*time.Millisecond)
	if err != nil {
		return Config{}, err
	}
	outboxLease, err := positiveDuration("OUTBOX_LEASE", 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	outboxRetryDelay, err := positiveDuration("OUTBOX_RETRY_DELAY", 5*time.Second)
	if err != nil {
		return Config{}, err
	}
	batchSize, err := positiveInt("WORKER_BATCH_SIZE", 100)
	if err != nil {
		return Config{}, err
	}
	outboxConcurrency, err := positiveInt("OUTBOX_CONCURRENCY", 4)
	if err != nil {
		return Config{}, err
	}
	databaseURL := value("DATABASE_URL", "")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	avitoAdapterURL := value("AVITO_ADAPTER_URL", "")
	if avitoAdapterURL == "" {
		return Config{}, fmt.Errorf("AVITO_ADAPTER_URL is required")
	}
	rabbitMQURL := value("RABBITMQ_URL", "")
	if rabbitMQURL == "" {
		return Config{}, fmt.Errorf("RABBITMQ_URL is required")
	}
	serviceAuthToken := value("SERVICE_AUTH_TOKEN", "")
	if serviceAuthToken == "" {
		return Config{}, fmt.Errorf("SERVICE_AUTH_TOKEN is required")
	}

	return Config{
		HTTP: HTTPConfig{
			Host:            value("HTTP_HOST", "0.0.0.0"),
			Port:            port,
			ReadTimeout:     readTimeout,
			WriteTimeout:    writeTimeout,
			ShutdownTimeout: shutdownTimeout,
		},
		PostgreSQL: PostgreSQLConfig{
			URL:            databaseURL,
			ConnectTimeout: connectTimeout,
		},
		AvitoAdapter: AvitoAdapterConfig{
			URL:     avitoAdapterURL,
			Timeout: avitoAdapterTimeout,
		},
		RabbitMQ: RabbitMQConfig{
			URL:      rabbitMQURL,
			Exchange: value("RABBITMQ_EXCHANGE", "domain.events"),
		},
		Ticket: TicketConfig{
			ActivationTTL: activationTTL,
		},
		Workers: WorkerConfig{
			MaintenanceInterval:       maintenanceInterval,
			BatchSize:                 batchSize,
			ActivationRecoveryTimeout: activationRecoveryTimeout,
			OutboxInterval:            outboxInterval,
			OutboxLease:               outboxLease,
			OutboxRetryDelay:          outboxRetryDelay,
			OutboxConcurrency:         outboxConcurrency,
		},
		ServiceAuthToken: serviceAuthToken,
	}, nil
}

func value(name, fallback string) string {
	if result, ok := os.LookupEnv(name); ok {
		return result
	}

	return fallback
}

func intValue(name string, fallback int) (int, error) {
	raw, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}

	result, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}

	return result, nil
}

func durationValue(name string, fallback time.Duration) (time.Duration, error) {
	raw, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}

	result, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}

	return result, nil
}

func positiveDuration(name string, fallback time.Duration) (time.Duration, error) {
	result, err := durationValue(name, fallback)
	if err != nil {
		return 0, err
	}
	if result <= 0 {
		return 0, fmt.Errorf("%s must be positive", name)
	}

	return result, nil
}

func positiveInt(name string, fallback int) (int, error) {
	result, err := intValue(name, fallback)
	if err != nil {
		return 0, err
	}
	if result <= 0 {
		return 0, fmt.Errorf("%s must be positive", name)
	}

	return result, nil
}
