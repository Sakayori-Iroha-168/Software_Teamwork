package httpapi_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpapi "github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/http"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/repository"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

func TestCreateListGetPatchDeleteSession(t *testing.T) {
	server, _ := newHTTPTestServer()

	createReq := authorizedRequest(http.MethodPost, "/api/v1/qa-sessions", strings.NewReader(`{"title":"inspection"}`))
	createRes := httptest.NewRecorder()
	server.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", createRes.Code, createRes.Body.String())
	}
	var createBody sessionBody
	decodeJSON(t, createRes.Body, &createBody)
	if createBody.RequestID != "req_test" || createBody.Data.ID == "" || createBody.Data.Status != service.SessionStatusActive {
		t.Fatalf("create body = %+v", createBody)
	}

	listReq := authorizedRequest(http.MethodGet, "/api/v1/qa-sessions", nil)
	listRes := httptest.NewRecorder()
	server.ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRes.Code, listRes.Body.String())
	}
	var listBody sessionListBody
	decodeJSON(t, listRes.Body, &listBody)
	if listBody.Page.Total != 1 || len(listBody.Data) != 1 || listBody.Data[0].ID != createBody.Data.ID {
		t.Fatalf("list body = %+v", listBody)
	}

	getReq := authorizedRequest(http.MethodGet, "/api/v1/qa-sessions/"+createBody.Data.ID, nil)
	getRes := httptest.NewRecorder()
	server.ServeHTTP(getRes, getReq)
	if getRes.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getRes.Code, getRes.Body.String())
	}

	patchReq := authorizedRequest(http.MethodPatch, "/api/v1/qa-sessions/"+createBody.Data.ID, strings.NewReader(`{"title":"renamed","status":"archived"}`))
	patchRes := httptest.NewRecorder()
	server.ServeHTTP(patchRes, patchReq)
	if patchRes.Code != http.StatusOK {
		t.Fatalf("patch status = %d, body = %s", patchRes.Code, patchRes.Body.String())
	}
	var patchBody sessionBody
	decodeJSON(t, patchRes.Body, &patchBody)
	if patchBody.Data.Title != "renamed" || patchBody.Data.Status != service.SessionStatusArchived {
		t.Fatalf("patch body = %+v", patchBody)
	}

	deleteReq := authorizedRequest(http.MethodDelete, "/api/v1/qa-sessions/"+createBody.Data.ID, nil)
	deleteRes := httptest.NewRecorder()
	server.ServeHTTP(deleteRes, deleteReq)
	if deleteRes.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", deleteRes.Code, deleteRes.Body.String())
	}

	getDeletedReq := authorizedRequest(http.MethodGet, "/api/v1/qa-sessions/"+createBody.Data.ID, nil)
	getDeletedRes := httptest.NewRecorder()
	server.ServeHTTP(getDeletedRes, getDeletedReq)
	if getDeletedRes.Code != http.StatusNotFound {
		t.Fatalf("get deleted status = %d", getDeletedRes.Code)
	}
}

func TestCreateSessionAllowsEmptyBody(t *testing.T) {
	server, _ := newHTTPTestServer()

	req := authorizedRequest(http.MethodPost, "/api/v1/qa-sessions", nil)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("create empty body status = %d, body = %s", res.Code, res.Body.String())
	}

	var body sessionBody
	decodeJSON(t, res.Body, &body)
	if body.Data.ID == "" || body.Data.Status != service.SessionStatusActive {
		t.Fatalf("create empty body response = %+v", body)
	}
}

