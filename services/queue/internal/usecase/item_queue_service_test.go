package usecase

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/avito-hack/queue/services/queue/infrastructure/auth"
	"github.com/avito-hack/queue/services/queue/internal/domain"
)

type txManagerStub struct {
	queueRepository  domain.ItemQueueRepository
	memberRepository domain.ItemQueueMemberRepository
}

func (m *txManagerStub) WithinTransaction(
	ctx context.Context,
	fn func(
		ctx context.Context,
		queueRepository domain.ItemQueueRepository,
		memberRepository domain.ItemQueueMemberRepository,
	) error,
) error {
	return fn(ctx, m.queueRepository, m.memberRepository)
}

type queueRepositoryStub struct {
	exists    bool
	existsErr error
	queue     *domain.ItemQueue
	getErr    error
}

func (r *queueRepositoryStub) Create(context.Context, *domain.ItemQueue) error { return nil }
func (r *queueRepositoryStub) Update(context.Context, *domain.ItemQueue) error { return nil }
func (r *queueRepositoryStub) Delete(context.Context, uuid.UUID) error         { return nil }

func (r *queueRepositoryStub) Exists(context.Context, uuid.UUID) (bool, error) {
	return r.exists, r.existsErr
}

func (r *queueRepositoryStub) GetByItemID(context.Context, uuid.UUID) (*domain.ItemQueue, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}

	return r.queue, nil
}

type memberRepositoryStub struct {
	exists bool

	existsErr            error
	count                int
	countErr             error
	createErr            error
	getByUser            *domain.ItemQueueMember
	getByUserErr         error
	deleteErr            error
	shiftErr             error
	deletedAllByItemErr  error
	allByUser            []*domain.ItemQueueMember
	allByUserErr         error
	createdItemID        uuid.UUID
	createdMember        *domain.ItemQueueMember
	deletedItemID        uuid.UUID
	deletedUserID        uuid.UUID
	shiftedItemID        uuid.UUID
	shiftedFromPosition  uint
	deleteAllByItemIDArg uuid.UUID
}

func (r *memberRepositoryStub) Create(_ context.Context, itemID uuid.UUID, member *domain.ItemQueueMember) error {
	r.createdItemID = itemID
	r.createdMember = member
	return r.createErr
}

func (r *memberRepositoryStub) GetByUserID(context.Context, uuid.UUID, uuid.UUID) (*domain.ItemQueueMember, error) {
	if r.getByUserErr != nil {
		return nil, r.getByUserErr
	}

	return r.getByUser, nil
}

func (r *memberRepositoryStub) GetAllByItemID(context.Context, uuid.UUID) ([]*domain.ItemQueueMember, error) {
	return nil, nil
}

func (r *memberRepositoryStub) GetAllByUserID(context.Context, uuid.UUID) ([]*domain.ItemQueueMember, error) {
	if r.allByUserErr != nil {
		return nil, r.allByUserErr
	}

	return r.allByUser, nil
}

func (r *memberRepositoryStub) Update(context.Context, uuid.UUID, *domain.ItemQueueMember) error {
	return nil
}

func (r *memberRepositoryStub) Delete(_ context.Context, itemID, userID uuid.UUID) error {
	r.deletedItemID = itemID
	r.deletedUserID = userID
	return r.deleteErr
}

func (r *memberRepositoryStub) DeleteAllByItemID(_ context.Context, itemID uuid.UUID) error {
	r.deleteAllByItemIDArg = itemID
	return r.deletedAllByItemErr
}

func (r *memberRepositoryStub) Exists(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return r.exists, r.existsErr
}

func (r *memberRepositoryStub) Count(context.Context, uuid.UUID) (int, error) {
	if r.countErr != nil {
		return 0, r.countErr
	}

	return r.count, nil
}

func (r *memberRepositoryStub) GetPosition(context.Context, uuid.UUID, uuid.UUID) (uint, error) {
	return 0, nil
}

func (r *memberRepositoryStub) ShiftPositionsAfterDelete(_ context.Context, itemID uuid.UUID, position uint) error {
	r.shiftedItemID = itemID
	r.shiftedFromPosition = position
	return r.shiftErr
}

type avitoClientStub struct {
	listing *domain.Listing
	err     error
}

func (s *avitoClientStub) GetListing(context.Context, uuid.UUID) (*domain.Listing, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.listing, nil
}

type ticketsClientStub struct {
	tickets []*domain.Ticket
	err     error
}

func (s *ticketsClientStub) GetTickets(context.Context, uuid.UUID) ([]*domain.Ticket, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.tickets, nil
}

