package usecase

import (
	"context"

	"github.com/avito-hack/queue/services/queue/internal/domain"
)

func (s *itemQueueService) HandleTicketRedeemed(
	ctx context.Context,
	event TicketRedeemedEvent,
) error {

	member, err := s.memberRepository.GetByTicketID(
		ctx,
		event.TicketID,
	)

	if err != nil {
		return err
	}

	member.Status = domain.UserPlacedAnOrder

	return s.memberRepository.Update(
		ctx,
		member.ItemID,
		member,
	)
}