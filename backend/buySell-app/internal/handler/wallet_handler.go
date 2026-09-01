package handler

import (
	"errors"
	"net/http"

	"minimarket/buySell-app/internal/auth"
	"minimarket/buySell-app/internal/model"
	"minimarket/buySell-app/internal/repository"
	"minimarket/buySell-app/internal/service"
)

type WalletHandler struct {
	walletService *service.WalletService
}

func NewWalletHandler(walletService *service.WalletService) *WalletHandler {
	return &WalletHandler{walletService: walletService}
}

func (h *WalletHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_request", "invalid user id")
		return
	}
	balance, err := h.walletService.GetBalance(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_server_error", "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, model.BalanceResponse{Balance: balance})
}

func (h *WalletHandler) GetTransactionsList(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_request", "invalid user id")
		return
	}

	list, err := h.walletService.TransactionsList(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_server_error", "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *WalletHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_request", "invalid user id")
		return
	}

	var request model.DepositRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	balance, err := h.walletService.Deposit(r.Context(), claims.UserID, request.Amount)
	if err != nil {
		h.writeWalletError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, model.BalanceResponse{Balance: balance})
}

func (h *WalletHandler) Charge(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_request", "invalid user id")
		return
	}

	var request model.ChargeRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	balance, err := h.walletService.Charge(r.Context(), claims.UserID, request.Amount, request.ReferenceID)
	if err != nil {
		h.writeWalletError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, model.BalanceResponse{Balance: balance})
}

func (h *WalletHandler) writeWalletError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidAmount),
		errors.Is(err, service.ErrReferenceRequired):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, repository.ErrInsufficientFunds):
		writeError(w, http.StatusConflict, "insufficient_funds", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_server_error", "internal server error")
	}
}
