package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	buysellauth "minimarket/buySell-app/internal/auth"
	"minimarket/buySell-app/internal/config"
	"minimarket/buySell-app/internal/handler"
	"minimarket/buySell-app/internal/repository"
	"minimarket/buySell-app/internal/service"
	"minimarket/database"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	if err := run(); err != nil {
		slog.Error("buySell service stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	startupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := database.Open(startupCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/items_health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	itemRepository := repository.NewPostgresBuySellRepository(db)
	itemService := service.NewItemService(itemRepository)
	itemHandler := handler.NewItemHandler(itemService)

	walletRepository := repository.NewPostgresWalletRepository(db)
	walletService := service.NewWalletService(walletRepository)
	walletHandler := handler.NewWalletHandler(walletService)

	cartRepository := repository.NewPostgresCartRepository(db)
	cartService := service.NewCartService(cartRepository)
	cartHandler := handler.NewCartHandler(cartService)

	authMiddleware := buysellauth.NewMiddleware(cfg.JWTSecret, cfg.JWTIssuer)

	r.Get("/items/{id}", itemHandler.GetItem)
	r.Get("/items", itemHandler.GetItemsList)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		//items
		r.Post("/items", itemHandler.CreateItem)
		//wallet
		r.Get("/wallet", walletHandler.GetBalance)
		r.Get("/wallet/transactions", walletHandler.GetTransactionsList)
		r.Post("/wallet/deposit", walletHandler.Deposit)
		r.Post("/wallet/charge", walletHandler.Charge)
		//cart
		r.Get("/cart", cartHandler.GetCart)
		r.Post("/cart", cartHandler.AddToCart)
		r.Delete("/cart/{itemID}", cartHandler.RemoveFromCart)
	})

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("buySell service listening", "address", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-stop:
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	return server.Shutdown(shutdownCtx)
}
