package handler

import (
	"errors"
	"net/http"

	"minimarket/buySell-app/internal/model"
	"minimarket/buySell-app/internal/repository"
	"minimarket/buySell-app/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ItemHandler struct {
	service *service.ItemService
}

func NewItemHandler(service *service.ItemService) *ItemHandler {
	return &ItemHandler{service: service}
}

func (h *ItemHandler) GetItem(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid item id")
		return
	}
	item, err := h.service.GetItem(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_server_error", "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *ItemHandler) GetItemsList(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.GetItemsList(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_server_error", "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *ItemHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	var request model.CreateItemRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	item, err := h.service.CreateItem(r.Context(), request)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
