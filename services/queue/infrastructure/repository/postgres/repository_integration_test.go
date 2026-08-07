//go:build integration

package postgres

import (
	"context"
	"database/sql"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/avito-hack/queue/services/queue/internal/domain"
	"github.com/avito-hack/queue/services/queue/postgresql/sqlc"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Test_ItemQueueRepositories_Integration_CRUD(t *testing.T) {
	// given
	ctx := context.Background()

	container, err := postgrescontainer.Run(
		ctx,
		"postgres:16-alpine",
		postgrescontainer.WithDatabase("queue"),
		postgrescontainer.WithUsername("queue"),
		postgrescontainer.WithPassword("queue"),
	)
	if err != nil {
		t.Skipf("skip integration test: cannot start postgres container: %v", err)
	}
	defer func() {
		_ = container.Terminate(ctx)
	}()

	databaseURL, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	applyMigrations(t, databaseURL)

	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	defer pool.Close()

	queries := sqlc.New(pool)
	queueRepository := NewItemQueueRepository(queries)
	memberRepository := NewItemQueueMemberRepository(queries)

	itemID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	err = queueRepository.Create(ctx, &domain.ItemQueue{
		ItemID:    itemID,
		State:     domain.QueueTicketsAvailable,
		CreatedAt: now,
		UpdatedAt: now,
	})
	require.NoError(t, err)

	err = memberRepository.Create(ctx, itemID, &domain.ItemQueueMember{
		ItemID:    itemID,
		UserID:    userID,
		Position:  1,
		Status:    domain.UserWaitingInLine,
		CreatedAt: now,
	})
	require.NoError(t, err)

	// when
	exists, err := queueRepository.Exists(ctx, itemID)
	require.NoError(t, err)

	member, err := memberRepository.GetByUserID(ctx, itemID, userID)
	require.NoError(t, err)

	position, err := memberRepository.GetPosition(ctx, itemID, userID)
	require.NoError(t, err)

	// then
	assert.True(t, exists)
	assert.Equal(t, itemID, member.ItemID)
	assert.Equal(t, userID, member.UserID)
	assert.Equal(t, uint(1), position)
}

func applyMigrations(t *testing.T, databaseURL string) {
	t.Helper()

	db, err := sql.Open("pgx", databaseURL)
	require.NoError(t, err)
	defer db.Close()

	require.NoError(t, goose.SetDialect("postgres"))

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	migrationsDir := filepath.Join(filepath.Dir(thisFile), "../../../postgresql/migrations")

	require.NoError(t, goose.Up(db, migrationsDir))
}
