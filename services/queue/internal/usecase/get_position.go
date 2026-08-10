package usecase

import (
	"context"

	"github.com/google/uuid"

)

func (s *itemQueueService) GetUserPosition(
	ctx context.Context,
	itemID uuid.UUID,
	userID uuid.UUID,
) (uint, error) {
	position, err := s.memberRepository.GetRank(
		ctx,
		itemID,
		userID,
	)

	if err != nil {
		s.logger.Warn(
			"failed to get user position",
			"error",
			err,
			"item_id",
			itemID,
			"user_id",
			userID,
		)

		return 0, ErrUserNotInQueue
	}

	return position, nil
}