package postgresql

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

var integrationPool *pgxpool.Pool

func TestMain(m *testing.M) {
	os.Exit(runIntegrationTests(m))
}

func Test_IssueRepository_ConcurrentSameQueueEntry_CreateOneTicket(t *testing.T) {
	// given
	truncateIntegrationTables(t)
	repository := NewIssueRepository(integrationPool)
	now := time.Date(2026, time.August, 7, 10, 0, 0, 0, time.UTC)
	command := usecase.IssueTicketCommand{
		QueueEntryID:       uuid.New(),
		UserID:             uuid.New(),
		ListingID:          uuid.New(),
		SKUID:              uuid.New(),
		IdempotencyKey:     uuid.New(),
		IssuedAt:           now,
		ActivationDeadline: now.Add(15 * time.Minute),
	}
	commands := []usecase.IssueTicketCommand{command, command}
	commands[1].IdempotencyKey = uuid.New()

	// when
	results := issueConcurrently(repository, commands)

	// then
	created := 0
	for _, result := range results {
		require.NoError(t, result.err)
		if result.value.Created {
			created++
		}
	}
	require.Equal(t, 1, created)
	require.Equal(t, results[0].value.Ticket.ID, results[1].value.Ticket.ID)
	require.Equal(t, 1, integrationRowCount(t, "public.tickets"))
	require.Equal(t, 1, integrationRowCount(t, "public.outbox_events"))
}

func Test_ActivationRepository_ConcurrentPrepare_CreateOneOperation(t *testing.T) {
	// given
	truncateIntegrationTables(t)
	issued := issueIntegrationTicket(t)
	repository := NewActivationRepository(integrationPool)
	now := issued.Ticket.IssuedAt.Add(time.Minute)
	command := usecase.PrepareActivationCommand{
		UserID:         issued.UserID,
		TicketID:       issued.Ticket.ID,
		IdempotencyKey: uuid.New(),
		Now:            now,
	}
	commands := []usecase.PrepareActivationCommand{command, command}
	commands[1].IdempotencyKey = uuid.New()

	// when
	results := prepareConcurrently(repository, commands)

	// then
	succeeded := 0
	inProgress := 0
	for _, result := range results {
		switch {
		case result.err == nil:
			succeeded++
			require.NotEqual(t, uuid.Nil, result.value.OperationID)
		case errors.Is(result.err, usecase.ErrActivationInProgress):
			inProgress++
		default:
			require.NoError(t, result.err)
		}
	}
	require.Equal(t, 1, succeeded)
	require.Equal(t, 1, inProgress)
	require.Equal(t, 1, integrationRowCountWhere(t, "public.idempotency_operations", "operation = 'activate_ticket'"))
}

func Test_ActivationRepository_ConcurrentComplete_PersistOneOrder(t *testing.T) {
	// given
	truncateIntegrationTables(t)
	issued := issueIntegrationTicket(t)
	repository := NewActivationRepository(integrationPool)
	prepared, err := repository.Prepare(context.Background(), usecase.PrepareActivationCommand{
		UserID:         issued.UserID,
		TicketID:       issued.Ticket.ID,
		IdempotencyKey: uuid.New(),
		Now:            issued.Ticket.IssuedAt.Add(time.Minute),
	})
	require.NoError(t, err)
	orders := []usecase.CreatedOrder{
		{ID: uuid.New(), CheckoutURL: "https://example.com/checkout/first"},
		{ID: uuid.New(), CheckoutURL: "https://example.com/checkout/second"},
	}

	// when
	results := completeConcurrently(repository, prepared.OperationID, orders, issued.Ticket.IssuedAt.Add(2*time.Minute))

	// then
	for _, result := range results {
		require.NoError(t, result.err)
	}
	require.Equal(t, results[0].value, results[1].value)
	require.Contains(t, []uuid.UUID{orders[0].ID, orders[1].ID}, results[0].value.OrderID)
	require.Equal(t, 1, integrationRowCountWhere(t, "public.outbox_events", "event_type = 'ticket.activated'"))

	var orderID uuid.UUID
	require.NoError(t, integrationPool.QueryRow(
		context.Background(),
		"SELECT order_id FROM public.tickets WHERE id = $1",
		issued.Ticket.ID,
	).Scan(&orderID))
	require.Equal(t, results[0].value.OrderID, orderID)
}

func Test_DeclineRepository_ConcurrentDecline_CloseTicketOnce(t *testing.T) {
	// given
	truncateIntegrationTables(t)
	issued := issueIntegrationTicket(t)
	repository := NewDeclineRepository(integrationPool)
	command := usecase.DeclineTicketCommand{
		UserID:         issued.UserID,
		TicketID:       issued.Ticket.ID,
		IdempotencyKey: uuid.New(),
		Now:            issued.Ticket.IssuedAt.Add(time.Minute),
	}
	commands := []usecase.DeclineTicketCommand{command, command}
	commands[1].IdempotencyKey = uuid.New()

	// when
	results := declineConcurrently(repository, commands)

	// then
	succeeded := 0
	notDeclinable := 0
	for _, result := range results {
		switch {
		case result.err == nil:
			succeeded++
		case errors.Is(result.err, usecase.ErrTicketNotDeclinable):
			notDeclinable++
		default:
			require.NoError(t, result.err)
		}
	}
	require.Equal(t, 1, succeeded)
	require.Equal(t, 1, notDeclinable)
	require.Equal(t, 1, integrationRowCountWhere(t, "public.outbox_events", "event_type = 'ticket.declined'"))
}

