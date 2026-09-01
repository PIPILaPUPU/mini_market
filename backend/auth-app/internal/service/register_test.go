package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"minimarket/auth-app/internal/model"
	"minimarket/auth-app/internal/repository"

	"github.com/google/uuid"
)

type fakeSession struct {
	model.RefreshSession
	revoked bool
}

type fakeRepository struct {
	users    map[uuid.UUID]model.User
	sessions map[string]fakeSession
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		users:    make(map[uuid.UUID]model.User),
		sessions: make(map[string]fakeSession),
	}
}

func (r *fakeRepository) CreateUserWithSession(
	_ context.Context,
	user model.User,
	session model.RefreshSession,
) (model.User, error) {
	for _, existing := range r.users {
		if strings.EqualFold(existing.Username, user.Username) {
			return model.User{}, repository.ErrUsernameExists
		}
		if strings.EqualFold(existing.Email, user.Email) {
			return model.User{}, repository.ErrEmailExists
		}
	}
	now := time.Now()
	user.CreatedAt, user.UpdatedAt = now, now
	r.users[user.ID] = user
	session.UserID = user.ID
	r.sessions[session.TokenHash] = fakeSession{RefreshSession: session}
	return user, nil
}

func (r *fakeRepository) FindUserByUsername(_ context.Context, username string) (model.User, error) {
	for _, user := range r.users {
		if strings.EqualFold(user.Username, username) {
			return user, nil
		}
	}
	return model.User{}, repository.ErrNotFound
}

func (r *fakeRepository) FindUserByID(_ context.Context, id uuid.UUID) (model.User, error) {
	user, ok := r.users[id]
	if !ok {
		return model.User{}, repository.ErrNotFound
	}
	return user, nil
}

func (r *fakeRepository) CreateRefreshSession(_ context.Context, session model.RefreshSession) error {
	r.sessions[session.TokenHash] = fakeSession{RefreshSession: session}
	return nil
}

func (r *fakeRepository) RotateRefreshSession(
	_ context.Context,
	oldHash string,
	replacement model.RefreshSession,
) (model.User, error) {
	old, ok := r.sessions[oldHash]
	if !ok || old.revoked || !old.ExpiresAt.After(time.Now()) {
		return model.User{}, repository.ErrInvalidSession
	}
	old.revoked = true
	r.sessions[oldHash] = old
	replacement.UserID = old.UserID
	r.sessions[replacement.TokenHash] = fakeSession{RefreshSession: replacement}
	return r.users[old.UserID], nil
}

func (r *fakeRepository) RevokeRefreshSession(_ context.Context, hash string) error {
	session, ok := r.sessions[hash]
	if !ok || session.revoked {
		return repository.ErrInvalidSession
	}
	session.revoked = true
	r.sessions[hash] = session
	return nil
}

func newTestService(repo repository.AuthRepository) *RegisterService {
	return NewRegisterService(repo, Config{
		JWTSecret:  "01234567890123456789012345678901",
		JWTIssuer:  "test-auth",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: time.Hour,
	})
}

func validRegistration() model.RegisterRequest {
	return model.RegisterRequest{
		Username:  "stepan_1",
		Email:     "stepan@example.com",
		Password:  "correct horse battery staple",
		FirstName: "Stepan",
	}
}

func TestRegisterLoginAndAccessToken(t *testing.T) {
	repo := newFakeRepository()
	auth := newTestService(repo)

	registered, err := auth.Register(context.Background(), validRegistration())
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if registered.RefreshToken == "" || registered.Response.AccessToken == "" {
		t.Fatal("expected access and refresh tokens")
	}
	stored := repo.users[registered.Response.User.ID]
	if stored.PasswordHash == validRegistration().Password || stored.PasswordHash == "" {
		t.Fatal("password was not securely hashed")
	}
	if registered.Response.User.PasswordHash != "" {
		t.Fatal("password hash leaked in response")
	}

	claims, err := auth.ParseAccessToken(registered.Response.AccessToken)
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	if claims.UserID != registered.Response.User.ID || claims.Username != "stepan_1" {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	loggedIn, err := auth.Login(context.Background(), model.LoginRequest{
		Username: "STEPAN_1",
		Password: validRegistration().Password,
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if loggedIn.RefreshToken == registered.RefreshToken {
		t.Fatal("login must create an independent refresh token")
	}
}

func TestRegisterRejectsInvalidAndDuplicateData(t *testing.T) {
	repo := newFakeRepository()
	auth := newTestService(repo)

	invalid := validRegistration()
	invalid.Username = "!"
	if _, err := auth.Register(context.Background(), invalid); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
	if _, err := auth.Register(context.Background(), validRegistration()); err != nil {
		t.Fatalf("first register: %v", err)
	}
	duplicate := validRegistration()
	duplicate.Email = "other@example.com"
	if _, err := auth.Register(context.Background(), duplicate); !errors.Is(err, repository.ErrUsernameExists) {
		t.Fatalf("expected duplicate username, got %v", err)
	}
}

func TestLoginRejectsWrongCredentials(t *testing.T) {
	auth := newTestService(newFakeRepository())
	if _, err := auth.Login(context.Background(), model.LoginRequest{
		Username: "missing",
		Password: "wrong-password",
	}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestRefreshRotationReplayAndLogout(t *testing.T) {
	auth := newTestService(newFakeRepository())
	registered, err := auth.Register(context.Background(), validRegistration())
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	refreshed, err := auth.Refresh(context.Background(), registered.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if refreshed.RefreshToken == registered.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}
	if _, err := auth.Refresh(context.Background(), registered.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected replay rejection, got %v", err)
	}
	if err := auth.Logout(context.Background(), refreshed.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := auth.Refresh(context.Background(), refreshed.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected revoked token rejection, got %v", err)
	}
}

func TestExpiredAccessTokenIsRejected(t *testing.T) {
	auth := newTestService(newFakeRepository())
	now := time.Now()
	auth.now = func() time.Time { return now }
	result, err := auth.Register(context.Background(), validRegistration())
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	auth.now = func() time.Time { return now.Add(16 * time.Minute) }
	if _, err := auth.ParseAccessToken(result.Response.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected expired token, got %v", err)
	}
}
