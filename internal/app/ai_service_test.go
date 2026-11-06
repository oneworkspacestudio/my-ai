package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestManagedOllamaUsesSharedServiceAndDelegatedUser(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer fixture-token" || r.Header.Get("X-AI-App") != "myai" || r.Header.Get("X-AI-User") != "alice" {
			t.Error("missing app identity")
		}
		if r.URL.Path != "/_internal/chat" && r.URL.Path != "/_internal/models" && r.URL.Path != "/_internal/show" && r.URL.Path != "/_internal/ps" && r.URL.Path != "/_internal/version" {
			t.Error("unexpected service endpoint", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"models":[],"done":true}`)
	}))
	defer server.Close()
	t.Setenv("MYAI_SERVICE_URL", server.URL)
	t.Setenv("MYAI_SERVICE_TOKEN", "fixture-token")
	p := Provider{ID: "ollama", Base: "http://127.0.0.1:1", User: "alice"}
	for _, path := range []string{"/api/chat", "/api/tags", "/api/show", "/api/ps", "/api/version"} {
		resp, e := request(context.Background(), p, "POST", path, map[string]any{"model": "fixture:9b"})
		if e != nil {
			t.Fatal(e)
		}
		resp.Body.Close()
	}
	if calls != 5 {
		t.Fatal(calls)
	}
	t.Setenv("MYAI_SERVICE_URL", "https://example.invalid")
	if _, e := request(context.Background(), p, "POST", "/api/chat", nil); e == nil {
		t.Fatal("non-loopback service accepted")
	}
}
