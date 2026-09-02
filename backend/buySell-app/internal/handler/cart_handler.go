package handler

import (
	"errors"
	"net/http"

	"minimarket/buySell-app/internal/auth"
	"minimarket/buySell-app/internal/model"
	"minimarket/buySell-app/internal/repository"
	"minimarket/buySell-app/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CartHandler struct {
	cartService *service.CartService
}

func NewCartHandler(cartService *service.CartService) *CartHandler {
	return &CartHandler{cartService: cartService}
}

func (h *CartHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	cart, err := h.cartService.GetCart(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_server_error", "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

func (h *CartHandler) AddToCart(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}

	var request model.AddToCartRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	cartItem, err := h.cartService.AddToCart(r.Context(), claims.UserID, request)
	if err != nil {
		h.writeCartError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, cartItem)
}

func (h *CartHandler) RemoveFromCart(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}

	itemID, err := uuid.Parse(chi.URLParam(r, "itemID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid item id")
		return
	}

	err = h.cartService.RemoveFromCart(r.Context(), claims.UserID, itemID)
	if err != nil {
		h.writeCartError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CartHandler) writeCartError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidCartItemID),
		errors.Is(err, service.ErrInvalidQuantity):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, "item_not_found", "item not found")
	case errors.Is(err, repository.ErrCartItemNotFound):
		writeError(w, http.StatusNotFound, "cart_item_not_found", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_server_error", "internal server error")
	}
}