func Test_ItemQueueService_Enqueue_ReturnQueueUnavailableWhenNoAvailableQuantity(t *testing.T) {
	// given
	queueRepository := &queueRepositoryStub{exists: true}
	memberRepository := &memberRepositoryStub{}

	service := NewItemQueueService(
		queueRepository,
		memberRepository,
		&txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository},
		&avitoClientStub{listing: &domain.Listing{QueueEnabled: true, Status: "active", Quantity: 0}},
		&ticketsClientStub{},
		slog.Default(),
	)

	ctx := context.WithValue(context.Background(), auth.AuthorizationHeaderKey, "Bearer test-token")

	// when
	err := service.Enqueue(ctx, uuid.New(), uuid.New())

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrQueueUnavailable)
}

func Test_ItemQueueService_Enqueue_ReturnConflictWhenUserHasActiveTicket(t *testing.T) {
	// given
	queueRepository := &queueRepositoryStub{exists: true}
	memberRepository := &memberRepositoryStub{}

	service := NewItemQueueService(
		queueRepository,
		memberRepository,
		&txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository},
		&avitoClientStub{listing: &domain.Listing{QueueEnabled: true, Status: "active", Quantity: 10}},
		&ticketsClientStub{tickets: []*domain.Ticket{{ID: "ticket-id", Status: domain.TicketIssued}}},
		slog.Default(),
	)

	ctx := context.WithValue(context.Background(), auth.AuthorizationHeaderKey, "Bearer test-token")

	// when
	err := service.Enqueue(ctx, uuid.New(), uuid.New())

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUserHasActiveTicket)
}

func Test_ItemQueueService_Enqueue_CreateQueueWhenNotFound(t *testing.T) {
	// given
	queueRepository := &queueRepositoryStub{exists: false}
	memberRepository := &memberRepositoryStub{}
	service := NewItemQueueService(
		queueRepository,
		memberRepository,
		&txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository},
		&avitoClientStub{listing: &domain.Listing{QueueEnabled: true, Status: "active", Quantity: 10}},
		&ticketsClientStub{},
		slog.Default(),
	)
	ctx := context.WithValue(context.Background(), auth.AuthorizationHeaderKey, "Bearer test-token")

	// when
	err := service.Enqueue(ctx, uuid.New(), uuid.New())

	// then
	require.NoError(t, err)
}

func Test_ItemQueueService_Enqueue_ReturnUserAlreadyInQueue(t *testing.T) {
	// given
	queueRepository := &queueRepositoryStub{exists: true}
	memberRepository := &memberRepositoryStub{exists: true}
	service := NewItemQueueService(
		queueRepository,
		memberRepository,
		&txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository},
		&avitoClientStub{listing: &domain.Listing{QueueEnabled: true, Status: "active", Quantity: 10}},
		&ticketsClientStub{},
		slog.Default(),
	)
	ctx := context.WithValue(context.Background(), auth.AuthorizationHeaderKey, "Bearer test-token")

	// when
	err := service.Enqueue(ctx, uuid.New(), uuid.New())

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUserAlreadyInQueue)
}

func Test_ItemQueueService_Enqueue_CreateMemberWithNextPosition(t *testing.T) {
	// given
	itemID := uuid.New()
	userID := uuid.New()

	queueRepository := &queueRepositoryStub{exists: true}
	memberRepository := &memberRepositoryStub{exists: false, count: 2}
	service := NewItemQueueService(
		queueRepository,
		memberRepository,
		&txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository},
		&avitoClientStub{listing: &domain.Listing{QueueEnabled: true, Status: "active", Quantity: 10}},
		&ticketsClientStub{},
		slog.Default(),
	)
	ctx := context.WithValue(context.Background(), auth.AuthorizationHeaderKey, "Bearer test-token")

	// when
	err := service.Enqueue(ctx, itemID, userID)

	// then
	require.NoError(t, err)
	require.NotNil(t, memberRepository.createdMember)
	assert.Equal(t, itemID, memberRepository.createdItemID)
	assert.Equal(t, userID, memberRepository.createdMember.UserID)
	assert.Equal(t, uint(3), memberRepository.createdMember.Position)
	assert.Equal(t, domain.UserWaitingInLine, memberRepository.createdMember.Status)
	assert.WithinDuration(t, time.Now(), memberRepository.createdMember.CreatedAt, 2*time.Second)
}

func Test_ItemQueueService_Dequeue_ReturnUserNotInQueue(t *testing.T) {
	// given
	queueRepository := &queueRepositoryStub{}
	memberRepository := &memberRepositoryStub{getByUserErr: errors.New("not found")}
	service := NewItemQueueService(queueRepository, memberRepository, &txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository}, nil, nil, slog.Default())

	// when
	err := service.Dequeue(context.Background(), uuid.New(), uuid.New())

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUserNotInQueue)
}

