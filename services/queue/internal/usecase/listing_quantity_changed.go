package usecase

import (
	"context"

	"github.com/avito-hack/queue/services/queue/internal/domain"
)

func (s *itemQueueService) HandleListingQuantityChanged(
	ctx context.Context,
	event ListingQuantityChangedEvent,
) error {

	if event.Quantity > 0 {
		return nil
	}

	members, err := s.memberRepository.GetAllByItemID(
		ctx,
		event.ListingID,
	)

	if err != nil {
		return err
	}


	for _, member := range members {

		if !domain.IsActiveMemberStatus(member.Status) {
			continue
		}

		err = s.memberRepository.Leave(
			ctx,
			event.ListingID,
			member.UserID,
			domain.UserItemOutOfStock,
		)

		if err != nil {
			return err
		}
	}


	return nil
}