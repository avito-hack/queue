package usecase

import (
	"context"

	"github.com/avito-hack/queue/services/queue/internal/domain"
)

func (s *itemQueueService) HandleTicketClosed(
	ctx context.Context,
	event TicketClosedEvent,
) error {

	member, err := s.memberRepository.GetByTicketID(
		ctx,
		event.TicketID,
	)

	if err != nil {
		return err
	}

	member.Status = domain.UserLostRights

	return s.memberRepository.Update(
		ctx,
		member.ItemID,
		member,
	)
}