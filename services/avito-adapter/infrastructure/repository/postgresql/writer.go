package postgresql

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/avito-hack/queue/services/avito-adapter/internal/usecase"
)

func (r *Repository) CreateUser(ctx context.Context, user usecase.User) error {
	_, err := r.pool.Exec(ctx, "INSERT INTO public.users (id, name, token, created_at) VALUES ($1, $2, $3, $4)", user.ID, user.Name, user.Token, user.CreatedAt)
	if uniqueViolation(err) {
		return usecase.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *Repository) CreateListing(ctx context.Context, listing usecase.Listing) error {
	_, err := r.pool.Exec(ctx, "INSERT INTO public.listings (id, seller_id, title, price, quantity, queue_enabled, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)", listing.ID, listing.SellerID, listing.Title, listing.Price, listing.Quantity, listing.QueueEnabled, listing.Status, listing.CreatedAt, listing.UpdatedAt)
	if foreignKeyViolation(err) {
		return usecase.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("create listing: %w", err)
	}
	return nil
}

func (r *Repository) SaveListing(ctx context.Context, listing usecase.Listing) error {
	result, err := r.pool.Exec(ctx, "UPDATE public.listings SET title = $2, price = $3, quantity = $4, queue_enabled = $5, status = $6, updated_at = $7 WHERE id = $1", listing.ID, listing.Title, listing.Price, listing.Quantity, listing.QueueEnabled, listing.Status, listing.UpdatedAt)
	if err != nil {
		return fmt.Errorf("save listing: %w", err)
	}
	if result.RowsAffected() == 0 {
		return usecase.ErrNotFound
	}
	return nil
}

func (r *Repository) CreateOrder(ctx context.Context, requested usecase.Order) (usecase.Order, error) {
	transaction, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return usecase.Order{}, fmt.Errorf("begin create order: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	if _, err := transaction.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", requested.IdempotencyKey); err != nil {
		return usecase.Order{}, fmt.Errorf("lock idempotency key: %w", err)
	}

	existing, err := getOrderByIdempotencyKey(ctx, transaction, requested.IdempotencyKey)
	if err == nil {
		if existing.TicketID != requested.TicketID || existing.ListingID != requested.ListingID || existing.SkuID != requested.SkuID || existing.UserID != requested.UserID {
			return usecase.Order{}, usecase.ErrConflict
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return usecase.Order{}, fmt.Errorf("get idempotent order: %w", err)
	}

	var ticketExists bool
	if err := transaction.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM public.orders WHERE ticket_id = $1)", requested.TicketID).Scan(&ticketExists); err != nil {
		return usecase.Order{}, fmt.Errorf("check order ticket: %w", err)
	}
	if ticketExists {
		return usecase.Order{}, usecase.ErrConflict
	}

	var userExists bool
	if err := transaction.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM public.users WHERE id = $1)", requested.UserID).Scan(&userExists); err != nil {
		return usecase.Order{}, fmt.Errorf("check order user: %w", err)
	}
	if !userExists {
		return usecase.Order{}, usecase.ErrNotFound
	}

	var quantity int
	var status usecase.ListingStatus
	err = transaction.QueryRow(ctx, "SELECT quantity, status FROM public.listings WHERE id = $1 FOR UPDATE", requested.ListingID).Scan(&quantity, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return usecase.Order{}, usecase.ErrNotFound
	}
	if err != nil {
		return usecase.Order{}, fmt.Errorf("lock order listing: %w", err)
	}
	if status != usecase.ListingActive {
		return usecase.Order{}, usecase.ErrUnavailable
	}

	var reservedQuantity int
	if err := transaction.QueryRow(ctx, "SELECT count(*) FROM public.orders WHERE listing_id = $1", requested.ListingID).Scan(&reservedQuantity); err != nil {
		return usecase.Order{}, fmt.Errorf("count listing reservations: %w", err)
	}
	if reservedQuantity >= quantity {
		return usecase.Order{}, usecase.ErrConflict
	}

	_, err = transaction.Exec(ctx, "INSERT INTO public.orders (id, ticket_id, listing_id, sku_id, user_id, idempotency_key, checkout_url, status, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)", requested.ID, requested.TicketID, requested.ListingID, requested.SkuID, requested.UserID, requested.IdempotencyKey, requested.CheckoutURL, requested.Status, requested.CreatedAt)
	if uniqueViolation(err) {
		return usecase.Order{}, usecase.ErrConflict
	}
	if foreignKeyViolation(err) {
		return usecase.Order{}, usecase.ErrNotFound
	}
	if err != nil {
		return usecase.Order{}, fmt.Errorf("insert order: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return usecase.Order{}, fmt.Errorf("commit create order: %w", err)
	}
	return requested, nil
}

func getOrderByIdempotencyKey(ctx context.Context, transaction pgx.Tx, key string) (usecase.Order, error) {
	return scanOrder(transaction.QueryRow(ctx, "SELECT id, ticket_id, listing_id, sku_id, user_id, idempotency_key, checkout_url, status, created_at FROM public.orders WHERE idempotency_key = $1", key))
}

func uniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23505"
}

func foreignKeyViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23503"
}
