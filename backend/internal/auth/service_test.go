package auth

import (
	"testing"
	"time"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("Password1!")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(hash, "Password1!") {
		t.Error("expected password to verify")
	}
	if VerifyPassword(hash, "wrong") {
		t.Error("expected wrong password to fail")
	}
}

func TestTokenPair(t *testing.T) {
	svc := NewService("test-secret-key-32-bytes-long!", "postchi", 15*time.Minute, 7*24*time.Hour)
	pair, hash, _, err := svc.GenerateTokenPair("user-1", "test@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Error("expected tokens")
	}
	if HashToken(pair.RefreshToken) != hash {
		t.Error("hash mismatch")
	}
	claims, err := svc.ParseAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("expected user-1, got %s", claims.UserID)
	}
}
