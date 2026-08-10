package usecase

import (
	"context"

	"github.com/avito-hack/queue/services/queue/internal/domain"
)

func (s *itemQueueService) HandleListingStatusChanged(
	ctx context.Context,
	event ListingStatusChangedEvent,
) error {

	if event.Status != "removed" {
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
			domain.UserLostRights,
		)

		if err != nil {
			return err
		}
	}


	return s.queueRepository.Delete(
		ctx,
		event.ListingID,
	)
}