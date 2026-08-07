package usecase

import (
	"context"
	"net/http"

	"github.com/avito-hack/queue/services/queue/gen/clients/avito"
	"github.com/avito-hack/queue/services/queue/gen/clients/tickets"
	"github.com/avito-hack/queue/services/queue/infrastructure/repository/postgres"
	"github.com/avito-hack/queue/services/queue/internal/domain"
)

type AvitoClient interface {
	GetListing(ctx context.Context, listingId avito.ListingId, reqEditors ...avito.RequestEditorFn) (*http.Response, error)
}

type TicketsClient interface {
	ListTickets(ctx context.Context, params *tickets.ListTicketsParams, reqEditors ...tickets.RequestEditorFn) (*http.Response, error)
}

type itemQueueService struct {
	queueRepository  domain.ItemQueueRepository
	memberRepository domain.ItemQueueMemberRepository
	txManager        postgres.TransactionManager
	avitoClient      AvitoClient
	ticketsClient    TicketsClient
}

func NewItemQueueService(
	queueRepository domain.ItemQueueRepository,
	memberRepository domain.ItemQueueMemberRepository,
	txManager postgres.TransactionManager,
	avitoClient AvitoClient,
	ticketsClient TicketsClient,
) ItemQueueService {
	return &itemQueueService{
		queueRepository:  queueRepository,
		memberRepository: memberRepository,
		txManager:        txManager,
		avitoClient:      avitoClient,
		ticketsClient:    ticketsClient,
	}
}
