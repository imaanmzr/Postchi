package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrUnauthorized       = errors.New("unauthorized")
)

type contextKey string

const UserIDKey contextKey = "userID"

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type Service struct {
	secret          []byte
	issuer          string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewService(secret, issuer string, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{
		secret:          []byte(secret),
		issuer:          issuer,
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}
}

func (s *Service) GenerateTokenPair(userID, email string) (*TokenPair, string, time.Time, error) {
	now := time.Now()
	accessClaims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTokenTTL)),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessSigned, err := accessToken.SignedString(s.secret)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	refreshRaw, err := generateRandomToken(32)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	expiresAt := now.Add(s.refreshTokenTTL)
	return &TokenPair{
		AccessToken:  accessSigned,
		RefreshToken: refreshRaw,
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
	}, HashToken(refreshRaw), expiresAt, nil
}

func (s *Service) ParseAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=1,p=4$%s$%s",
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) < 6 {
		return false
	}
	// $argon2id$v=19$m=65536,t=1,p=4$salt$hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	var m, t uint32
	var p uint8
	_, _ = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p)
	actual := argon2.IDKey([]byte(password), salt, t, m, p, uint32(len(expected)))
	if len(actual) != len(expected) {
		return false
	}
	var diff byte
	for i := range actual {
		diff |= actual[i] ^ expected[i]
	}
	return diff == 0
}

func generateRandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func RequireAuth(s *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""
			header := r.Header.Get("Authorization")
			if header != "" {
				parts := strings.SplitN(header, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
					token = parts[1]
				}
			}
			if token == "" && strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
				token = r.URL.Query().Get("access_token")
			}
			if token == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			claims, err := s.ParseAccessToken(token)
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	raw, ok := ctx.Value(UserIDKey).(string)
	if !ok || raw == "" {
		return uuid.Nil, ErrUnauthorized
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, ErrUnauthorized
	}
	return id, nil
}

// AuthProvider interface for future OIDC/SSO support.
type AuthProvider interface {
	Name() string
	Authenticate(ctx context.Context, credentials any) (userID uuid.UUID, email string, err error)
}

type LocalAuthProvider struct{}

func (LocalAuthProvider) Name() string { return "local" }
