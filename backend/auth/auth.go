// Package auth issues and verifies stateless session tokens for the panel.
//
// Credentials are checked against the database (Argon2id) by the login handler;
// this package only mints and validates a signed token whose subject is the
// authenticated user's ID. The token is HMAC-SHA256 signed (JWT-like:
// payload.signature) and verified by the Auth middleware on every protected
// request, so no external JWT dependency is required.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"
)

// Errors returned by Validate.
var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// Service issues and verifies session tokens.
type Service struct {
	secret []byte
	ttl    time.Duration
}

// claims is the signed token payload. Sub is the user ID as a decimal string.
type claims struct {
	Sub string `json:"sub"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
}

// NewService builds a Service from the signing secret and token TTL. An empty
// secret triggers an ephemeral random key (sessions reset on restart); set
// AUTH_TOKEN_SECRET to keep tokens valid across restarts. A non-positive ttl
// falls back to 24h.
func NewService(secret string, ttl time.Duration) *Service {
	key := []byte(secret)
	if strings.TrimSpace(secret) == "" {
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			// Refuse to run with a predictable signing key.
			panic("auth: failed to generate token secret: " + err.Error())
		}
		log.Println("AUTH_TOKEN_SECRET is empty; generated an ephemeral token secret (sessions reset on restart). Set AUTH_TOKEN_SECRET to persist sessions.")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &Service{secret: key, ttl: ttl}
}

// TTL is the lifetime of issued tokens.
func (s *Service) TTL() time.Duration { return s.ttl }

// Issue returns a signed token (subject = userID) and its expiry from now.
func (s *Service) Issue(userID uint, now time.Time) (token string, expiresAt time.Time, err error) {
	expiresAt = now.Add(s.ttl)
	body, err := json.Marshal(claims{Sub: strconv.FormatUint(uint64(userID), 10), Iat: now.Unix(), Exp: expiresAt.Unix()})
	if err != nil {
		return "", time.Time{}, err
	}
	payload := base64.RawURLEncoding.EncodeToString(body)
	return payload + "." + s.sign(payload), expiresAt, nil
}

// Validate verifies the token signature and expiry, returning the user ID.
func (s *Service) Validate(token string, now time.Time) (uint, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return 0, ErrInvalidToken
	}
	expected := s.sign(parts[0])
	if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(expected)) != 1 {
		return 0, ErrInvalidToken
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, ErrInvalidToken
	}
	var c claims
	if err := json.Unmarshal(body, &c); err != nil {
		return 0, ErrInvalidToken
	}
	if now.Unix() >= c.Exp {
		return 0, ErrExpiredToken
	}
	id, err := strconv.ParseUint(c.Sub, 10, 64)
	if err != nil || id == 0 {
		return 0, ErrInvalidToken
	}
	return uint(id), nil
}

func (s *Service) sign(payload string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