func Test_ItemQueueService_Dequeue_DeleteMemberAndShiftPositions(t *testing.T) {
	// given
	itemID := uuid.New()
	userID := uuid.New()

	queueRepository := &queueRepositoryStub{}
	memberRepository := &memberRepositoryStub{getByUser: &domain.ItemQueueMember{Position: 2}}
	service := NewItemQueueService(queueRepository, memberRepository, &txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository}, nil, nil, slog.Default())

	// when
	err := service.Dequeue(context.Background(), itemID, userID)

	// then
	require.NoError(t, err)
	assert.Equal(t, itemID, memberRepository.deletedItemID)
	assert.Equal(t, userID, memberRepository.deletedUserID)
	assert.Equal(t, itemID, memberRepository.shiftedItemID)
	assert.Equal(t, uint(2), memberRepository.shiftedFromPosition)
}

func Test_ItemQueueService_ClearItemQueue_ReturnQueueNotFound(t *testing.T) {
	// given
	queueRepository := &queueRepositoryStub{exists: false}
	memberRepository := &memberRepositoryStub{}
	service := NewItemQueueService(queueRepository, memberRepository, &txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository}, nil, nil, slog.Default())

	// when
	err := service.ClearItemQueue(context.Background(), uuid.New())

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrQueueNotFound)
}

func Test_ItemQueueService_ClearItemQueue_DeleteAllMembersByItemID(t *testing.T) {
	// given
	itemID := uuid.New()
	queueRepository := &queueRepositoryStub{exists: true}
	memberRepository := &memberRepositoryStub{}
	service := NewItemQueueService(queueRepository, memberRepository, &txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository}, nil, nil, slog.Default())

	// when
	err := service.ClearItemQueue(context.Background(), itemID)

	// then
	require.NoError(t, err)
	assert.Equal(t, itemID, memberRepository.deleteAllByItemIDArg)
}

func Test_ItemQueueService_GetUserPosition_ReturnUserNotInQueue(t *testing.T) {
	// given
	queueRepository := &queueRepositoryStub{}
	memberRepository := &memberRepositoryStub{getByUserErr: errors.New("not found")}
	service := NewItemQueueService(queueRepository, memberRepository, &txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository}, nil, nil, slog.Default())

	// when
	_, err := service.GetUserPosition(context.Background(), uuid.New(), uuid.New())

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUserNotInQueue)
}

func Test_ItemQueueService_GetItemQueueState_ReturnQueueNotFound(t *testing.T) {
	// given
	queueRepository := &queueRepositoryStub{getErr: errors.New("not found")}
	memberRepository := &memberRepositoryStub{}
	service := NewItemQueueService(
		queueRepository,
		memberRepository,
		&txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository},
		nil,
		nil,
		slog.Default(),
	)

	// when
	_, err := service.GetItemQueueState(context.Background(), uuid.New())

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrQueueNotFound)
}

func Test_ItemQueueService_GetItemQueueState_ReturnStateAndWaitingCount(t *testing.T) {
	// given
	itemID := uuid.New()
	queueRepository := &queueRepositoryStub{
		queue: &domain.ItemQueue{ItemID: itemID, State: domain.QueueTicketsAvailable},
	}
	memberRepository := &memberRepositoryStub{count: 2}
	service := NewItemQueueService(
		queueRepository,
		memberRepository,
		&txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository},
		nil,
		nil,
		slog.Default(),
	)

	// when
	info, err := service.GetItemQueueState(context.Background(), itemID)

	// then
	require.NoError(t, err)
	assert.Equal(t, domain.QueueTicketsAvailable, info.State)
	assert.Equal(t, 2, info.WaitingCount)
}

func Test_ItemQueueService_GetUserQueues_ReturnMappedQueues(t *testing.T) {
	// given
	userID := uuid.New()
	itemID := uuid.New()

	queueRepository := &queueRepositoryStub{}
	memberRepository := &memberRepositoryStub{
		allByUser: []*domain.ItemQueueMember{{
			ItemID:   itemID,
			Position: 4,
			Status:   domain.UserWaitingInLine,
		}},
	}
	service := NewItemQueueService(queueRepository, memberRepository, &txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository}, nil, nil, slog.Default())

	// when
	result, err := service.GetUserQueues(context.Background(), userID)

	// then
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, itemID, result[0].ItemID)
	assert.Equal(t, 4, result[0].Position)
	assert.Equal(t, domain.UserWaitingInLine, result[0].Status)
}
