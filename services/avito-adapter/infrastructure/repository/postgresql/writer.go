package postgresql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin save listing: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	var previousQuantity int
	var previousStatus usecase.ListingStatus
	var sellerID string
	err = transaction.QueryRow(ctx, "SELECT quantity, status, seller_id FROM public.listings WHERE id = $1 FOR UPDATE", listing.ID).Scan(&previousQuantity, &previousStatus, &sellerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return usecase.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("lock listing: %w", err)
	}
	result, err := transaction.Exec(ctx, "UPDATE public.listings SET title = $2, price = $3, quantity = $4, queue_enabled = $5, status = $6, updated_at = $7 WHERE id = $1", listing.ID, listing.Title, listing.Price, listing.Quantity, listing.QueueEnabled, listing.Status, listing.UpdatedAt)
	if err != nil {
		return fmt.Errorf("save listing: %w", err)
	}
	if result.RowsAffected() == 0 {
		return usecase.ErrNotFound
	}
	if previousQuantity != listing.Quantity {
		if err := insertOutboxEvent(ctx, transaction, "listing.quantity.changed", map[string]any{"listing_id": listing.ID, "seller_id": sellerID, "previous_quantity": previousQuantity, "quantity": listing.Quantity, "changed_at": listing.UpdatedAt}); err != nil {
			return err
		}
	}
	if previousStatus != listing.Status && (listing.Status == usecase.ListingPaused || listing.Status == usecase.ListingRemoved) {
		if err := insertOutboxEvent(ctx, transaction, "listing.status.changed", map[string]any{"listing_id": listing.ID, "seller_id": sellerID, "previous_status": previousStatus, "status": listing.Status, "changed_at": listing.UpdatedAt}); err != nil {
			return err
		}
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit save listing: %w", err)
	}
	return nil
}

func insertOutboxEvent(ctx context.Context, transaction pgx.Tx, eventType string, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode %s: %w", eventType, err)
	}
	now := time.Now().UTC()
	_, err = transaction.Exec(ctx, "INSERT INTO public.outbox_events (id, event_type, payload, created_at, available_at) VALUES ($1, $2, $3, $4, $4)", uuid.NewString(), eventType, encoded, now)
	if err != nil {
		return fmt.Errorf("insert %s: %w", eventType, err)
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
