package model

import (
	"time"

	"github.com/google/uuid"
)

type Wallet struct {
	UserID    uuid.UUID `json:"user_id"`
	Balance   float64   `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type WalletTransaction struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Amount      float64   `json:"amount"`
	Type        string    `json:"type"`
	ReferenceID string    `json:"reference_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type DepositRequest struct {
	Amount float64 `json:"amount"`
}

type ChargeRequest struct {
	Amount      float64 `json:"amount"`
	ReferenceID string  `json:"reference_id"`
}

type BalanceResponse struct {
	Balance float64 `json:"balance"`
}
