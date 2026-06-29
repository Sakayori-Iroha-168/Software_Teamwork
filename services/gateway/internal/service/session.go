package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const SessionCacheKeyPrefix = "gateway:session:"

type UserSummary struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

type SessionSummary struct {
	SessionID   string    `json:"sessionId"`
	AccessToken string    `json:"accessToken"`
	TokenType   string    `json:"tokenType"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

type SessionIdentity struct {
	User    UserSummary    `json:"user"`
	Session SessionSummary `json:"session"`
}

type GatewaySessionCacheEntry struct {
	SessionID       string    `json:"session_id"`
	UserID          string    `json:"user_id"`
	Username        string    `json:"username"`
	Roles           []string  `json:"roles"`
	Permissions     []string  `json:"permissions"`
	TokenType       string    `json:"token_type"`
	AccessTokenHash string    `json:"access_token_hash"`
	IssuedAt        time.Time `json:"issued_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	CachedAt        time.Time `json:"cached_at"`
	RequestID       string    `json:"request_id"`
}

type AuthContext struct {
	RequestID   string
	SessionID   string
	UserID      string
	Username    string
	Roles       []string
	Permissions []string
	ExpiresAt   time.Time
}

type SessionStore interface {
	Save(context.Context, GatewaySessionCacheEntry, time.Duration) error
	Get(context.Context, string) (GatewaySessionCacheEntry, error)
	Delete(context.Context, string) error
	Ping(context.Context) error
}

type AuthClient interface {
	CreateUser(context.Context, string, string, string) (SessionIdentity, error)
	CreateSession(context.Context, string, string, string) (SessionIdentity, error)
	DeleteSession(context.Context, string, string) error
}

type TokenHasher struct {
	secret     []byte
	keyVersion string
}

func NewTokenHasher(secret string, keyVersion string) (TokenHasher, error) {
	secret = strings.TrimSpace(secret)
	keyVersion = strings.TrimSpace(keyVersion)
	if secret == "" {
		return TokenHasher{}, fmt.Errorf("token hash secret is required")
	}
	if keyVersion == "" {
		keyVersion = "v1"
	}
	return TokenHasher{secret: []byte(secret), keyVersion: keyVersion}, nil
}

func (h TokenHasher) Hash(accessToken string) (string, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return "", fmt.Errorf("access token is required")
	}
	mac := hmac.New(sha256.New, h.secret)
	_, _ = mac.Write([]byte(accessToken))
	return "hmac-sha256:" + h.keyVersion + ":" + hex.EncodeToString(mac.Sum(nil)), nil
}

func SessionCacheKey(accessTokenHash string) string {
	return SessionCacheKeyPrefix + accessTokenHash
}

func NewCacheEntry(identity SessionIdentity, accessTokenHash string, requestID string, now time.Time) (GatewaySessionCacheEntry, time.Duration, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	entry := GatewaySessionCacheEntry{
		SessionID:       strings.TrimSpace(identity.Session.SessionID),
		UserID:          strings.TrimSpace(identity.User.ID),
		Username:        strings.TrimSpace(identity.User.Username),
		Roles:           normalizeStrings(identity.User.Roles),
		Permissions:     normalizeStrings(identity.User.Permissions),
		TokenType:       strings.TrimSpace(identity.Session.TokenType),
		AccessTokenHash: strings.TrimSpace(accessTokenHash),
		ExpiresAt:       identity.Session.ExpiresAt.UTC(),
		IssuedAt:        now,
		CachedAt:        now,
		RequestID:       requestID,
	}
	if entry.TokenType == "" {
		entry.TokenType = "Bearer"
	}
	if err := entry.Validate(accessTokenHash, now); err != nil {
		return GatewaySessionCacheEntry{}, 0, err
	}
	return entry, entry.ExpiresAt.Sub(now), nil
}

func (e GatewaySessionCacheEntry) Validate(accessTokenHash string, now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if e.SessionID == "" || e.UserID == "" || e.Username == "" || e.AccessTokenHash == "" || e.ExpiresAt.IsZero() {
		return ErrMalformedSession
	}
	if e.AccessTokenHash != accessTokenHash {
		return ErrMalformedSession
	}
	if !e.ExpiresAt.After(now) {
		return ErrSessionNotFound
	}
	return nil
}

func (e GatewaySessionCacheEntry) AuthContext(requestID string) AuthContext {
	return AuthContext{
		RequestID:   requestID,
		SessionID:   e.SessionID,
		UserID:      e.UserID,
		Username:    e.Username,
		Roles:       append([]string(nil), e.Roles...),
		Permissions: append([]string(nil), e.Permissions...),
		ExpiresAt:   e.ExpiresAt,
	}
}

func (c AuthContext) UserSummary() UserSummary {
	return UserSummary{
		ID:          c.UserID,
		Username:    c.Username,
		Roles:       append([]string(nil), c.Roles...),
		Permissions: append([]string(nil), c.Permissions...),
	}
}

func (c AuthContext) ApplyDownstreamHeaders(header http.Header, forwardedFor string, forwardedProto string) {
	header.Set("X-Request-Id", c.RequestID)
	header.Set("X-User-Id", c.UserID)
	header.Set("X-User-Roles", strings.Join(c.Roles, ","))
	header.Set("X-User-Permissions", strings.Join(c.Permissions, ","))
	if strings.TrimSpace(forwardedFor) != "" {
		header.Set("X-Forwarded-For", strings.TrimSpace(forwardedFor))
	}
	if strings.TrimSpace(forwardedProto) != "" {
		header.Set("X-Forwarded-Proto", strings.TrimSpace(forwardedProto))
	}
}

func normalizeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
