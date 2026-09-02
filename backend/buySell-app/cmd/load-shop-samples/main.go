package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
	"unicode/utf8"

	"minimarket/buySell-app/internal/model"
	"minimarket/buySell-app/internal/repository"
	"minimarket/buySell-app/internal/service"
	"minimarket/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	fakeStoreURL   = "https://fakestoreapi.com/products?limit=10"
	maxNameRunes   = 255
	requestTimeout = 15 * time.Second
)

type fakeStoreProduct struct {
	Title       string  `json:"title"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	Image       string  `json:"image"`
}

func main() {
	if err := run(); err != nil {
		slog.Error("load shop samples failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	products, err := fetchProducts(ctx)
	if err != nil {
		return err
	}

	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	itemService := service.NewItemService(repository.NewPostgresBuySellRepository(pool))

	inserted := 0
	skipped := 0
	for _, product := range products {
		exists, err := itemExists(ctx, pool, product.Title)
		if err != nil {
			return err
		}
		if exists {
			skipped++
			continue
		}

		_, err = itemService.CreateItem(ctx, model.CreateItemRequest{
			Name:        product.Title,
			Description: product.Description,
			Price:       product.Price,
			ImageURL:    product.Image,
		})
		if err != nil {
			return fmt.Errorf("insert %q: %w", product.Title, err)
		}
		inserted++
	}

	slog.Info("shop samples loaded", "inserted", inserted, "skipped", skipped)
	return nil
}

func fetchProducts(ctx context.Context) ([]fakeStoreProduct, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fakeStoreURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build fake store request: %w", err)
	}

	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch fake store products: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fake store returned status %d", resp.StatusCode)
	}

	var products []fakeStoreProduct
	if err := json.NewDecoder(resp.Body).Decode(&products); err != nil {
		return nil, fmt.Errorf("decode fake store products: %w", err)
	}
	if len(products) > 10 {
		products = products[:10]
	}
	if len(products) == 0 {
		return nil, fmt.Errorf("fake store returned no products")
	}

	for i := range products {
		products[i].Title = clipRunes(products[i].Title, maxNameRunes)
	}
	return products, nil
}

func itemExists(ctx context.Context, pool *pgxpool.Pool, name string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM items WHERE name = $1)`, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check existing item: %w", err)
	}
	return exists, nil
}

func clipRunes(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit])
}
