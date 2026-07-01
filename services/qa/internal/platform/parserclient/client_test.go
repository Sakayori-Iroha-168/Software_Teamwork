package parserclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

func TestClientParsesDocumentBytes(t *testing.T) {
	var sawPayload, sawHeaders bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/internal/v1/parsed-documents" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		sawHeaders = r.Header.Get("X-Service-Token") == "token" && r.Header.Get("X-Caller-Service") == "qa"
		var payload struct {
			DocumentName string `json:"documentName"`
			ContentType  string `json:"contentType"`
			DataBase64   string `json:"dataBase64"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		data, err := base64.StdEncoding.DecodeString(payload.DataBase64)
		if err != nil {
			t.Fatal(err)
		}
		sawPayload = payload.DocumentName == "manual.pdf" && payload.ContentType == "application/pdf" && string(data) == "payload"
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"content": "parsed text", "backend": "fake", "pages": []map[string]any{{"pageNumber": 1, "content": "parsed text"}}}})
	}))
	defer server.Close()
	client, err := New(server.URL, "token", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := client.Parse(context.Background(), service.ParseDocumentInput{DocumentName: "manual.pdf", ContentType: "application/pdf", SizeBytes: 7, Data: []byte("payload")})
	if err != nil || parsed.Content != "parsed text" || len(parsed.Pages) != 1 {
		t.Fatalf("parsed=%+v err=%v", parsed, err)
	}
	if !sawPayload || !sawHeaders {
		t.Fatalf("request not normalized: payload=%v headers=%v", sawPayload, sawHeaders)
	}
}

func TestClientSanitizesParserFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `internalUrl=http://parser.local token=secret`, http.StatusBadGateway)
	}))
	defer server.Close()
	client, err := New(server.URL, "token", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Parse(context.Background(), service.ParseDocumentInput{DocumentName: "manual.pdf", ContentType: "application/pdf", SizeBytes: 7, Data: []byte("payload")})
	if err == nil {
		t.Fatal("expected dependency error")
	}
	if strings.Contains(err.Error(), "parser.local") || strings.Contains(err.Error(), "token=secret") {
		t.Fatalf("parser failure leaked raw body: %v", err)
	}
}

func TestClientDoesNotFollowRedirectWithServiceToken(t *testing.T) {
	redirectTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token := r.Header.Get("X-Service-Token"); token != "" {
			t.Fatalf("service token leaked across redirect: %q", token)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer redirectTarget.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectTarget.URL+"/leak", http.StatusFound)
	}))
	defer server.Close()

	client, err := New(server.URL, "token", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Parse(context.Background(), service.ParseDocumentInput{DocumentName: "manual.pdf", ContentType: "application/pdf", SizeBytes: 7, Data: []byte("payload")})
	if err == nil {
		t.Fatal("expected redirect to be treated as dependency error")
	}
}
