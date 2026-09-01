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

var ErrInsufficientFunds = errors.New("insufficient funds")

type WalletRepository interface {
	GetBalance(ctx context.Context, userID uuid.UUID) (float64, error)
	TransactionsList(ctx context.Context, userID uuid.UUID) ([]model.WalletTransaction, error)
	ApplyTransaction(ctx context.Context, transaction model.WalletTransaction) (float64, error)
}

type PostgresWalletRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresWalletRepository(pool *pgxpool.Pool) *PostgresWalletRepository {
	return &PostgresWalletRepository{pool: pool}
}

func (r *PostgresWalletRepository) initWallet(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO wallets (user_id, balance)
		VALUES ($1, 0)
		ON CONFLICT (user_id) DO NOTHING`, userID)
	if err != nil {
		return fmt.Errorf("initialize wallet: %w", err)
	}
	return nil
}

func (r *PostgresWalletRepository) GetBalance(ctx context.Context, userID uuid.UUID) (float64, error) {
	if err := r.initWallet(ctx, userID); err != nil {
		return 0, err
	}

	var balance float64
	if err := r.pool.QueryRow(
		ctx,
		`SELECT balance FROM wallets WHERE user_id = $1`,
		userID,
	).Scan(&balance); err != nil {
		return 0, fmt.Errorf("get wallet balance: %w", err)
	}
	return balance, nil
}

func (r *PostgresWalletRepository) TransactionsList(ctx context.Context, userID uuid.UUID) ([]model.WalletTransaction, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, amount, type, reference_id, created_at, updated_at
		FROM transactions
		WHERE user_id = $1
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list wallet transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]model.WalletTransaction, 0)
	for rows.Next() {
		var transaction model.WalletTransaction
		if err := rows.Scan(&transaction.ID, &transaction.UserID, &transaction.Amount, &transaction.Type, &transaction.ReferenceID, &transaction.CreatedAt, &transaction.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan wallet transaction: %w", err)
		}
		transactions = append(transactions, transaction)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate wallet transactions: %w", err)
	}
	return transactions, nil
}

func (r *PostgresWalletRepository) ApplyTransaction(ctx context.Context, transaction model.WalletTransaction) (float64, error) {
	dbTx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin wallet transaction: %w", err)
	}
	defer func() { _ = dbTx.Rollback(ctx) }()

	if _, err := dbTx.Exec(ctx, `
		INSERT INTO wallets (user_id, balance)
		VALUES ($1, 0)
		ON CONFLICT (user_id) DO NOTHING`, transaction.UserID); err != nil {
		return 0, fmt.Errorf("initialize wallet in transaction: %w", err)
	}

	var balance float64
	err = dbTx.QueryRow(ctx, `
		UPDATE wallets
		SET balance = balance + $1, updated_at = NOW()
		WHERE user_id = $2 AND balance + $1 >= 0
		RETURNING balance`,
		transaction.Amount, transaction.UserID,
	).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrInsufficientFunds
	}
	if err != nil {
		return 0, fmt.Errorf("update wallet balance: %w", err)
	}

	_, err = dbTx.Exec(ctx, `
		INSERT INTO transactions (id, user_id, amount, type, reference_id)
		VALUES ($1, $2, $3, $4, $5)`,
		uuid.New(), transaction.UserID, transaction.Amount, transaction.Type, transaction.ReferenceID,
	)
	if err != nil {
		return 0, fmt.Errorf("record wallet transaction: %w", err)
	}

	if err := dbTx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit wallet transaction: %w", err)
	}

	return balance, nil
}
