package fileclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

func TestClientUploadsReadsAndDeletesFileObject(t *testing.T) {
	var sawToken, sawCaller bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawToken = r.Header.Get("X-Service-Token") == "token"
		sawCaller = r.Header.Get("X-Caller-Service") == "qa"
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/internal/v1/files":
			if err := r.ParseMultipartForm(1024); err != nil {
				t.Fatalf("parse multipart: %v", err)
			}
			file, header, err := r.FormFile("file")
			if err != nil {
				t.Fatalf("file part missing: %v", err)
			}
			defer file.Close()
			data, _ := io.ReadAll(file)
			if header.Filename != "manual.pdf" || string(data) != "payload" {
				t.Fatalf("unexpected upload: name=%q data=%q", header.Filename, data)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "file-1", "filename": "manual.pdf", "contentType": "application/pdf", "sizeBytes": 7}})
		case r.Method == http.MethodGet && r.URL.Path == "/internal/v1/files/file-1/content":
			_, _ = w.Write([]byte("payload"))
		case r.Method == http.MethodDelete && r.URL.Path == "/internal/v1/files/file-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := New(server.URL, "token", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	obj, err := client.Upload(context.Background(), service.FileUploadInput{Filename: "manual.pdf", ContentType: "application/pdf", SizeBytes: 7, Body: strings.NewReader("payload")})
	if err != nil || obj.ID != "file-1" {
		t.Fatalf("upload=%+v err=%v", obj, err)
	}
	data, err := client.Read(context.Background(), obj.ID)
	if err != nil || string(data) != "payload" {
		t.Fatalf("read=%q err=%v", data, err)
	}
	if err := client.Delete(context.Background(), obj.ID); err != nil {
		t.Fatal(err)
	}
	if !sawToken || !sawCaller {
		t.Fatalf("trusted headers not sent: token=%v caller=%v", sawToken, sawCaller)
	}
}

func TestClientSanitizesDownstreamErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `bucket=secret objectKey=hidden token=secret`, http.StatusInternalServerError)
	}))
	defer server.Close()
	client, err := New(server.URL, "token", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Read(context.Background(), "file-1")
	if err == nil {
		t.Fatal("expected dependency error")
	}
	if strings.Contains(err.Error(), "objectKey") || strings.Contains(err.Error(), "token=secret") {
		t.Fatalf("downstream details leaked: %v", err)
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
	if _, err := client.Read(context.Background(), "file-1"); err == nil {
		t.Fatal("expected redirect to be treated as dependency error")
	}
}
