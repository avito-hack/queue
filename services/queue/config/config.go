package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTP      HTTPConfig
	Postgres  PostgresConfig
	Auth      AuthConfig
	Services  ServicesConfig
	RabbitMQ  RabbitMQConfig
	Workers   WorkersConfig
}

type HTTPConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type PostgresConfig struct {
	Host           string
	Port           int
	DB             string
	User           string
	Password       string
	MaxConnections int32
	AutoMigrate    bool
	MigrationsDir  string
}

type AuthConfig struct {
	IntrospectionURL string
	Timeout          time.Duration
	CacheTTL         time.Duration
}

type ServicesConfig struct {
	AvitoBaseURL   string
	TicketsBaseURL string
}

type RabbitMQConfig struct {
	URL      string
	Exchange string
	Queue    string
}

type WorkersConfig struct {
	BatchSize int
}

func Load() (Config, error) {
	httpPort, err := intValue("HTTP_PORT", 8080)
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

	maxConnections, err := intValue("POSTGRES_MAX_CONNECTIONS", 10)
	if err != nil {
		return Config{}, err
	}

	autoMigrate, err := boolValue("POSTGRES_AUTO_MIGRATE", false)
	if err != nil {
		return Config{}, err
	}

	dbURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok || strings.TrimSpace(dbURL) == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	host, portParsed, dbName, dbUser, dbPass, _, err := parseDatabaseURL(dbURL)
	if err != nil {
		return Config{}, fmt.Errorf("invalid DATABASE_URL: %w", err)
	}

	if host == "" || dbName == "" || dbUser == "" {
		return Config{}, errors.New("DATABASE_URL must include host, db name and user")
	}

	port := 5432

	if portParsed != 0 {
		port = portParsed
	}

	authTimeout, err := durationValue(
		"AUTH_TIMEOUT",
		2*time.Second,
	)
	if err != nil {
		return Config{}, err
	}

	authCacheTTL, err := durationValue(
		"AUTH_CACHE_TTL",
		30*time.Second,
	)
	if err != nil {
		return Config{}, err
	}

	batchSize, err := intValue(
		"WORKERS_BATCH_SIZE",
		100,
	)
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTP: HTTPConfig{
			Host:            value("HTTP_HOST", "0.0.0.0"),
			Port:            httpPort,
			ReadTimeout:     readTimeout,
			WriteTimeout:    writeTimeout,
			ShutdownTimeout: shutdownTimeout,
		},

		Postgres: PostgresConfig{
			Host:           host,
			Port:           port,
			DB:             dbName,
			User:           dbUser,
			Password:       dbPass,
			MaxConnections: int32(maxConnections),
			AutoMigrate:    autoMigrate,
			MigrationsDir: value(
				"POSTGRES_MIGRATIONS_DIR",
				"postgresql/migrations",
			),
		},

		Auth: AuthConfig{
			IntrospectionURL: value(
				"AUTH_INTROSPECTION_URL",
				"http://avito-adapter:8080/v1/users/validate",
			),
			Timeout:  authTimeout,
			CacheTTL: authCacheTTL,
		},

		Services: ServicesConfig{
			AvitoBaseURL: value(
				"AVITO_BASE_URL",
				"http://avito-adapter:8080",
			),
			TicketsBaseURL: value(
				"TICKETS_BASE_URL",
				"http://tickets:8080",
			),
		},

		RabbitMQ: RabbitMQConfig{
			URL: value(
				"RABBITMQ_URL",
				"amqp://tickets:tickets@rabbitmq:5672/",
			),
			Exchange: value(
				"RABBITMQ_EXCHANGE",
				"domain.events",
			),
			Queue: value(
				"RABBITMQ_QUEUE",
				"queue.service.events",
			),
		},

		Workers: WorkersConfig{
			BatchSize: batchSize,
		},
	}, nil
}

func parseDatabaseURL(raw string) (
	host string,
	port int,
	db string,
	user string,
	password string,
	sslmode string,
	err error,
) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", 0, "", "", "", "", err
	}

	if u.User != nil {
		user = u.User.Username()

		if pw, ok := u.User.Password(); ok {
			password = pw
		}
	}

	hostPart := u.Host

	if strings.Contains(hostPart, ":") {
		parts := strings.Split(hostPart, ":")

		host = parts[0]

		if p, err2 := strconv.Atoi(parts[1]); err2 == nil {
			port = p
		}
	} else {
		host = hostPart
	}

	if u.Path != "" {
		db = strings.TrimPrefix(u.Path, "/")
	}

	q := u.Query()
	sslmode = q.Get("sslmode")

	return host, port, db, user, password, sslmode, nil
}

func value(name, fallback string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}

	return fallback
}

func intValue(name string, fallback int) (int, error) {
	raw, ok := os.LookupEnv(name)

	if !ok || strings.TrimSpace(raw) == "" {
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

	if !ok || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}

	result, err := time.ParseDuration(raw)

	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}

	return result, nil
}

func boolValue(name string, fallback bool) (bool, error) {
	raw, ok := os.LookupEnv(name)

	if !ok || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}

	result, err := strconv.ParseBool(raw)

	if err != nil {
		return false, fmt.Errorf("parse %s: %w", name, err)
	}

	return result, nil
}