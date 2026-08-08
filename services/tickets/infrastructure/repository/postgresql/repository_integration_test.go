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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"

	sqlgen "github.com/avito-hack/queue/services/tickets/gen/sql"
	"github.com/avito-hack/queue/services/tickets/internal/domain"
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
	require.Equal(t, int64(1), integrationTicketCount(t))
	require.Zero(t, integrationOutboxCount(t))
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
	require.Equal(t, int64(1), integrationOperationCount(t, activationOperation))
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
	require.Equal(t, int64(1), integrationOutboxTypeCount(t, domain.TicketEventRedeemed))

	orderID, err := sqlgen.New(integrationPool).GetTicketOrderID(context.Background(), toPGUUID(issued.Ticket.ID))
	require.NoError(t, err)
	require.True(t, orderID.Valid)
	require.Equal(t, results[0].value.OrderID, uuid.UUID(orderID.Bytes))
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
	require.Equal(t, int64(1), integrationOutboxTypeCount(t, domain.TicketEventClosed))
}

func Test_LifecycleRepository_ExpiredIssued_CloseAndPublishOnce(t *testing.T) {
	// given
	truncateIntegrationTables(t)
	issued := issueIntegrationTicket(t)
	repository := NewLifecycleRepository(integrationPool)
	expiredAt := issued.Ticket.ActivationDeadline.Add(time.Second)

	// when
	expired, err := repository.ExpireIssued(context.Background(), expiredAt, 10)
	replayed, replayErr := repository.ExpireIssued(context.Background(), expiredAt, 10)

	// then
	require.NoError(t, err)
	require.NoError(t, replayErr)
	require.Equal(t, 1, expired)
	require.Zero(t, replayed)
	ticket, err := NewTicketRepository(integrationPool).Get(context.Background(), issued.UserID, issued.Ticket.ID)
	require.NoError(t, err)
	require.Equal(t, domain.TicketStatusClosed, ticket.Status)
	require.NotNil(t, ticket.CloseReason)
	require.Equal(t, domain.TicketCloseReasonActivationTimeout, *ticket.CloseReason)
	require.NotNil(t, ticket.FinishedAt)
	require.Equal(t, int64(1), integrationOutboxTypeCount(t, domain.TicketEventClosed))
}

func Test_LifecycleRepository_StaleActivation_RecoverForRetryAndDecline(t *testing.T) {
	// given
	truncateIntegrationTables(t)
	issued := issueIntegrationTicket(t)
	activationRepository := NewActivationRepository(integrationPool)
	command := usecase.PrepareActivationCommand{
		UserID:         issued.UserID,
		TicketID:       issued.Ticket.ID,
		IdempotencyKey: uuid.New(),
		Now:            issued.Ticket.IssuedAt.Add(time.Minute),
	}
	prepared, err := activationRepository.Prepare(context.Background(), command)
	require.NoError(t, err)
	repository := NewLifecycleRepository(integrationPool)

	// when
	recovered, err := repository.RecoverStaleActivations(context.Background(), command.Now.Add(time.Second), 10)
	retried, retryErr := activationRepository.Prepare(context.Background(), usecase.PrepareActivationCommand{
		UserID:         command.UserID,
		TicketID:       command.TicketID,
		IdempotencyKey: command.IdempotencyKey,
		Now:            command.Now.Add(time.Minute),
	})
	secondRecovered, secondRecoveryErr := repository.RecoverStaleActivations(
		context.Background(),
		time.Date(2100, time.January, 1, 0, 0, 0, 0, time.UTC),
		10,
	)
	declined, declineErr := NewDeclineRepository(integrationPool).Decline(context.Background(), usecase.DeclineTicketCommand{
		UserID:         command.UserID,
		TicketID:       command.TicketID,
		IdempotencyKey: uuid.New(),
		Now:            command.Now.Add(2 * time.Minute),
	})

	// then
	require.NoError(t, err)
	require.Equal(t, 1, recovered)
	require.NoError(t, retryErr)
	require.Equal(t, prepared.OperationID, retried.OperationID)
	require.NoError(t, secondRecoveryErr)
	require.Equal(t, 1, secondRecovered)
	require.NoError(t, declineErr)
	require.Equal(t, domain.TicketStatusClosed, declined.Status)
}

