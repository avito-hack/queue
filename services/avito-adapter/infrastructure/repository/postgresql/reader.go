package postgresql

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/avito-hack/queue/services/avito-adapter/internal/usecase"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ListListings(ctx context.Context, filter usecase.ListingFilter) ([]usecase.Listing, int, error) {
	conditions := make([]string, 0, 2)
	arguments := make([]any, 0, 4)
	if filter.SellerID != nil {
		arguments = append(arguments, *filter.SellerID)
		conditions = append(conditions, fmt.Sprintf("seller_id = $%d", len(arguments)))
	}
	if filter.Status != nil {
		arguments = append(arguments, string(*filter.Status))
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(arguments)))
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}
	var total int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM public.listings"+where, arguments...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count listings: %w", err)
	}
	arguments = append(arguments, filter.Limit, filter.Offset)
	query := "SELECT id, seller_id, title, price, quantity, queue_enabled, status, created_at, updated_at FROM public.listings" + where + fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d OFFSET $%d", len(arguments)-1, len(arguments))
	rows, err := r.pool.Query(ctx, query, arguments...)
	if err != nil {
		return nil, 0, fmt.Errorf("list listings: %w", err)
	}
	defer rows.Close()
	listings := make([]usecase.Listing, 0)
	for rows.Next() {
		listing, err := scanListing(rows)
		if err != nil {
			return nil, 0, err
		}
		listings = append(listings, listing)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate listings: %w", err)
	}
	return listings, total, nil
}

func (r *Repository) GetListing(ctx context.Context, id string) (usecase.Listing, error) {
	listing, err := scanListing(r.pool.QueryRow(ctx, "SELECT id, seller_id, title, price, quantity, queue_enabled, status, created_at, updated_at FROM public.listings WHERE id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return usecase.Listing{}, usecase.ErrNotFound
	}
	return listing, err
}

func (r *Repository) GetUser(ctx context.Context, id string) (usecase.User, error) {
	return r.getUser(ctx, "id", id, usecase.ErrNotFound)
}

func (r *Repository) GetUserByToken(ctx context.Context, token string) (usecase.User, error) {
	return r.getUser(ctx, "token", token, usecase.ErrUnauthorized)
}

func (r *Repository) getUser(ctx context.Context, column, value string, notFound error) (usecase.User, error) {
	var user usecase.User
	err := r.pool.QueryRow(ctx, "SELECT id, name, token, created_at FROM public.users WHERE "+column+" = $1", value).Scan(&user.ID, &user.Name, &user.Token, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return usecase.User{}, notFound
	}
	if err != nil {
		return usecase.User{}, fmt.Errorf("get user by %s: %w", column, err)
	}
	return user, nil
}

func (r *Repository) GetOrder(ctx context.Context, id string) (usecase.Order, error) {
	order, err := scanOrder(r.pool.QueryRow(ctx, "SELECT id, ticket_id, listing_id, sku_id, user_id, idempotency_key, checkout_url, status, created_at FROM public.orders WHERE id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return usecase.Order{}, usecase.ErrNotFound
	}
	if err != nil {
		return usecase.Order{}, fmt.Errorf("get order: %w", err)
	}
	return order, nil
}

func scanOrder(scanner listingScanner) (usecase.Order, error) {
	var order usecase.Order
	if err := scanner.Scan(&order.ID, &order.TicketID, &order.ListingID, &order.SkuID, &order.UserID, &order.IdempotencyKey, &order.CheckoutURL, &order.Status, &order.CreatedAt); err != nil {
		return usecase.Order{}, err
	}
	return order, nil
}

type listingScanner interface {
	Scan(...any) error
}

func scanListing(scanner listingScanner) (usecase.Listing, error) {
	var listing usecase.Listing
	if err := scanner.Scan(&listing.ID, &listing.SellerID, &listing.Title, &listing.Price, &listing.Quantity, &listing.QueueEnabled, &listing.Status, &listing.CreatedAt, &listing.UpdatedAt); err != nil {
		return usecase.Listing{}, err
	}
	return listing, nil
}
