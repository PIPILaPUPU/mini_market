package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"minimarket/auth-app/internal/model"
	"minimarket/auth-app/internal/service"

	"github.com/google/uuid"
)

type fakeAuthService struct {
	registerResult service.AuthResult
	registerErr    error
	claims         model.Claims
	parseErr       error
	user           model.User
}

func (f *fakeAuthService) Register(context.Context, model.RegisterRequest) (service.AuthResult, error) {
	return f.registerResult, f.registerErr
}
func (f *fakeAuthService) Login(context.Context, model.LoginRequest) (service.AuthResult, error) {
	return service.AuthResult{}, nil
}
func (f *fakeAuthService) Refresh(context.Context, string) (service.AuthResult, error) {
	return service.AuthResult{}, nil
}
func (f *fakeAuthService) Logout(context.Context, string) error { return nil }
func (f *fakeAuthService) ParseAccessToken(string) (model.Claims, error) {
	return f.claims, f.parseErr
}
func (f *fakeAuthService) UserByID(context.Context, uuid.UUID) (model.User, error) {
	return f.user, nil
}

func TestRegisterSetsRefreshCookie(t *testing.T) {
	userID := uuid.New()
	fake := &fakeAuthService{registerResult: service.AuthResult{
		Response: model.AuthResponse{
			User:        model.User{ID: userID, Username: "stepan"},
			AccessToken: "access",
			TokenType:   "Bearer",
			ExpiresAt:   time.Now().Add(time.Minute),
		},
		RefreshToken: "refresh",
	}}
	handler := NewRegisterHandler(fake, CookieConfig{TTL: time.Hour, SameSite: http.SameSiteLaxMode})
	request := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(
		`{"username":"stepan","email":"stepan@example.com","password":"password123"}`,
	))
	response := httptest.NewRecorder()

	handler.Register(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != refreshCookieName || cookies[0].Value != "refresh" {
		t.Fatalf("unexpected refresh cookie: %+v", cookies)
	}
	if !cookies[0].HttpOnly || cookies[0].Path != "/auth" {
		t.Fatalf("refresh cookie security attributes are missing: %+v", cookies[0])
	}
}

func TestRegisterRejectsUnknownJSONField(t *testing.T) {
	handler := NewRegisterHandler(&fakeAuthService{}, CookieConfig{})
	request := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(
		`{"username":"stepan","unexpected":true}`,
	))
	response := httptest.NewRecorder()

	handler.Register(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
}

func TestAuthenticateProtectsMe(t *testing.T) {
	userID := uuid.New()
	fake := &fakeAuthService{
		claims: model.Claims{UserID: userID, Username: "stepan"},
		user:   model.User{ID: userID, Username: "stepan"},
	}
	handler := NewRegisterHandler(fake, CookieConfig{})
	protected := handler.Authenticate(http.HandlerFunc(handler.Me))

	unauthorized := httptest.NewRecorder()
	protected.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/auth/me", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected missing token to return 401, got %d", unauthorized.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()
	protected.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), userID.String()) {
		t.Fatalf("unexpected protected response: %d %s", response.Code, response.Body.String())
	}
}

func TestAuthenticateRejectsInvalidToken(t *testing.T) {
	handler := NewRegisterHandler(&fakeAuthService{parseErr: errors.New("invalid")}, CookieConfig{})
	protected := handler.Authenticate(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler must not be called")
	}))
	request := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	request.Header.Set("Authorization", "Bearer invalid")
	response := httptest.NewRecorder()

	protected.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}
