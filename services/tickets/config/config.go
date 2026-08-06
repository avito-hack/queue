package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTP         HTTPConfig
	PostgreSQL   PostgreSQLConfig
	AvitoAdapter AvitoAdapterConfig
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

	databaseURL := value("DATABASE_URL", "")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	avitoAdapterURL := value("AVITO_ADAPTER_URL", "")
	if avitoAdapterURL == "" {
		return Config{}, fmt.Errorf("AVITO_ADAPTER_URL is required")
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
