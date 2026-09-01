package repository

import (
	"context"
	"errors"
	"fmt"

	"minimarket/buySell-app/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("item not found")
)

type BuySellRepository interface {
	GetItem(context.Context, uuid.UUID) (*model.Item, error)
	CreateItem(context.Context, model.CreateItemRequest) (*model.Item, error)
	GetItemsList(context.Context) ([]*model.Item, error)
}

type PostgresBuySellRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresBuySellRepository(pool *pgxpool.Pool) *PostgresBuySellRepository {
	return &PostgresBuySellRepository{pool: pool}
}

const itemColumns = `id, name, description, price, created_at, updated_at`

func (r *PostgresBuySellRepository) GetItem(ctx context.Context, id uuid.UUID) (*model.Item, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+itemColumns+" FROM items WHERE id = $1", id)
	var item model.Item
	err := row.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get item: %w", err)
	}
	return &item, nil
}

func (r *PostgresBuySellRepository) GetItemsList(ctx context.Context) ([]*model.Item, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+itemColumns+" FROM items")
	if err != nil {
		return nil, fmt.Errorf("get items list: %w", err)
	}
	defer rows.Close()

	items := make([]*model.Item, 0)
	for rows.Next() {
		var item model.Item
		err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("get items list: %w", err)
		}
		items = append(items, &item)
	}
	return items, nil
}

func (r *PostgresBuySellRepository) CreateItem(ctx context.Context, item model.CreateItemRequest) (*model.Item, error) {
	query := `INSERT INTO items (name, description, price) VALUES ($1, $2, $3) RETURNING ` + itemColumns
	row := r.pool.QueryRow(ctx, query, item.Name, item.Description, item.Price)
	var createdItem model.Item
	err := row.Scan(&createdItem.ID, &createdItem.Name, &createdItem.Description, &createdItem.Price, &createdItem.CreatedAt, &createdItem.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create item: %w", err)
	}
	return &createdItem, nil
}
