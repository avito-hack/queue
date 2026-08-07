package config

import (
    "fmt"
    "os"
    "strconv"
    "time"
)

type Config struct {
    HTTP     HTTPConfig
    Postgres PostgresConfig
    Auth     AuthConfig
    Services ServicesConfig
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
    JWTSecret string
}

type ServicesConfig struct {
    AvitoBaseURL   string
    TicketsBaseURL string
}

func Load() (Config, error) {
    httpPort, err := intValue("HTTP_PORT", 8080)
    if err != nil {
        return Config{}, err
    }

    postgresPort, err := intValue("POSTGRES_PORT", 5432)
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

    return Config{
        HTTP: HTTPConfig{
            Host:            value("HTTP_HOST", "0.0.0.0"),
            Port:            httpPort,
            ReadTimeout:     readTimeout,
            WriteTimeout:    writeTimeout,
            ShutdownTimeout: shutdownTimeout,
        },
        Postgres: PostgresConfig{
            Host:           value("POSTGRES_HOST", "localhost"),
            Port:           postgresPort,
            DB:             value("QUEUE_POSTGRES_DB", "queue"),
            User:           value("QUEUE_POSTGRES_USER", "queue"),
            Password:       value("QUEUE_POSTGRES_PASSWORD", ""),
            MaxConnections: int32(maxConnections),
            AutoMigrate:    autoMigrate,
            MigrationsDir:  value("POSTGRES_MIGRATIONS_DIR", "postgresql/migrations"),
        },
        Auth: AuthConfig{
            JWTSecret: value("JWT_SECRET", ""),
        },
        Services: ServicesConfig{
            AvitoBaseURL:   value("AVITO_BASE_URL", "http://avito-adapter:8080"),
            TicketsBaseURL: value("TICKETS_BASE_URL", value("TICKETS_URL", "http://tickets:8080")),
        },
    }, nil
}

func value(name, fallback string) string {
    result, ok := os.LookupEnv(name)
    if ok {
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

func boolValue(name string, fallback bool) (bool, error) {
    raw, ok := os.LookupEnv(name)
    if !ok {
        return fallback, nil
    }

    result, err := strconv.ParseBool(raw)
    if err != nil {
        return false, fmt.Errorf("parse %s: %w", name, err)
    }

    return result, nil
}
