package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gatewayhttp "github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/http"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/service"
)

func TestHealthReturnsRequestID(t *testing.T) {
	server := newTestServer(t, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Request-Id", "req_health")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	if got := res.Header().Get("X-Request-Id"); got != "req_health" {
		t.Fatalf("X-Request-Id = %q", got)
	}
	var body successBody[map[string]string]
	decodeJSON(t, res.Body, &body)
	if body.RequestID != "req_health" || body.Data["status"] != "ok" {
		t.Fatalf("body = %+v", body)
	}
}

func TestCreateSessionCachesTokenHashWithoutRawToken(t *testing.T) {
	store := newFakeStore()
	server := newTestServer(t, nil, store)
	req := jsonRequest(http.MethodPost, "/api/v1/sessions", `{"username":"alice","password":"secret"}`)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body successBody[sessionResponseData]
	decodeJSON(t, res.Body, &body)
	if body.Data.Session.AccessToken != "tok_valid" {
		t.Fatalf("access token = %q", body.Data.Session.AccessToken)
	}
	if store.saved.AccessTokenHash == "" || store.saved.AccessTokenHash == "tok_valid" {
		t.Fatalf("saved token hash = %q", store.saved.AccessTokenHash)
	}
	if !strings.HasPrefix(store.saved.AccessTokenHash, "hmac-sha256:v1:") {
		t.Fatalf("saved token hash = %q", store.saved.AccessTokenHash)
	}
	raw, err := json.Marshal(store.saved)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if bytes.Contains(raw, []byte("tok_valid")) {
		t.Fatalf("session cache entry contains raw access token: %s", raw)
	}
	if store.ttl <= 0 {
		t.Fatalf("ttl = %s", store.ttl)
	}
}

