package repository

import (
	"context"
	"errors"
	"fmt"

	"minimarket/buySell-app/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrCartItemNotFound = errors.New("cart item not found")

type CartRepository interface {
	GetCart(ctx context.Context, userID uuid.UUID) ([]model.CartItem, error)
	AddToCart(ctx context.Context, userID, itemID uuid.UUID, quantity int) (*model.CartItem, error)
	RemoveFromCart(ctx context.Context, userID, itemID uuid.UUID) error
}

type PostgresCartRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresCartRepository(pool *pgxpool.Pool) *PostgresCartRepository {
	return &PostgresCartRepository{pool: pool}
}

const cartItemColumns = `
	ci.id,
	i.id, i.name, i.description, i.price, i.created_at, i.updated_at,
	ci.quantity, ci.created_at, ci.updated_at`

func (r *PostgresCartRepository) GetCart(ctx context.Context, userID uuid.UUID) ([]model.CartItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+cartItemColumns+`
		FROM cart_items ci
		JOIN items i ON i.id = ci.item_id
		WHERE ci.user_id = $1
		ORDER BY ci.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get cart: %w", err)
	}
	defer rows.Close()

	cartItems := make([]model.CartItem, 0)
	for rows.Next() {
		cartItem, err := scanCartItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan cart item: %w", err)
		}
		cartItems = append(cartItems, *cartItem)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cart items: %w", err)
	}

	return cartItems, nil
}

func (r *PostgresCartRepository) AddToCart(
	ctx context.Context,
	userID, itemID uuid.UUID,
	quantity int,
) (*model.CartItem, error) {
	row := r.pool.QueryRow(ctx, `
		WITH upserted AS (
			INSERT INTO cart_items (user_id, item_id, quantity)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, item_id) DO UPDATE
			SET quantity = cart_items.quantity + EXCLUDED.quantity,
				updated_at = NOW()
			RETURNING id, item_id, quantity, created_at, updated_at
		)
		SELECT
			u.id,
			i.id, i.name, i.description, i.price, i.created_at, i.updated_at,
			u.quantity, u.created_at, u.updated_at
		FROM upserted u
		JOIN items i ON i.id = u.item_id`,
		userID, itemID, quantity,
	)

	cartItem, err := scanCartItem(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("add to cart: %w", err)
	}
	return cartItem, nil
}

func (r *PostgresCartRepository) RemoveFromCart(ctx context.Context, userID, itemID uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `
		DELETE FROM cart_items
		WHERE user_id = $1 AND item_id = $2
	`, userID, itemID)
	if err != nil {
		return fmt.Errorf("remove from cart: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrCartItemNotFound
	}
	return nil
}

type cartItemScanner interface {
	Scan(dest ...any) error
}

func scanCartItem(row cartItemScanner) (*model.CartItem, error) {
	var cartItem model.CartItem
	err := row.Scan(
		&cartItem.ID,
		&cartItem.Item.ID,
		&cartItem.Item.Name,
		&cartItem.Item.Description,
		&cartItem.Item.Price,
		&cartItem.Item.CreatedAt,
		&cartItem.Item.UpdatedAt,
		&cartItem.Quantity,
		&cartItem.CreatedAt,
		&cartItem.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &cartItem, nil
}
