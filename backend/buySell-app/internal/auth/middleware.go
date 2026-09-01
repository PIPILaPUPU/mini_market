package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID   uuid.UUID
	Username string
}

type tokenClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type Middleware struct {
	secret string
	issuer string
}

func NewMiddleware(secret, issuer string) *Middleware {
	return &Middleware{secret: secret, issuer: issuer}
}

func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeUnauthorized(w, "Bearer token is required")
			return
		}

		claims, err := m.parse(parts[1])
		if err != nil {
			writeUnauthorized(w, "access token is invalid or expired")
			return
		}

		ctx := context.WithValue(r.Context(), claimsContextKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) parse(rawToken string) (Claims, error) {
	parsed, err := jwt.ParseWithClaims(
		rawToken,
		&tokenClaims{},
		func(token *jwt.Token) (any, error) {
			return []byte(m.secret), nil
		},
		jwt.WithIssuer(m.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || !parsed.Valid {
		return Claims{}, jwt.ErrTokenInvalidClaims
	}

	tokenData, ok := parsed.Claims.(*tokenClaims)
	if !ok || tokenData.Username == "" {
		return Claims{}, jwt.ErrTokenInvalidClaims
	}
	userID, err := uuid.Parse(tokenData.Subject)
	if err != nil {
		return Claims{}, jwt.ErrTokenInvalidSubject
	}

	return Claims{UserID: userID, Username: tokenData.Username}, nil
}

type claimsContextKey struct{}

func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(Claims)
	return claims, ok
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    "invalid_token",
		"message": message,
	})
}