func TestCurrentUserRequiresBearerToken(t *testing.T) {
	server := newTestServer(t, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("X-Request-Id", "req_no_token")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	assertError(t, res, http.StatusUnauthorized, "unauthorized")
}

func TestCurrentUserReturnsUnauthorizedOnSessionMiss(t *testing.T) {
	server := newTestServer(t, nil, newFakeStore())
	req := authorizedRequest(http.MethodGet, "/api/v1/users/me", nil, "tok_missing")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	assertError(t, res, http.StatusUnauthorized, "unauthorized")
}

func TestCurrentUserReturnsDependencyWhenSessionCacheUnavailable(t *testing.T) {
	store := newFakeStore()
	store.getErr = errors.New("redis unavailable")
	server := newTestServer(t, nil, store)
	req := authorizedRequest(http.MethodGet, "/api/v1/users/me", nil, "tok_valid")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	assertError(t, res, http.StatusBadGateway, "dependency_error")
}

func TestCurrentUserReturnsUnauthorizedOnExpiredSession(t *testing.T) {
	store := newFakeStore()
	hasher := testHasher(t)
	identity := defaultIdentity()
	identity.Session.ExpiresAt = time.Now().UTC().Add(time.Hour)
	tokenHash, err := hasher.Hash("tok_expired")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	entry, _, err := service.NewCacheEntry(identity, tokenHash, "req_seed", time.Now().UTC())
	if err != nil {
		t.Fatalf("NewCacheEntry() error = %v", err)
	}
	entry.ExpiresAt = time.Now().UTC().Add(-time.Minute)
	store.entries[tokenHash] = entry
	server := newTestServerWithHasher(t, nil, store, hasher)
	req := authorizedRequest(http.MethodGet, "/api/v1/users/me", nil, "tok_expired")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	assertError(t, res, http.StatusUnauthorized, "unauthorized")
}

func TestCurrentUserReadsSessionCache(t *testing.T) {
	store := newFakeStore()
	hasher := testHasher(t)
	cacheSession(t, store, hasher, defaultIdentity(), "tok_valid", "req_seed")
	server := newTestServerWithHasher(t, nil, store, hasher)
	req := authorizedRequest(http.MethodGet, "/api/v1/users/me", nil, "tok_valid")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body successBody[userSummaryResponse]
	decodeJSON(t, res.Body, &body)
	if body.Data.ID != "usr_123" || body.Data.Username != "alice" {
		t.Fatalf("user = %+v", body.Data)
	}
	if got := strings.Join(body.Data.Permissions, ","); got != "knowledge:read,document:upload" {
		t.Fatalf("permissions = %q", got)
	}
}

func TestDeleteCurrentSessionRevokesAuthAndDeletesCache(t *testing.T) {
	store := newFakeStore()
	hasher := testHasher(t)
	cacheSession(t, store, hasher, defaultIdentity(), "tok_valid", "req_seed")
	auth := &fakeAuth{identity: defaultIdentity()}
	server := newTestServerWithHasher(t, auth, store, hasher)
	req := authorizedRequest(http.MethodDelete, "/api/v1/sessions/current", nil, "tok_valid")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	if auth.deletedSessionID != "sess_123" {
		t.Fatalf("deleted session = %q", auth.deletedSessionID)
	}
	if store.deleted == "" {
		t.Fatalf("cache key was not deleted")
	}
}

func TestAuthForbiddenIsMapped(t *testing.T) {
	auth := &fakeAuth{createSessionErr: service.ForbiddenError("permission denied")}
	server := newTestServer(t, auth, nil)
	req := jsonRequest(http.MethodPost, "/api/v1/sessions", `{"username":"alice","password":"secret"}`)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	assertError(t, res, http.StatusForbidden, "forbidden")
}

func TestAuthContextAppliesDownstreamHeaders(t *testing.T) {
	ctx := service.AuthContext{
		RequestID:   "req_123",
		UserID:      "usr_123",
		Roles:       []string{"admin", "operator"},
		Permissions: []string{"knowledge:read", "document:upload"},
	}
	headers := http.Header{}

	ctx.ApplyDownstreamHeaders(headers, "203.0.113.10", "https")

	assertHeader(t, headers, "X-Request-Id", "req_123")
	assertHeader(t, headers, "X-User-Id", "usr_123")
	assertHeader(t, headers, "X-User-Roles", "admin,operator")
	assertHeader(t, headers, "X-User-Permissions", "knowledge:read,document:upload")
	assertHeader(t, headers, "X-Forwarded-For", "203.0.113.10")
	assertHeader(t, headers, "X-Forwarded-Proto", "https")
}

func newTestServer(t *testing.T, auth *fakeAuth, store *fakeSessionStore) http.Handler {
	t.Helper()
	return newTestServerWithHasher(t, auth, store, testHasher(t))
}

func newTestServerWithHasher(t *testing.T, auth *fakeAuth, store *fakeSessionStore, hasher service.TokenHasher) http.Handler {
	t.Helper()
	if auth == nil {
		auth = &fakeAuth{identity: defaultIdentity()}
	}
	if store == nil {
		store = newFakeStore()
	}
	server, err := gatewayhttp.NewServer(auth, store, hasher, gatewayhttp.Config{MaxRequestBytes: 1024 * 1024})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	return server
}

func testHasher(t *testing.T) service.TokenHasher {
	t.Helper()
	hasher, err := service.NewTokenHasher("test-secret", "v1")
	if err != nil {
		t.Fatalf("NewTokenHasher() error = %v", err)
	}
	return hasher
}

func defaultIdentity() service.SessionIdentity {
	return service.SessionIdentity{
		User: service.UserSummary{
			ID:          "usr_123",
			Username:    "alice",
			Roles:       []string{"admin"},
			Permissions: []string{"knowledge:read", "document:upload"},
		},
		Session: service.SessionSummary{
			SessionID:   "sess_123",
			AccessToken: "tok_valid",
			TokenType:   "Bearer",
			ExpiresAt:   time.Now().UTC().Add(time.Hour),
		},
	}
}

func cacheSession(t *testing.T, store *fakeSessionStore, hasher service.TokenHasher, identity service.SessionIdentity, token string, requestID string) {
	t.Helper()
	identity.Session.AccessToken = token
	tokenHash, err := hasher.Hash(token)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	entry, ttl, err := service.NewCacheEntry(identity, tokenHash, requestID, time.Now().UTC())
	if err != nil {
		t.Fatalf("NewCacheEntry() error = %v", err)
	}
	if err := store.Save(context.Background(), entry, ttl); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}

func jsonRequest(method string, target string, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "req_test")
	return req
}

