package postgres

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/queue/internal/domain"
	"github.com/avito-hack/queue/services/queue/postgresql/sqlc"
)

type ItemQueueMemberRepository struct {
	queries *sqlc.Queries
	logger  *slog.Logger
}

func NewItemQueueMemberRepository(queries *sqlc.Queries, logger *slog.Logger) *ItemQueueMemberRepository {
	return &ItemQueueMemberRepository{
		queries: queries,
		logger:  logger,
	}
}



func (r *ItemQueueMemberRepository) Create(
	ctx context.Context,
	itemID uuid.UUID,
	member *domain.ItemQueueMember,
) (*domain.ItemQueueMember, error) {

	row, err := r.queries.CreateItemQueueMember(
		ctx,
		sqlc.CreateItemQueueMemberParams{
			ItemID:    uuidToPg(itemID),
			UserID:    uuidToPg(member.UserID),
			TicketID:  uuidToPg(member.TicketID),
			Position:  int32(member.Position),
			Status:    string(member.Status),
			CreatedAt: timeToPg(member.CreatedAt),
		},
	)

	if err != nil {
		return nil, err
	}

	return toDomainMember(row), nil
}

func (r *ItemQueueMemberRepository) GetByUserID(
	ctx context.Context,
	itemID uuid.UUID,
	userID uuid.UUID,
) (*domain.ItemQueueMember, error) {
	row, err := r.queries.GetItemQueueMemberByUserID(
		ctx,
		sqlc.GetItemQueueMemberByUserIDParams{
			ItemID: uuidToPg(itemID),
			UserID: uuidToPg(userID),
		},
	)

	if err != nil {
		r.logger.Error(
			"failed to get queue member",
			"error",
			err,
			"item_id",
			itemID,
			"user_id",
			userID,
		)

		return nil, err
	}

	return toDomainMember(row), nil
}

func (r *ItemQueueMemberRepository) GetAllByItemID(
	ctx context.Context,
	itemID uuid.UUID,
) ([]*domain.ItemQueueMember, error) {
	rows, err := r.queries.GetAllItemQueueMembersByItemID(ctx, uuidToPg(itemID))

	if err != nil {
		r.logger.Error(
			"failed to get queue members by item",
			"error",
			err,
			"item_id",
			itemID,
		)

		return nil, err
	}

	result := make([]*domain.ItemQueueMember, 0, len(rows))

	for _, row := range rows {
		result = append(result, toDomainMember(row))
	}

	return result, nil
}

func (r *ItemQueueMemberRepository) GetAllByUserID(
	ctx context.Context,
	userID uuid.UUID,
) ([]*domain.ItemQueueMember, error) {
	rows, err := r.queries.GetAllItemQueueMembersByUserID(ctx, uuidToPg(userID))

	if err != nil {
		r.logger.Error(
			"failed to get queue members by user",
			"error",
			err,
			"user_id",
			userID,
		)

		return nil, err
	}

	result := make([]*domain.ItemQueueMember, 0, len(rows))

	for _, row := range rows {
		result = append(result, toDomainMember(row))
	}

	return result, nil
}

func (r *ItemQueueMemberRepository) Update(
	ctx context.Context,
	itemID uuid.UUID,
	member *domain.ItemQueueMember,
) error {
	err := r.queries.UpdateItemQueueMember(
		ctx,
		sqlc.UpdateItemQueueMemberParams{
			ItemID:   uuidToPg(itemID),
			UserID:   uuidToPg(member.UserID),
			Position: int32(member.Position),
			Status:   string(member.Status),
		},
	)

	if err != nil {
		r.logger.Error(
			"failed to update queue member",
			"error",
			err,
			"item_id",
			itemID,
			"user_id",
			member.UserID,
		)
	}

	return err
}

func (r *ItemQueueMemberRepository) Delete(
	ctx context.Context,
	itemID uuid.UUID,
	userID uuid.UUID,
) error {
	err := r.queries.DeleteItemQueueMember(
		ctx,
		sqlc.DeleteItemQueueMemberParams{
			ItemID: uuidToPg(itemID),
			UserID: uuidToPg(userID),
		},
	)

	if err != nil {
		r.logger.Error(
			"failed to delete queue member",
			"error",
			err,
			"item_id",
			itemID,
			"user_id",
			userID,
		)
	}

	return err
}

func (r *ItemQueueMemberRepository) DeleteAllByItemID(
	ctx context.Context,
	itemID uuid.UUID,
) error {
	err := r.queries.DeleteAllItemQueueMembersByItemID(ctx, uuidToPg(itemID))

	if err != nil {
		r.logger.Error(
			"failed to delete all queue members",
			"error",
			err,
			"item_id",
			itemID,
		)
	}

	return err
}

func (r *ItemQueueMemberRepository) Exists(
	ctx context.Context,
	itemID uuid.UUID,
	userID uuid.UUID,
) (bool, error) {
	exists, err := r.queries.ExistsItemQueueMember(
		ctx,
		sqlc.ExistsItemQueueMemberParams{
			ItemID: uuidToPg(itemID),
			UserID: uuidToPg(userID),
		},
	)

	if err != nil {
		r.logger.Error(
			"failed to check queue member existence",
			"error",
			err,
			"item_id",
			itemID,
			"user_id",
			userID,
		)
	}

	return exists, err
}

func (r *ItemQueueMemberRepository) Count(
	ctx context.Context,
	itemID uuid.UUID,
) (int, error) {
	count, err := r.queries.CountItemQueueMembers(ctx, uuidToPg(itemID))

	if err != nil {
		r.logger.Error(
			"failed to count queue members",
			"error",
			err,
			"item_id",
			itemID,
		)

		return 0, err
	}

	return int(count), nil
}

func (r *ItemQueueMemberRepository) GetPosition(
	ctx context.Context,
	itemID uuid.UUID,
	userID uuid.UUID,
) (uint, error) {
	pos, err := r.queries.GetItemQueueMemberPosition(
		ctx,
		sqlc.GetItemQueueMemberPositionParams{
			ItemID: uuidToPg(itemID),
			UserID: uuidToPg(userID),
		},
	)

	if err != nil {
		r.logger.Error(
			"failed to get queue member position",
			"error",
			err,
			"item_id",
			itemID,
			"user_id",
			userID,
		)

		return 0, err
	}

	return uint(pos), nil
}

func (r *ItemQueueMemberRepository) ShiftPositionsAfterDelete(
	ctx context.Context,
	itemID uuid.UUID,
	position uint,
) error {
	err := r.queries.ShiftItemQueueMembersPositions(
		ctx,
		sqlc.ShiftItemQueueMembersPositionsParams{
			ItemID:   uuidToPg(itemID),
			Position: int32(position),
		},
	)

	if err != nil {
		r.logger.Error(
			"failed to shift queue member positions",
			"error",
			err,
			"item_id",
			itemID,
			"position",
			position,
		)
	}

	return err
}

func toDomainMember(row sqlc.ItemQueueMember) *domain.ItemQueueMember {
	return &domain.ItemQueueMember{
		ID:        pgToUUID(row.ID),
		ItemID:    pgToUUID(row.ItemID),
		UserID:    pgToUUID(row.UserID),
		TicketID:  pgToUUID(row.TicketID),
		Position:  uint(row.Position),
		Status:    domain.ItemQueueMemberStatus(row.Status),
		CreatedAt: row.CreatedAt.Time,
	}
}