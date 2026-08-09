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
    authinfra "github.com/avito-hack/queue/services/queue/infrastructure/auth"
    transporthttp "github.com/avito-hack/queue/services/queue/infrastructure/transport/http"
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

    introspectionURL := cfg.Auth.IntrospectionURL
    if introspectionURL == "" {
        introspectionURL = "http://avito-adapter:8080/v1/users/validate"
    }

    sessionService := authinfra.NewSessionService(
        introspectionURL,
        cfg.Auth.Timeout,
        cfg.Auth.CacheTTL,
    )

    resolver := authinfra.NewSessionResolver(sessionService)

    handler := transporthttp.NewHandler(nil, nil)

    router, err := transporthttp.NewRouter(handler, resolver)
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
