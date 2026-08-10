package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTP       HTTPConfig
	PostgreSQL PostgreSQLConfig
	RabbitMQ   RabbitMQConfig
}

type RabbitMQConfig struct{ URL, Exchange string }

type PostgreSQLConfig struct {
	URL            string
	ConnectTimeout time.Duration
}

type HTTPConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
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

	connectTimeout, err := durationValue("POSTGRES_CONNECT_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	databaseURL := value("DATABASE_URL", "")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	rabbitMQURL := value("RABBITMQ_URL", "")
	if rabbitMQURL == "" {
		return Config{}, fmt.Errorf("RABBITMQ_URL is required")
	}

	return Config{HTTP: HTTPConfig{
		Host:            value("HTTP_HOST", "0.0.0.0"),
		Port:            port,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		ShutdownTimeout: shutdownTimeout,
	}, PostgreSQL: PostgreSQLConfig{URL: databaseURL, ConnectTimeout: connectTimeout}, RabbitMQ: RabbitMQConfig{URL: rabbitMQURL, Exchange: value("RABBITMQ_EXCHANGE", "domain.events")}}, nil
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
