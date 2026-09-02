package service

import (
	"context"
	"errors"
	"strings"

	"minimarket/buySell-app/internal/model"
	"minimarket/buySell-app/internal/repository"

	"github.com/google/uuid"
)

type ItemService struct {
	repository repository.BuySellRepository
}

func NewItemService(repository repository.BuySellRepository) *ItemService {
	return &ItemService{repository: repository}
}

func (s *ItemService) GetItem(ctx context.Context, id uuid.UUID) (*model.Item, error) {
	return s.repository.GetItem(ctx, id)
}

func (s *ItemService) GetItemsList(ctx context.Context) ([]*model.Item, error) {
	return s.repository.GetItemsList(ctx)
}

func (s *ItemService) CreateItem(ctx context.Context, item model.CreateItemRequest) (*model.Item, error) {
	name := strings.TrimSpace(item.Name)
	desc := strings.TrimSpace(item.Description)
	price := item.Price

	if name == "" {
		return nil, errors.New("name is required")
	}
	if desc == "" {
		return nil, errors.New("description is required")
	}
	if price <= 0 {
		return nil, errors.New("price must be greater than 0")
	}

	item.Name = name
	item.Description = desc
	item.ImageURL = strings.TrimSpace(item.ImageURL)
	return s.repository.CreateItem(ctx, item)
}
