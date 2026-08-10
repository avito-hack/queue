package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/queue/internal/domain"
)

func (s *itemQueueService) GetUserQueues(
	ctx context.Context,
	userID uuid.UUID,
) ([]*domain.UserQueueInfo, error) {
	members, err := s.memberRepository.GetAllByUserID(
		ctx,
		userID,
	)

	if err != nil {
		s.logger.Error(
			"failed to get user queues",
			"error",
			err,
			"user_id",
			userID,
		)

		return nil, err
	}

	result := make(
		[]*domain.UserQueueInfo,
		0,
		len(members),
	)

	for _, member := range members {
		position := 0

		if member.Position != nil {
			rank, err := s.memberRepository.GetRank(
				ctx,
				member.ItemID,
				member.UserID,
			)

			if err == nil {
				position = int(rank)
			}
		}

		result = append(
			result,
			&domain.UserQueueInfo{
				ItemID:   member.ItemID,
				Position: position,
				Status:   member.Status,
			},
		)
	}

	return result, nil
}