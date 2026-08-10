package usecase

import (
	"log/slog"

	"github.com/avito-hack/queue/services/queue/infrastructure/repository/postgres"
	"github.com/avito-hack/queue/services/queue/internal/domain"
)

type itemQueueService struct {
	queueRepository  domain.ItemQueueRepository
	memberRepository domain.ItemQueueMemberRepository
	txManager        postgres.TransactionManager
	avitoClient      AvitoClient
	ticketsClient    TicketsClient
	logger           *slog.Logger
}

func NewItemQueueService(
	queueRepository domain.ItemQueueRepository,
	memberRepository domain.ItemQueueMemberRepository,
	txManager postgres.TransactionManager,
	avitoClient AvitoClient,
	ticketsClient TicketsClient,
	logger *slog.Logger,
) ItemQueueService {
	return &itemQueueService{
		queueRepository:  queueRepository,
		memberRepository: memberRepository,
		txManager:        txManager,
		avitoClient:      avitoClient,
		ticketsClient:    ticketsClient,
		logger:           logger,
	}
}
