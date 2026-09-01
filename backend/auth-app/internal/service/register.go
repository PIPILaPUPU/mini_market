package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"minimarket/auth-app/internal/model"
	"minimarket/auth-app/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

// Hash for a fixed non-secret value, used to reduce username enumeration timing differences.
const dummyPasswordHash = "$2a$10$7EqJtq98hPqEX7fNZaFWoO5uF2e5.ZYRxG9rH8QqlWl1KSFMyY6e."

type Config struct {
	JWTSecret  string
	JWTIssuer  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type AuthResult struct {
	Response     model.AuthResponse
	RefreshToken string
}

type tokenClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type RegisterService struct {
	repository repository.AuthRepository
	config     Config
	now        func() time.Time
}

func NewRegisterService(repo repository.AuthRepository, config Config) *RegisterService {
	return &RegisterService{repository: repo, config: config, now: time.Now}
}

func (s *RegisterService) Register(ctx context.Context, request model.RegisterRequest) (AuthResult, error) {
	request.Username = strings.TrimSpace(request.Username)
	request.Email = strings.ToLower(strings.TrimSpace(request.Email))
	request.FirstName = strings.TrimSpace(request.FirstName)
	request.LastName = strings.TrimSpace(request.LastName)
	if err := validateRegistration(request); err != nil {
		return AuthResult{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hash password: %w", err)
	}
	refreshToken, tokenHash, err := generateRefreshToken()
	if err != nil {
		return AuthResult{}, err
	}
	now := s.now()
	user, err := s.repository.CreateUserWithSession(ctx, model.User{
		ID:           uuid.New(),
		Username:     request.Username,
		Email:        request.Email,
		PasswordHash: string(passwordHash),
		FirstName:    request.FirstName,
		LastName:     request.LastName,
	}, model.RefreshSession{
		ID:        uuid.New(),
		TokenHash: tokenHash,
		ExpiresAt: now.Add(s.config.RefreshTTL),
	})
	if err != nil {
		return AuthResult{}, err
	}
	response, err := s.newAccessResponse(user, now)
	if err != nil {
		return AuthResult{}, err
	}
	return AuthResult{Response: response, RefreshToken: refreshToken}, nil
}

func (s *RegisterService) Login(ctx context.Context, request model.LoginRequest) (AuthResult, error) {
	username := strings.TrimSpace(request.Username)
	user, err := s.repository.FindUserByUsername(ctx, username)
	if errors.Is(err, repository.ErrNotFound) {
		_ = bcrypt.CompareHashAndPassword([]byte(dummyPasswordHash), []byte(request.Password))
		return AuthResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return AuthResult{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)) != nil {
		return AuthResult{}, ErrInvalidCredentials
	}
	return s.newSession(ctx, user)
}

func (s *RegisterService) Refresh(ctx context.Context, refreshToken string) (AuthResult, error) {
	if refreshToken == "" {
		return AuthResult{}, ErrInvalidToken
	}
	rawToken, hash, err := generateRefreshToken()
	if err != nil {
		return AuthResult{}, err
	}
	now := s.now()
	user, err := s.repository.RotateRefreshSession(ctx, hashToken(refreshToken), model.RefreshSession{
		ID:        uuid.New(),
		TokenHash: hash,
		ExpiresAt: now.Add(s.config.RefreshTTL),
	})
	if errors.Is(err, repository.ErrInvalidSession) {
		return AuthResult{}, ErrInvalidToken
	}
	if err != nil {
		return AuthResult{}, err
	}
	response, err := s.newAccessResponse(user, now)
	if err != nil {
		return AuthResult{}, err
	}
	return AuthResult{Response: response, RefreshToken: rawToken}, nil
}

func (s *RegisterService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return ErrInvalidToken
	}
	err := s.repository.RevokeRefreshSession(ctx, hashToken(refreshToken))
	if errors.Is(err, repository.ErrInvalidSession) {
		return ErrInvalidToken
	}
	return err
}

func (s *RegisterService) ParseAccessToken(rawToken string) (model.Claims, error) {
	parsed, err := jwt.ParseWithClaims(rawToken, &tokenClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return []byte(s.config.JWTSecret), nil
	}, jwt.WithIssuer(s.config.JWTIssuer), jwt.WithExpirationRequired(), jwt.WithTimeFunc(s.now))
	if err != nil || !parsed.Valid {
		return model.Claims{}, ErrInvalidToken
	}
	claims, ok := parsed.Claims.(*tokenClaims)
	if !ok {
		return model.Claims{}, ErrInvalidToken
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil || claims.Username == "" {
		return model.Claims{}, ErrInvalidToken
	}
	return model.Claims{UserID: userID, Username: claims.Username}, nil
}

func (s *RegisterService) UserByID(ctx context.Context, id uuid.UUID) (model.User, error) {
	user, err := s.repository.FindUserByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return model.User{}, ErrInvalidToken
	}
	return publicUser(user), err
}

func (s *RegisterService) newSession(ctx context.Context, user model.User) (AuthResult, error) {
	refreshToken, tokenHash, err := generateRefreshToken()
	if err != nil {
		return AuthResult{}, err
	}
	now := s.now()
	if err := s.repository.CreateRefreshSession(ctx, model.RefreshSession{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(s.config.RefreshTTL),
	}); err != nil {
		return AuthResult{}, err
	}
	response, err := s.newAccessResponse(user, now)
	if err != nil {
		return AuthResult{}, err
	}
	return AuthResult{Response: response, RefreshToken: refreshToken}, nil
}

func (s *RegisterService) newAccessResponse(user model.User, now time.Time) (model.AuthResponse, error) {
	expiresAt := now.Add(s.config.AccessTTL)
	claims := tokenClaims{
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			Issuer:    s.config.JWTIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        uuid.NewString(),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return model.AuthResponse{}, fmt.Errorf("sign access token: %w", err)
	}
	return model.AuthResponse{
		User:        publicUser(user),
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
	}, nil
}

func generateRefreshToken() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	rawToken := base64.RawURLEncoding.EncodeToString(bytes)
	return rawToken, hashToken(rawToken), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func validateRegistration(request model.RegisterRequest) error {
	if !usernamePattern.MatchString(request.Username) {
		return fmt.Errorf("%w: username must contain 3-32 letters, digits or underscores", ErrInvalidInput)
	}
	address, err := mail.ParseAddress(request.Email)
	if err != nil || !strings.EqualFold(address.Address, request.Email) {
		return fmt.Errorf("%w: invalid email", ErrInvalidInput)
	}
	if len(request.Password) < 8 || len(request.Password) > 72 {
		return fmt.Errorf("%w: password must contain 8-72 bytes", ErrInvalidInput)
	}
	if len(request.FirstName) > 100 || len(request.LastName) > 100 {
		return fmt.Errorf("%w: name is too long", ErrInvalidInput)
	}
	return nil
}

func publicUser(user model.User) model.User {
	user.PasswordHash = ""
	return user
}
