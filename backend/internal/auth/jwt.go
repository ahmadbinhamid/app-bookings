// Package auth is a minimal HS256 JWT implementation for this microservice. The
// main FlowPOS system signs tokens with the shared JWT_SECRET; this service only
// validates them (and can mint dev tokens for local testing). Claims carry the
// tenant + user identity so every request is attributable without a user table.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// Claims are the identity fields this microservice needs. Numeric IDs are
// unmarshalled from JSON numbers.
type Claims struct {
	TenantID  uint64 `json:"tenant_id"`
	UserID    uint64 `json:"user_id"`
	UserEmail string `json:"user_email"`
	IssuedAt  int64  `json:"iat,omitempty"`
	ExpiresAt int64  `json:"exp,omitempty"`
}

const headerB64 = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9" // {"alg":"HS256","typ":"JWT"}

func sign(signingInput, secret string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(signingInput))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

// Generate signs a JWT for the given claims. ttl==0 means no expiry.
func Generate(claims Claims, secret string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims.IssuedAt = now.Unix()
	if ttl > 0 {
		claims.ExpiresAt = now.Add(ttl).Unix()
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signingInput := headerB64 + "." + base64.RawURLEncoding.EncodeToString(payload)
	return signingInput + "." + sign(signingInput, secret), nil
}

// Parse validates the signature + expiry (HS256) and returns the claims.
func Parse(token, secret string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}
	signingInput := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(sign(signingInput, secret)), []byte(parts[2])) {
		return nil, ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, ErrInvalidToken
	}
	if claims.TenantID == 0 {
		return nil, ErrInvalidToken
	}
	if claims.ExpiresAt != 0 && time.Now().Unix() > claims.ExpiresAt {
		return nil, ErrExpiredToken
	}
	return &claims, nil
}
