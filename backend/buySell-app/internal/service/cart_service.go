package service

import (
	"context"
	"errors"

	"minimarket/buySell-app/internal/model"
	"minimarket/buySell-app/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrInvalidCartItemID = errors.New("item_id is required")
	ErrInvalidQuantity   = errors.New("quantity must be positive")
)

type CartService struct {
	cartRepository repository.CartRepository
}

func NewCartService(cartRepository repository.CartRepository) *CartService {
	return &CartService{cartRepository: cartRepository}
}

func (s *CartService) GetCart(ctx context.Context, userID uuid.UUID) ([]model.CartItem, error) {
	return s.cartRepository.GetCart(ctx, userID)
}

func (s *CartService) AddToCart(
	ctx context.Context,
	userID uuid.UUID,
	request model.AddToCartRequest,
) (*model.CartItem, error) {
	if request.ItemID == uuid.Nil {
		return nil, ErrInvalidCartItemID
	}
	if request.Quantity <= 0 {
		return nil, ErrInvalidQuantity
	}
	return s.cartRepository.AddToCart(ctx, userID, request.ItemID, request.Quantity)
}

func (s *CartService) RemoveFromCart(ctx context.Context, userID, itemID uuid.UUID) error {
	if itemID == uuid.Nil {
		return ErrInvalidCartItemID
	}
	return s.cartRepository.RemoveFromCart(ctx, userID, itemID)
}
