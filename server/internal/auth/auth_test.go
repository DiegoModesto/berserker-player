package auth

import (
	"testing"
	"time"
)

func TestPasswordHashVerify(t *testing.T) {
	hash, err := HashPassword("s3cr3t")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyPassword("s3cr3t", hash)
	if err != nil || !ok {
		t.Fatalf("senha correta deveria validar: ok=%v err=%v", ok, err)
	}
	ok, _ = VerifyPassword("errada", hash)
	if ok {
		t.Fatal("senha errada não deveria validar")
	}
}

func TestAccessTokenRoundTrip(t *testing.T) {
	svc := NewService("test-secret", time.Minute, time.Minute, time.Hour)
	now := time.Now()
	tok, _, err := svc.AccessToken("user-1", true, now)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := svc.ParseAccess(tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "user-1" || !claims.IsAdmin {
		t.Fatalf("claims inesperadas: %+v", claims)
	}
	// Um access token não deve validar como media token.
	if _, err := svc.ParseMedia(tok); err == nil {
		t.Fatal("access token não deveria passar como media token")
	}
}

func TestMediaTokenKind(t *testing.T) {
	svc := NewService("secret", time.Minute, time.Minute, time.Hour)
	tok, _, _ := svc.MediaToken("u", time.Now())
	if _, err := svc.ParseMedia(tok); err != nil {
		t.Fatalf("media token deveria validar: %v", err)
	}
	if _, err := svc.ParseAccess(tok); err == nil {
		t.Fatal("media token não deveria passar como access")
	}
}

func TestExpiredToken(t *testing.T) {
	svc := NewService("secret", time.Minute, time.Minute, time.Hour)
	past := time.Now().Add(-2 * time.Hour)
	tok, _, _ := svc.AccessToken("u", false, past)
	if _, err := svc.ParseAccess(tok); err == nil {
		t.Fatal("token expirado deveria falhar")
	}
}

func TestRefreshTokenHashing(t *testing.T) {
	raw, hash, err := NewRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	if HashToken(raw) != hash {
		t.Fatal("hash do refresh token inconsistente")
	}
}
