package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"minimarket/buySell-app/internal/model"
	"minimarket/buySell-app/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrInvalidAmount     = errors.New("amount must be positive")
	ErrReferenceRequired = errors.New("reference_id is required")
)

type WalletService struct {
	walletRepository repository.WalletRepository
}

func NewWalletService(walletRepository repository.WalletRepository) *WalletService {
	return &WalletService{walletRepository: walletRepository}
}

func (s *WalletService) GetBalance(ctx context.Context, userID uuid.UUID) (float64, error) {
	return s.walletRepository.GetBalance(ctx, userID)
}

func (s *WalletService) TransactionsList(ctx context.Context, userID uuid.UUID) ([]model.WalletTransaction, error) {
	return s.walletRepository.TransactionsList(ctx, userID)
}

func (s *WalletService) Deposit(ctx context.Context, userID uuid.UUID, request model.DepositRequest) (float64, error) {
	if request.Amount <= 0 {
		return 0, ErrInvalidAmount
	}
	card, err := validateCard(request.Card, time.Now())
	if err != nil {
		return 0, err
	}
	return s.walletRepository.ApplyTransaction(ctx, model.WalletTransaction{
		UserID:      userID,
		Amount:      request.Amount,
		Type:        "deposit",
		ReferenceID: fmt.Sprintf("card:%s:%s", card.paymentSystem, card.lastFour),
	})
}

func (s *WalletService) Charge(ctx context.Context, userID uuid.UUID, amount float64, orderID string) (float64, error) {
	if amount <= 0 {
		return 0, ErrInvalidAmount
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return 0, ErrReferenceRequired
	}
	return s.walletRepository.ApplyTransaction(ctx, model.WalletTransaction{
		UserID:      userID,
		Amount:      -amount,
		Type:        "purchase",
		ReferenceID: orderID,
	})
}
