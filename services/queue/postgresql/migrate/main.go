package main

import (
	"database/sql"
	"log/slog"
	"os"

	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	dsn := os.Getenv("DATABASE_URL")

	if dsn == "" {
		logger.Error(
			"DATABASE_URL is empty",
		)

		os.Exit(1)
	}

	logger.Info(
		"starting database migrations",
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		logger.Error(
			"failed to open database connection",
			"error",
			err,
		)

		os.Exit(1)
	}

	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		logger.Error(
			"failed to set goose dialect",
			"error",
			err,
		)

		os.Exit(1)
	}

	if err := goose.Up(db, "./migrations"); err != nil {
		logger.Error(
			"failed to apply migrations",
			"error",
			err,
		)

		os.Exit(1)
	}

	logger.Info(
		"database migrations completed successfully",
	)
}