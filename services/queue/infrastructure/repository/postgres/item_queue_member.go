package postgres

import (
    "context"

    "github.com/google/uuid"

    "github.com/avito-hack/queue/services/queue/internal/domain"
    "github.com/avito-hack/queue/services/queue/postgresql/sqlc"
)

type ItemQueueMemberRepository struct {
    queries *sqlc.Queries
}

func NewItemQueueMemberRepository(queries *sqlc.Queries) *ItemQueueMemberRepository {
    return &ItemQueueMemberRepository{
        queries: queries,
    }
}

func (r *ItemQueueMemberRepository) Create(
    ctx context.Context,
    itemID uuid.UUID,
    member *domain.ItemQueueMember,
) error {
    return r.queries.CreateItemQueueMember(
        ctx,
        sqlc.CreateItemQueueMemberParams{
            ItemID:    uuidToPg(itemID),
            UserID:    uuidToPg(member.UserID),
            Position:  int32(member.Position),
            Status:    string(member.Status),
            CreatedAt: timeToPg(member.CreatedAt),
        },
    )
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
    return r.queries.UpdateItemQueueMember(
        ctx,
        sqlc.UpdateItemQueueMemberParams{
            ItemID:   uuidToPg(itemID),
            UserID:   uuidToPg(member.UserID),
            Position: int32(member.Position),
            Status:   string(member.Status),
        },
    )
}

func (r *ItemQueueMemberRepository) Delete(
    ctx context.Context,
    itemID uuid.UUID,
    userID uuid.UUID,
) error {
    return r.queries.DeleteItemQueueMember(
        ctx,
        sqlc.DeleteItemQueueMemberParams{
            ItemID: uuidToPg(itemID),
            UserID: uuidToPg(userID),
        },
    )
}

func (r *ItemQueueMemberRepository) DeleteAllByItemID(
    ctx context.Context,
    itemID uuid.UUID,
) error {
    return r.queries.DeleteAllItemQueueMembersByItemID(ctx, uuidToPg(itemID))
}

func (r *ItemQueueMemberRepository) Exists(
    ctx context.Context,
    itemID uuid.UUID,
    userID uuid.UUID,
) (bool, error) {
    return r.queries.ExistsItemQueueMember(
        ctx,
        sqlc.ExistsItemQueueMemberParams{
            ItemID: uuidToPg(itemID),
            UserID: uuidToPg(userID),
        },
    )
}

func (r *ItemQueueMemberRepository) Count(
    ctx context.Context,
    itemID uuid.UUID,
) (int, error) {
    count, err := r.queries.CountItemQueueMembers(ctx, uuidToPg(itemID))
    if err != nil {
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
        return 0, err
    }

    return uint(pos), nil
}

func (r *ItemQueueMemberRepository) ShiftPositionsAfterDelete(
    ctx context.Context,
    itemID uuid.UUID,
    position uint,
) error {
    return r.queries.ShiftItemQueueMembersPositions(
        ctx,
        sqlc.ShiftItemQueueMembersPositionsParams{
            ItemID:   uuidToPg(itemID),
            Position: int32(position),
        },
    )
}

func toDomainMember(row sqlc.ItemQueueMember) *domain.ItemQueueMember {
    return &domain.ItemQueueMember{
        ItemID:    pgToUUID(row.ItemID),
        UserID:    pgToUUID(row.UserID),
        Position:  uint(row.Position),
        Status:    domain.ItemQueueMemberStatus(row.Status),
        CreatedAt: row.CreatedAt.Time,
    }
}