func TestListSessionsAggregatesMessagePreviewAndFiltersOwner(t *testing.T) {
	server, repo := newHTTPTestServer()
	now := time.Date(2026, 6, 28, 9, 0, 0, 0, time.UTC)
	repo.SeedSession(repository.Session{ID: "s1", ExternalUserID: "usr_123", Title: "mine", Status: service.SessionStatusActive, CreatedAt: now, UpdatedAt: now.Add(time.Hour)})
	repo.SeedSession(repository.Session{ID: "s2", ExternalUserID: "other", Title: "other", Status: service.SessionStatusActive, CreatedAt: now, UpdatedAt: now.Add(2 * time.Hour)})
	repo.SeedMessages(repository.Message{ID: "m1", ConversationID: "s1", SequenceNo: 1, Content: "preview", CreatedAt: now})

	req := authorizedRequest(http.MethodGet, "/api/v1/qa-sessions?page=1&pageSize=10", nil)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}

	var body sessionListBody
	decodeJSON(t, res.Body, &body)
	if body.Page.Total != 1 || body.Data[0].ID != "s1" {
		t.Fatalf("list body = %+v", body)
	}
	if body.Data[0].MessageCount != 1 || body.Data[0].LastMessagePreview != "preview" {
		t.Fatalf("session aggregate = %+v", body.Data[0])
	}
}

func TestSessionRoutesReturnAuthValidationAndForbiddenErrors(t *testing.T) {
	server, repo := newHTTPTestServer()
	now := time.Date(2026, 6, 28, 9, 0, 0, 0, time.UTC)
	repo.SeedSession(repository.Session{ID: "s1", ExternalUserID: "owner", Status: service.SessionStatusActive, CreatedAt: now, UpdatedAt: now})

	noUserReq := httptest.NewRequest(http.MethodGet, "/api/v1/qa-sessions", nil)
	noUserReq.Header.Set("X-Request-Id", "req_no_user")
	noUserRes := httptest.NewRecorder()
	server.ServeHTTP(noUserRes, noUserReq)
	if noUserRes.Code != http.StatusUnauthorized {
		t.Fatalf("missing auth status = %d", noUserRes.Code)
	}

	badStatusReq := authorizedRequest(http.MethodGet, "/api/v1/qa-sessions?status=deleted", nil)
	badStatusRes := httptest.NewRecorder()
	server.ServeHTTP(badStatusRes, badStatusReq)
	if badStatusRes.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d", badStatusRes.Code)
	}
	var badStatusBody errorBody
	decodeJSON(t, badStatusRes.Body, &badStatusBody)
	if badStatusBody.Error.Code != "validation_error" || badStatusBody.Error.Fields["status"] == "" {
		t.Fatalf("bad status body = %+v", badStatusBody)
	}

	forbiddenReq := authorizedRequest(http.MethodGet, "/api/v1/qa-sessions/s1", nil)
	forbiddenRes := httptest.NewRecorder()
	server.ServeHTTP(forbiddenRes, forbiddenReq)
	if forbiddenRes.Code != http.StatusForbidden {
		t.Fatalf("forbidden status = %d, body = %s", forbiddenRes.Code, forbiddenRes.Body.String())
	}
}

func newHTTPTestServer() (http.Handler, *repository.MemoryRepository) {
	repo := repository.NewMemoryRepository()
	sessions := service.New(repo)
	return httpapi.NewServer(sessions, httpapi.Config{}), repo
}

func authorizedRequest(method string, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	req.Header.Set("X-Request-Id", "req_test")
	req.Header.Set("X-User-Id", "usr_123")
	req.Header.Set("Content-Type", "application/json")
	return req
}

func decodeJSON(t *testing.T, reader io.Reader, target any) {
	t.Helper()
	if err := json.NewDecoder(reader).Decode(target); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
}

type sessionBody struct {
	Data      service.QASession `json:"data"`
	RequestID string            `json:"requestId"`
}

type sessionListBody struct {
	Data      []service.QASession `json:"data"`
	Page      service.Page        `json:"page"`
	RequestID string              `json:"requestId"`
}

type errorBody struct {
	Error struct {
		Code      string            `json:"code"`
		Message   string            `json:"message"`
		RequestID string            `json:"requestId"`
		Fields    map[string]string `json:"fields"`
	} `json:"error"`
}
