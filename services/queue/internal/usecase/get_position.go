package usecase

import (
	"context"

	"github.com/google/uuid"
)

func (s *itemQueueService) GetUserPosition(ctx context.Context, itemID, userID uuid.UUID) (uint, error) {
	member, err := s.memberRepository.GetByUserID(ctx, itemID, userID)
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

	return member.Position, nil
}
