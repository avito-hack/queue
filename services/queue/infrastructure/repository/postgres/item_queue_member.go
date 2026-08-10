package postgres

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/avito-hack/queue/services/queue/internal/domain"
	"github.com/avito-hack/queue/services/queue/postgresql/sqlc"
)

type ItemQueueMemberRepository struct {
	queries *sqlc.Queries
	logger  *slog.Logger
}

func NewItemQueueMemberRepository(
	queries *sqlc.Queries,
	logger *slog.Logger,
) *ItemQueueMemberRepository {
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
			TicketID:  uuidToPgPtr(member.TicketID),
			Position:  uintPtrToPgInt4(member.Position),
			Status:    string(member.Status),
			CreatedAt: timeToPg(member.CreatedAt),
		},
	)

	if err != nil {
		r.logger.Error(
			"failed to create queue member",
			"error",
			err,
			"item_id",
			itemID,
			"user_id",
			member.UserID,
			"ticket_id",
			member.TicketID,
		)

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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMemberNotFound
		}

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
	rows, err := r.queries.GetAllItemQueueMembersByItemID(
		ctx,
		uuidToPg(itemID),
	)

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
		result = append(
			result,
			toDomainMember(row),
		)
	}

	return result, nil
}

func (r *ItemQueueMemberRepository) GetAllByUserID(
	ctx context.Context,
	userID uuid.UUID,
) ([]*domain.ItemQueueMember, error) {
	rows, err := r.queries.GetAllItemQueueMembersByUserID(
		ctx,
		uuidToPg(userID),
	)

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
		result = append(
			result,
			toDomainMember(row),
		)
	}

	return result, nil
}

func (r *ItemQueueMemberRepository) Update(
	ctx context.Context,
	itemID uuid.UUID,
	member *domain.ItemQueueMember,
) error {

	ticketID := uuidToPgPtr(member.TicketID)

	r.logger.Info(
		"repository update member",
		"item_id",
		itemID,
		"user_id",
		member.UserID,
		"ticket_id",
		member.TicketID,
		"ticket_pg_valid",
		ticketID.Valid,
		"status",
		member.Status,
	)

	err := r.queries.UpdateItemQueueMember(
		ctx,
		sqlc.UpdateItemQueueMemberParams{
			ItemID:   uuidToPg(itemID),
			UserID:   uuidToPg(member.UserID),
			TicketID: ticketID,
			Position: uintPtrToPgInt4(member.Position),
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
			"ticket_id",
			member.TicketID,
		)
	}

	return err
}

func (r *ItemQueueMemberRepository) Leave(
	ctx context.Context,
	itemID uuid.UUID,
	userID uuid.UUID,
	reason domain.ItemQueueMemberStatus,
) error {
	err := r.queries.LeaveItemQueueMember(
		ctx,
		sqlc.LeaveItemQueueMemberParams{
			ItemID: uuidToPg(itemID),
			UserID: uuidToPg(userID),
			Status: string(reason),
		},
	)

	if err != nil {
		r.logger.Error(
			"failed to leave queue member",
			"error",
			err,
			"item_id",
			itemID,
			"user_id",
			userID,
			"reason",
			reason,
		)
	}

	return err
}

func (r *ItemQueueMemberRepository) Reactivate(
	ctx context.Context,
	itemID uuid.UUID,
	userID uuid.UUID,
	position uint,
) error {
	err := r.queries.ReactivateItemQueueMember(
		ctx,
		sqlc.ReactivateItemQueueMemberParams{
			ItemID:   uuidToPg(itemID),
			UserID:   uuidToPg(userID),
			Position: pgtype.Int4{Int32: int32(position), Valid: true},
		},
	)

	if err != nil {
		r.logger.Error(
			"failed to reactivate queue member",
			"error",
			err,
			"item_id",
			itemID,
			"user_id",
			userID,
			"position",
			position,
		)
	}

	return err
}

func (r *ItemQueueMemberRepository) DeleteAllByItemID(
	ctx context.Context,
	itemID uuid.UUID,
) error {
	err := r.queries.DeleteAllItemQueueMembersByItemID(
		ctx,
		uuidToPg(itemID),
	)

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

func (r *ItemQueueMemberRepository) GetPosition(
	ctx context.Context,
	itemID uuid.UUID,
	userID uuid.UUID,
) (uint, error) {
	position, err := r.queries.GetItemQueueMemberPosition(
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

	if !position.Valid {
		return 0, domain.ErrMemberNotFound
	}

	return uint(position.Int32), nil
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
			Position: pgtype.Int4{Int32: int32(position), Valid: true},
		},
	)

	if err != nil {
		r.logger.Error(
			"failed to shift queue positions",
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

func (r *ItemQueueMemberRepository) GetRank(
	ctx context.Context,
	itemID uuid.UUID,
	userID uuid.UUID,
) (uint, error) {
	rank, err := r.queries.GetUserQueueRank(
		ctx,
		sqlc.GetUserQueueRankParams{
			ItemID: uuidToPg(itemID),
			UserID: uuidToPg(userID),
		},
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, domain.ErrMemberNotFound
		}

		r.logger.Error(
			"failed to get queue member rank",
			"error",
			err,
			"item_id",
			itemID,
			"user_id",
			userID,
		)

		return 0, err
	}

	return uint(rank), nil
}
func toDomainMember(row sqlc.ItemQueueMember) *domain.ItemQueueMember {
	return &domain.ItemQueueMember{
		ID:        pgToUUID(row.ID),
		ItemID:    pgToUUID(row.ItemID),
		UserID:    pgToUUID(row.UserID),
		TicketID:  pgToUUIDPtr(row.TicketID),
		Position:  pgInt4ToUintPtr(row.Position),
		Status:    domain.ItemQueueMemberStatus(row.Status),
		CreatedAt: row.CreatedAt.Time,
	}
}

func uintPtrToPgInt4(p *uint) pgtype.Int4 {
	if p == nil {
		return pgtype.Int4{
			Valid: false,
		}
	}

	return pgtype.Int4{
		Int32: int32(*p),
		Valid: true,
	}
}

func pgInt4ToUintPtr(v pgtype.Int4) *uint {
	if !v.Valid {
		return nil
	}

	u := uint(v.Int32)

	return &u
}

func pgToUUIDPtr(id pgtype.UUID) *uuid.UUID {
	if !id.Valid {
		return nil
	}

	value := uuid.UUID(id.Bytes)

	return &value
}

func uuidToPgPtr(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{
			Valid: false,
		}
	}

	var bytes [16]byte

	copy(
		bytes[:],
		id[:],
	)

	return pgtype.UUID{
		Bytes: bytes,
		Valid: true,
	}
}