func authorizedRequest(method string, target string, body io.Reader, token string) *http.Request {
	req := httptest.NewRequest(method, target, body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Request-Id", "req_test")
	return req
}

func assertError(t *testing.T, res *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if res.Code != status {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body errorResponseBody
	decodeJSON(t, res.Body, &body)
	if body.Error.Code != code {
		t.Fatalf("error code = %q", body.Error.Code)
	}
	if body.Error.RequestID == "" {
		t.Fatalf("missing request id: %+v", body.Error)
	}
}

func assertHeader(t *testing.T, headers http.Header, key string, value string) {
	t.Helper()
	if got := headers.Get(key); got != value {
		t.Fatalf("%s = %q, want %q", key, got, value)
	}
}

func decodeJSON(t *testing.T, reader io.Reader, target any) {
	t.Helper()
	if err := json.NewDecoder(reader).Decode(target); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
}

type fakeAuth struct {
	identity         service.SessionIdentity
	createUserErr    error
	createSessionErr error
	deleteErr        error
	deletedSessionID string
}

func (a *fakeAuth) CreateUser(context.Context, string, string, string) (service.SessionIdentity, error) {
	if a.createUserErr != nil {
		return service.SessionIdentity{}, a.createUserErr
	}
	return a.identity, nil
}

func (a *fakeAuth) CreateSession(context.Context, string, string, string) (service.SessionIdentity, error) {
	if a.createSessionErr != nil {
		return service.SessionIdentity{}, a.createSessionErr
	}
	return a.identity, nil
}

func (a *fakeAuth) DeleteSession(_ context.Context, _ string, sessionID string) error {
	a.deletedSessionID = sessionID
	return a.deleteErr
}

type fakeSessionStore struct {
	entries   map[string]service.GatewaySessionCacheEntry
	saved     service.GatewaySessionCacheEntry
	ttl       time.Duration
	deleted   string
	saveErr   error
	getErr    error
	deleteErr error
	pingErr   error
}

func newFakeStore() *fakeSessionStore {
	return &fakeSessionStore{entries: map[string]service.GatewaySessionCacheEntry{}}
}

func (s *fakeSessionStore) Save(_ context.Context, entry service.GatewaySessionCacheEntry, ttl time.Duration) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.saved = entry
	s.ttl = ttl
	s.entries[entry.AccessTokenHash] = entry
	return nil
}

func (s *fakeSessionStore) Get(_ context.Context, accessTokenHash string) (service.GatewaySessionCacheEntry, error) {
	if s.getErr != nil {
		return service.GatewaySessionCacheEntry{}, s.getErr
	}
	entry, ok := s.entries[accessTokenHash]
	if !ok {
		return service.GatewaySessionCacheEntry{}, service.ErrSessionNotFound
	}
	return entry, nil
}

func (s *fakeSessionStore) Delete(_ context.Context, accessTokenHash string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = accessTokenHash
	delete(s.entries, accessTokenHash)
	return nil
}

func (s *fakeSessionStore) Ping(context.Context) error {
	return s.pingErr
}

type successBody[T any] struct {
	Data      T      `json:"data"`
	RequestID string `json:"requestId"`
}

type userSummaryResponse struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

type sessionSummaryResponse struct {
	SessionID   string `json:"sessionId"`
	AccessToken string `json:"accessToken"`
	TokenType   string `json:"tokenType"`
	ExpiresAt   string `json:"expiresAt"`
}

type sessionResponseData struct {
	User    userSummaryResponse    `json:"user"`
	Session sessionSummaryResponse `json:"session"`
}

type errorResponseBody struct {
	Error struct {
		Code      string            `json:"code"`
		Message   string            `json:"message"`
		RequestID string            `json:"requestId"`
		Fields    map[string]string `json:"fields"`
	} `json:"error"`
}