func Test_ListingEventRepository_QuantityDecreased_CloseOldestExcessOnce(t *testing.T) {
	// given
	truncateIntegrationTables(t)
	listingID := uuid.New()
	issuedAt := time.Date(2026, time.August, 9, 10, 0, 0, 0, time.UTC)
	ticketIDs := make([]uuid.UUID, 0, 4)
	for index := range 4 {
		result := issueListingIntegrationTicket(t, listingID, issuedAt.Add(time.Duration(index)*time.Minute))
		ticketIDs = append(ticketIDs, result.Ticket.ID)
	}
	eventID := uuid.New()
	command := usecase.RevokeListingTicketsCommand{
		Event: usecase.IncomingListingEvent{
			ID:      eventID,
			Type:    domain.ListingEventQuantityChanged,
			Source:  "avito-adapter",
			Payload: []byte(`{"quantity":2}`),
		},
		ListingID:       listingID,
		RevocationLimit: 2,
		CloseReason:     domain.TicketCloseReasonSystemCancelled,
		HandledAt:       issuedAt.Add(10 * time.Minute),
	}
	repository := NewListingEventRepository(integrationPool)

	// when
	revoked, err := repository.RevokeListingTickets(context.Background(), command)
	replayed, replayErr := repository.RevokeListingTickets(context.Background(), command)

	// then
	require.NoError(t, err)
	require.NoError(t, replayErr)
	require.Equal(t, 2, revoked)
	require.Zero(t, replayed)
	states, err := sqlgen.New(integrationPool).ListListingTicketStates(context.Background(), toPGUUID(listingID))
	require.NoError(t, err)
	require.Len(t, states, 4)
	for index, state := range states {
		require.True(t, state.ID.Valid)
		assert.Equal(t, ticketIDs[index], uuid.UUID(state.ID.Bytes))
		if index < 2 {
			assert.Equal(t, string(domain.TicketStatusClosed), state.Status)
			require.True(t, state.CloseReason.Valid)
			assert.Equal(t, string(domain.TicketCloseReasonSystemCancelled), state.CloseReason.String)
			continue
		}
		assert.Equal(t, string(domain.TicketStatusIssued), state.Status)
		assert.False(t, state.CloseReason.Valid)
	}
	assert.Equal(t, int64(1), integrationInboxCount(t))
	assert.Equal(t, int64(2), integrationOutboxTypeCount(t, domain.TicketEventClosed))
}

func Test_ListingEventRepository_StatusChanged_KeepTicketBeingActivated(t *testing.T) {
	// given
	truncateIntegrationTables(t)
	listingID := uuid.New()
	issuedAt := time.Date(2026, time.August, 9, 10, 0, 0, 0, time.UTC)
	activating := issueListingIntegrationTicket(t, listingID, issuedAt)
	revocable := issueListingIntegrationTicket(t, listingID, issuedAt.Add(time.Minute))
	_, err := NewActivationRepository(integrationPool).Prepare(context.Background(), usecase.PrepareActivationCommand{
		UserID:         activating.UserID,
		TicketID:       activating.Ticket.ID,
		IdempotencyKey: uuid.New(),
		Now:            issuedAt.Add(2 * time.Minute),
	})
	require.NoError(t, err)
	command := usecase.RevokeListingTicketsCommand{
		Event: usecase.IncomingListingEvent{
			ID:      uuid.New(),
			Type:    domain.ListingEventStatusChanged,
			Source:  "avito-adapter",
			Payload: []byte(`{"status":"paused"}`),
		},
		ListingID:   listingID,
		CloseAll:    true,
		CloseReason: domain.TicketCloseReasonListingClosed,
		HandledAt:   issuedAt.Add(3 * time.Minute),
	}

	// when
	revoked, err := NewListingEventRepository(integrationPool).RevokeListingTickets(context.Background(), command)

	// then
	require.NoError(t, err)
	require.Equal(t, 1, revoked)
	activatingTicket, err := NewTicketRepository(integrationPool).Get(
		context.Background(),
		activating.UserID,
		activating.Ticket.ID,
	)
	require.NoError(t, err)
	assert.Equal(t, domain.TicketStatusIssued, activatingTicket.Status)
	revokedTicket, err := NewTicketRepository(integrationPool).Get(
		context.Background(),
		revocable.UserID,
		revocable.Ticket.ID,
	)
	require.NoError(t, err)
	assert.Equal(t, domain.TicketStatusClosed, revokedTicket.Status)
	require.NotNil(t, revokedTicket.CloseReason)
	assert.Equal(t, domain.TicketCloseReasonListingClosed, *revokedTicket.CloseReason)
}

