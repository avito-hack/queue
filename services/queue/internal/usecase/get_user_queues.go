package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/queue/internal/domain"
)

func (s *itemQueueService) GetUserQueues(ctx context.Context, userID uuid.UUID) ([]*domain.UserQueueInfo, error) {
	members, err := s.memberRepository.GetAllByUserID(ctx, userID)
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

	result := make([]*domain.UserQueueInfo, 0, len(members))

	for _, member := range members {
		result = append(result, &domain.UserQueueInfo{
			ItemID:   member.ItemID,
			Position: int(member.Position),
			Status:   member.Status,
		})
	}

	return result, nil
}