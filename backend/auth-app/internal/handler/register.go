package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"minimarket/auth-app/internal/model"
	"minimarket/auth-app/internal/repository"
	"minimarket/auth-app/internal/service"

	"github.com/google/uuid"
)

const refreshCookieName = "refresh_token"

type authService interface {
	Register(context.Context, model.RegisterRequest) (service.AuthResult, error)
	Login(context.Context, model.LoginRequest) (service.AuthResult, error)
	Refresh(context.Context, string) (service.AuthResult, error)
	Logout(context.Context, string) error
	ParseAccessToken(string) (model.Claims, error)
	UserByID(context.Context, uuid.UUID) (model.User, error)
}

type CookieConfig struct {
	Secure   bool
	SameSite http.SameSite
	TTL      time.Duration
}

type RegisterHandler struct {
	service authService
	cookie  CookieConfig
}

func NewRegisterHandler(authService authService, cookie CookieConfig) *RegisterHandler {
	return &RegisterHandler{service: authService, cookie: cookie}
}

func (h *RegisterHandler) Register(w http.ResponseWriter, r *http.Request) {
	var request model.RegisterRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := h.service.Register(r.Context(), request)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.setRefreshCookie(w, result.RefreshToken)
	writeJSON(w, http.StatusCreated, result.Response)
}

func (h *RegisterHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request model.LoginRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := h.service.Login(r.Context(), request)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.setRefreshCookie(w, result.RefreshToken)
	writeJSON(w, http.StatusOK, result.Response)
}

func (h *RegisterHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_token", "refresh token is missing")
		return
	}
	result, err := h.service.Refresh(r.Context(), cookie.Value)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.setRefreshCookie(w, result.RefreshToken)
	writeJSON(w, http.StatusOK, result.Response)
}

func (h *RegisterHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_token", "refresh token is missing")
		return
	}
	if err := h.service.Logout(r.Context(), cookie.Value); err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *RegisterHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_token", "access token is invalid")
		return
	}
	user, err := h.service.UserByID(r.Context(), claims.UserID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *RegisterHandler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, service.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "username or password is incorrect")
	case errors.Is(err, service.ErrInvalidToken):
		writeError(w, http.StatusUnauthorized, "invalid_token", "token is invalid or expired")
	case errors.Is(err, repository.ErrUsernameExists):
		writeError(w, http.StatusConflict, "username_exists", "username is already registered")
	case errors.Is(err, repository.ErrEmailExists):
		writeError(w, http.StatusConflict, "email_exists", "email is already registered")
	default:
		slog.Error("auth request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func (h *RegisterHandler) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/auth",
		HttpOnly: true,
		Secure:   h.cookie.Secure,
		SameSite: h.cookie.SameSite,
		MaxAge:   int(h.cookie.TTL.Seconds()),
	})
}

func (h *RegisterHandler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Path:     "/auth",
		HttpOnly: true,
		Secure:   h.cookie.Secure,
		SameSite: h.cookie.SameSite,
		MaxAge:   -1,
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("write JSON response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"error": code, "message": message})
}

type claimsContextKey struct{}

func ClaimsFromContext(ctx context.Context) (model.Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(model.Claims)
	return claims, ok
}

func (h *RegisterHandler) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeError(w, http.StatusUnauthorized, "invalid_token", "Bearer token is required")
			return
		}
		claims, err := h.service.ParseAccessToken(parts[1])
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid_token", "access token is invalid or expired")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsContextKey{}, claims)))
	})
}