func Test_OutboxRepository_ClaimRetryAndPublish_ChangeDeliveryState(t *testing.T) {
	// given
	truncateIntegrationTables(t)
	issued := issueIntegrationTicket(t)
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	expired, err := NewLifecycleRepository(integrationPool).ExpireIssued(
		context.Background(),
		issued.Ticket.ActivationDeadline.Add(time.Second),
		1,
	)
	require.NoError(t, err)
	require.Equal(t, 1, expired)
	repository := NewOutboxRepository(integrationPool)

	// when
	claimed, err := repository.Claim(context.Background(), now, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	err = repository.Retry(context.Background(), claimed[0].ID, now.Add(time.Minute))
	require.NoError(t, err)
	beforeRetry, err := repository.Claim(context.Background(), now, 10, time.Minute)
	require.NoError(t, err)
	retried, err := repository.Claim(context.Background(), now.Add(time.Minute), 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, retried, 1)
	err = repository.MarkPublished(context.Background(), retried[0].ID, now.Add(2*time.Minute))

	// then
	require.NoError(t, err)
	require.Empty(t, beforeRetry)
	require.Equal(t, 1, claimed[0].Attempts)
	require.Equal(t, 2, retried[0].Attempts)
	require.Equal(t, int64(1), integrationOutboxStatusCount(t, "published"))
}

func Test_TicketsSchema_ActiveStatus_RejectObsoleteLifecycle(t *testing.T) {
	// given
	truncateIntegrationTables(t)
	issued := issueIntegrationTicket(t)

	// when
	err := sqlgen.New(integrationPool).SetTicketObsoleteStatus(context.Background(), toPGUUID(issued.Ticket.ID))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ck_tickets_")
}

func Test_TicketsSchema_ExternalCloseReason_RejectUnknownReason(t *testing.T) {
	// given
	truncateIntegrationTables(t)
	issued := issueIntegrationTicket(t)

	// when
	err := sqlgen.New(integrationPool).CloseTicketWithUnknownReason(context.Background(), toPGUUID(issued.Ticket.ID))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ck_tickets_close_reason")
}

func Test_TicketsSchema_InboxTable_ReturnPresent(t *testing.T) {
	// given
	// when
	exists, err := sqlgen.New(integrationPool).InboxTableExists(context.Background())

	// then
	require.NoError(t, err)
	assert.True(t, exists)
}

func Test_OutboxSchema_UnknownEventType_RejectEvent(t *testing.T) {
	// given
	truncateIntegrationTables(t)

	// when
	err := sqlgen.New(integrationPool).InsertOutboxEventWithType(
		context.Background(),
		sqlgen.InsertOutboxEventWithTypeParams{
			ID:          toPGUUID(uuid.New()),
			AggregateID: toPGUUID(uuid.New()),
			EventType:   "ticket.unknown",
			AvailableAt: toPGTimestamptz(time.Now()),
		},
	)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ck_outbox_events_type")
}

func Test_IssueRepository_SecondLiveTicketForListing_RejectTicket(t *testing.T) {
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
	_, err := repository.Issue(context.Background(), command)
	require.NoError(t, err)
	command.QueueEntryID = uuid.New()
	command.SKUID = uuid.New()
	command.IdempotencyKey = uuid.New()

	// when
	_, err = repository.Issue(context.Background(), command)

	// then
	require.ErrorIs(t, err, usecase.ErrTicketNotIssuable)
	require.Equal(t, int64(1), integrationTicketCount(t))
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
	err := sqlgen.New(integrationPool).TruncateIntegrationTables(context.Background())
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

func issueListingIntegrationTicket(
	t *testing.T,
	listingID uuid.UUID,
	issuedAt time.Time,
) usecase.IssueTicketResult {
	t.Helper()
	result, err := NewIssueRepository(integrationPool).Issue(context.Background(), usecase.IssueTicketCommand{
		QueueEntryID:       uuid.New(),
		UserID:             uuid.New(),
		ListingID:          listingID,
		SKUID:              uuid.New(),
		IdempotencyKey:     uuid.New(),
		IssuedAt:           issuedAt,
		ActivationDeadline: issuedAt.Add(time.Hour),
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

func integrationTicketCount(t *testing.T) int64 {
	t.Helper()
	count, err := sqlgen.New(integrationPool).CountTickets(context.Background())
	require.NoError(t, err)

	return count
}

func integrationOutboxCount(t *testing.T) int64 {
	t.Helper()
	count, err := sqlgen.New(integrationPool).CountOutboxEvents(context.Background())
	require.NoError(t, err)

	return count
}

func integrationOperationCount(t *testing.T, operation string) int64 {
	t.Helper()
	count, err := sqlgen.New(integrationPool).CountIdempotencyOperationsByOperation(context.Background(), operation)
	require.NoError(t, err)

	return count
}

func integrationOutboxTypeCount(t *testing.T, eventType string) int64 {
	t.Helper()
	count, err := sqlgen.New(integrationPool).CountOutboxEventsByType(context.Background(), eventType)
	require.NoError(t, err)

	return count
}

func integrationOutboxStatusCount(t *testing.T, status string) int64 {
	t.Helper()
	count, err := sqlgen.New(integrationPool).CountOutboxEventsByStatus(context.Background(), status)
	require.NoError(t, err)

	return count
}

func integrationInboxCount(t *testing.T) int64 {
	t.Helper()
	count, err := sqlgen.New(integrationPool).CountInboxEvents(context.Background())
	require.NoError(t, err)

	return count
}
