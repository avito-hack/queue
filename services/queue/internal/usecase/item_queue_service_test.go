package usecase

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	avitoclient "github.com/avito-hack/queue/services/queue/gen/clients/avito"
	ticketsclient "github.com/avito-hack/queue/services/queue/gen/clients/tickets"
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
	statusCode int
	body       string
	err        error
}

func (s *avitoClientStub) GetListing(context.Context, avitoclient.ListingId, ...avitoclient.RequestEditorFn) (*http.Response, error) {
	if s.err != nil {
		return nil, s.err
	}

	return &http.Response{
		StatusCode: s.statusCode,
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Header:     make(http.Header),
	}, nil
}

type ticketsClientStub struct {
	statusCode int
	body       string
	err        error
}

func (s *ticketsClientStub) ListTickets(_ context.Context, _ *ticketsclient.ListTicketsParams, _ ...ticketsclient.RequestEditorFn) (*http.Response, error) {
	if s.err != nil {
		return nil, s.err
	}

	return &http.Response{
		StatusCode: s.statusCode,
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Header:     make(http.Header),
	}, nil
}

func Test_ItemQueueService_Enqueue_ReturnQueueUnavailableWhenNoAvailableQuantity(t *testing.T) {
	// given
	queueRepository := &queueRepositoryStub{exists: true}
	memberRepository := &memberRepositoryStub{}

	service := NewItemQueueService(
		queueRepository,
		memberRepository,
		&txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository},
		&avitoClientStub{statusCode: http.StatusOK, body: `{"id":"11111111-1111-1111-1111-111111111111","title":"item","price":1000,"quantity":1,"reservedQuantity":1,"availableQuantity":0,"status":"active","queueEnabled":true,"sellerId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","createdAt":"2026-08-07T10:00:00Z","updatedAt":"2026-08-07T10:00:00Z"}`},
		&ticketsClientStub{statusCode: http.StatusOK, body: `{"ticket":[]}`},
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
		&avitoClientStub{statusCode: http.StatusOK, body: `{"id":"11111111-1111-1111-1111-111111111111","title":"item","price":1000,"quantity":10,"reservedQuantity":1,"availableQuantity":9,"status":"active","queueEnabled":true,"sellerId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","createdAt":"2026-08-07T10:00:00Z","updatedAt":"2026-08-07T10:00:00Z"}`},
		&ticketsClientStub{statusCode: http.StatusOK, body: `{"ticket":[{"id":"22222222-2222-2222-2222-222222222222","listing_id":"11111111-1111-1111-1111-111111111111","sku_id":"33333333-3333-3333-3333-333333333333","status":"active","issued_at":"2026-08-07T10:00:00Z","activation_deadline":"2026-08-07T11:00:00Z","available_actions":["activate"],"activated_at":null,"checkout_url":null,"order_id":null,"finished_at":null,"finish_reason":null}]}`},
	)

	ctx := context.WithValue(context.Background(), auth.AuthorizationHeaderKey, "Bearer test-token")

	// when
	err := service.Enqueue(ctx, uuid.New(), uuid.New())

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUserHasActiveTicket)
}

func Test_ItemQueueService_Enqueue_ReturnQueueNotFound(t *testing.T) {
	// given
	queueRepository := &queueRepositoryStub{exists: false}
	memberRepository := &memberRepositoryStub{}
	service := NewItemQueueService(
		queueRepository,
		memberRepository,
		&txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository},
		&avitoClientStub{statusCode: http.StatusOK, body: `{"id":"11111111-1111-1111-1111-111111111111","title":"item","price":1000,"quantity":10,"reservedQuantity":1,"availableQuantity":9,"status":"active","queueEnabled":true,"sellerId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","createdAt":"2026-08-07T10:00:00Z","updatedAt":"2026-08-07T10:00:00Z"}`},
		&ticketsClientStub{statusCode: http.StatusOK, body: `{"ticket":[]}`},
	)
	ctx := context.WithValue(context.Background(), auth.AuthorizationHeaderKey, "Bearer test-token")

	// when
	err := service.Enqueue(ctx, uuid.New(), uuid.New())

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrQueueNotFound)
}

func Test_ItemQueueService_Enqueue_ReturnUserAlreadyInQueue(t *testing.T) {
	// given
	queueRepository := &queueRepositoryStub{exists: true}
	memberRepository := &memberRepositoryStub{exists: true}
	service := NewItemQueueService(
		queueRepository,
		memberRepository,
		&txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository},
		&avitoClientStub{statusCode: http.StatusOK, body: `{"id":"11111111-1111-1111-1111-111111111111","title":"item","price":1000,"quantity":10,"reservedQuantity":1,"availableQuantity":9,"status":"active","queueEnabled":true,"sellerId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","createdAt":"2026-08-07T10:00:00Z","updatedAt":"2026-08-07T10:00:00Z"}`},
		&ticketsClientStub{statusCode: http.StatusOK, body: `{"ticket":[]}`},
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
		&avitoClientStub{statusCode: http.StatusOK, body: `{"id":"11111111-1111-1111-1111-111111111111","title":"item","price":1000,"quantity":10,"reservedQuantity":1,"availableQuantity":9,"status":"active","queueEnabled":true,"sellerId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","createdAt":"2026-08-07T10:00:00Z","updatedAt":"2026-08-07T10:00:00Z"}`},
		&ticketsClientStub{statusCode: http.StatusOK, body: `{"ticket":[]}`},
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
	service := NewItemQueueService(queueRepository, memberRepository, &txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository}, nil, nil)

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
	service := NewItemQueueService(queueRepository, memberRepository, &txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository}, nil, nil)

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
	service := NewItemQueueService(queueRepository, memberRepository, &txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository}, nil, nil)

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
	service := NewItemQueueService(queueRepository, memberRepository, &txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository}, nil, nil)

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
	service := NewItemQueueService(queueRepository, memberRepository, &txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository}, nil, nil)

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
	service := NewItemQueueService(queueRepository, memberRepository, &txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository}, nil, nil)

	// when
	_, err := service.GetItemQueueState(context.Background(), uuid.New())

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrQueueNotFound)
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
	service := NewItemQueueService(queueRepository, memberRepository, &txManagerStub{queueRepository: queueRepository, memberRepository: memberRepository}, nil, nil)

	// when
	result, err := service.GetUserQueues(context.Background(), userID)

	// then
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, itemID, result[0].ItemID)
	assert.Equal(t, 4, result[0].Position)
	assert.Equal(t, domain.UserWaitingInLine, result[0].Status)
}