type integrationResult[T any] struct {
	value T
	err   error
}

func runIntegrationTests(m *testing.M) int {
	ctx := context.Background()
	migrations, err := filepath.Glob(filepath.Join("..", "..", "..", "postgresql", "migrations", "*.sql"))
	if err != nil || len(migrations) == 0 {
		fmt.Fprintf(os.Stderr, "find PostgreSQL migrations: %v\n", err)
		return 1
	}

	container, err := postgrescontainer.Run(
		ctx,
		"postgres:16-alpine",
		postgrescontainer.WithDatabase("tickets"),
		postgrescontainer.WithUsername("tickets"),
		postgrescontainer.WithPassword("password"),
		postgrescontainer.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start PostgreSQL container: %v\n", err)
		return 1
	}
	defer func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			fmt.Fprintf(os.Stderr, "terminate PostgreSQL container: %v\n", err)
		}
	}()

	connectionString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "get PostgreSQL connection string: %v\n", err)
		return 1
	}
	integrationPool, err = pgxpool.New(ctx, connectionString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create PostgreSQL pool: %v\n", err)
		return 1
	}
	defer integrationPool.Close()
	if err := integrationPool.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "ping PostgreSQL: %v\n", err)
		return 1
	}
	if err := applyIntegrationMigrations(ctx, migrations); err != nil {
		fmt.Fprintf(os.Stderr, "apply PostgreSQL migrations: %v\n", err)
		return 1
	}

	return m.Run()
}

func applyIntegrationMigrations(ctx context.Context, migrations []string) error {
	for _, migration := range migrations {
		contents, err := os.ReadFile(migration)
		if err != nil {
			return fmt.Errorf("read %s: %w", filepath.Base(migration), err)
		}
		up, _, ok := strings.Cut(string(contents), "-- +goose Down")
		if !ok {
			return fmt.Errorf("split %s: missing down marker", filepath.Base(migration))
		}
		if _, err := integrationPool.Exec(ctx, up); err != nil {
			return fmt.Errorf("execute %s: %w", filepath.Base(migration), err)
		}
	}

	return nil
}

func truncateIntegrationTables(t *testing.T) {
	t.Helper()
	_, err := integrationPool.Exec(
		context.Background(),
		"TRUNCATE public.outbox_events, public.idempotency_operations, public.tickets CASCADE",
	)
	require.NoError(t, err)
}

func issueIntegrationTicket(t *testing.T) usecase.IssueTicketResult {
	t.Helper()
	now := time.Date(2026, time.August, 7, 10, 0, 0, 0, time.UTC)
	result, err := NewIssueRepository(integrationPool).Issue(context.Background(), usecase.IssueTicketCommand{
		QueueEntryID:       uuid.New(),
		UserID:             uuid.New(),
		ListingID:          uuid.New(),
		SKUID:              uuid.New(),
		IdempotencyKey:     uuid.New(),
		IssuedAt:           now,
		ActivationDeadline: now.Add(15 * time.Minute),
	})
	require.NoError(t, err)

	return result
}

func issueConcurrently(
	repository *IssueRepository,
	commands []usecase.IssueTicketCommand,
) []integrationResult[usecase.IssueTicketResult] {
	results := make([]integrationResult[usecase.IssueTicketResult], len(commands))
	runConcurrently(len(commands), func(index int) {
		results[index].value, results[index].err = repository.Issue(context.Background(), commands[index])
	})

	return results
}

func prepareConcurrently(
	repository *ActivationRepository,
	commands []usecase.PrepareActivationCommand,
) []integrationResult[usecase.PreparedActivation] {
	results := make([]integrationResult[usecase.PreparedActivation], len(commands))
	runConcurrently(len(commands), func(index int) {
		results[index].value, results[index].err = repository.Prepare(context.Background(), commands[index])
	})

	return results
}

func completeConcurrently(
	repository *ActivationRepository,
	operationID uuid.UUID,
	orders []usecase.CreatedOrder,
	completedAt time.Time,
) []integrationResult[usecase.ActivationResult] {
	results := make([]integrationResult[usecase.ActivationResult], len(orders))
	runConcurrently(len(orders), func(index int) {
		results[index].value, results[index].err = repository.Complete(
			context.Background(),
			operationID,
			orders[index],
			completedAt,
		)
	})

	return results
}

func declineConcurrently(
	repository *DeclineRepository,
	commands []usecase.DeclineTicketCommand,
) []integrationResult[usecase.DeclineTicketResult] {
	results := make([]integrationResult[usecase.DeclineTicketResult], len(commands))
	runConcurrently(len(commands), func(index int) {
		results[index].value, results[index].err = repository.Decline(context.Background(), commands[index])
	})

	return results
}

func runConcurrently(count int, action func(int)) {
	start := make(chan struct{})
	var waitGroup sync.WaitGroup
	waitGroup.Add(count)
	for index := 0; index < count; index++ {
		go func() {
			defer waitGroup.Done()
			<-start
			action(index)
		}()
	}
	close(start)
	waitGroup.Wait()
}

func integrationRowCount(t *testing.T, table string) int {
	t.Helper()

	return integrationRowCountWhere(t, table, "TRUE")
}

func integrationRowCountWhere(t *testing.T, table, condition string) int {
	t.Helper()
	var count int
	require.NoError(t, integrationPool.QueryRow(
		context.Background(),
		fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", table, condition),
	).Scan(&count))

	return count
}
