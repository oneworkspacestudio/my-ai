package app

import (
	"context"
	"encoding/json"
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestChatProviderProtocols(t *testing.T) {
	for _, provider := range []string{"openai", "deepseek", "ollama"} {
		t.Run(provider, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				json.NewDecoder(r.Body).Decode(&body)
				if body["model"] != "fixture-model" || body["stream"] != true {
					t.Error("model/stream were not preserved")
				}
				if r.Header.Get("Authorization") != "Bearer fixture-key" {
					t.Error("server credential missing")
				}
				switch provider {
				case "openai":
					if r.URL.Path != "/responses" || body["store"] != false || body["input"] == nil {
						t.Error("Responses API contract")
					}
					w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\" world\"}\n\ndata: {\"type\":\"response.completed\"}\n\n"))
				case "deepseek":
					if r.URL.Path != "/chat/completions" || body["messages"] == nil {
						t.Error("Chat completions contract")
					}
					w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hello world\"}}]}\n\ndata: [DONE]\n\n"))
				case "ollama":
					if r.URL.Path != "/api/chat" {
						t.Error("Ollama contract")
					}
					w.Write([]byte("{\"message\":{\"content\":\"hello world\"},\"done\":true,\"eval_count\":2}\n"))
				}
			}))
			defer srv.Close()
			a := &App{}
			s := &Session{Model: "fixture-model", Messages: []Message{{Role: "user", Content: "question"}}}
			ctx, c := context.WithTimeout(context.Background(), 5*time.Second)
			defer c()
			var deltas string
			out, e := a.chat(ctx, Provider{ID: provider, Base: srv.URL, Key: "fixture-key"}, s, func(k string, v any) {
				if k == "delta" {
					deltas += v.(string)
				}
			})
			if e != nil || out != "hello world" || deltas != out {
				t.Fatal(out, deltas, e)
			}
		})
	}
}
func TestStreamErrorsAndCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		w.Write([]byte(`{"error":"fixture-secret-should-not-leak"}`))
	}))
	defer srv.Close()
	_, e := request(context.Background(), Provider{Base: srv.URL}, "GET", "/", nil)
	if e == nil || strings.Contains(e.Error(), "fixture-secret") {
		t.Fatal("raw upstream error leaked", e)
	}
	ctx, c := context.WithCancel(context.Background())
	c()
	_, e = request(ctx, Provider{Base: srv.URL}, "GET", "/", nil)
	if e == nil {
		t.Fatal("cancelled request succeeded")
	}
}
func TestVersionedSessionAndPaths(t *testing.T) {
	root := t.TempDir()
	a := &App{}
	s := Session{ID: apphost.ID(), Title: "original", Mode: "chat", Policy: "read-only", Provider: "ollama"}
	if e := a.write(root, &s); e != nil {
		t.Fatal(e)
	}
	s.Title = "changed"
	if e := a.write(root, &s); e != nil {
		t.Fatal(e)
	}
	saved, e := a.read(root, s.ID)
	if e != nil || saved.Revision != 2 || saved.Title != "changed" {
		t.Fatal(saved, e)
	}
	b, e := os.ReadFile(filepath.Join(root, "Sessions", s.ID, "previous.json"))
	if e != nil || !strings.Contains(string(b), "original") {
		t.Fatal("missing recovery revision", e)
	}
	if _, e = a.read(root, "../../other-user"); e == nil {
		t.Fatal("path traversal")
	}
	if _, e = safePath(root, "..", "escape"); e == nil {
		t.Fatal("safePath escape")
	}
	if validSession(Session{ID: s.ID, Title: "x", Mode: "code", Provider: "deepseek", Policy: "danger-full-access"}) {
		t.Fatal("full access accepted")
	}
}
func TestDPAPIAndEndpointValidation(t *testing.T) {
	raw := []byte(`{"key":"fixture-not-real"}`)
	encrypted, e := protect(raw)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(encrypted), "fixture-not-real") {
		t.Fatal("plaintext secret")
	}
	back, e := unprotect(encrypted)
	if e != nil || string(back) != string(raw) {
		t.Fatal("DPAPI roundtrip", e)
	}
	for _, p := range []Provider{{ID: "openai", Base: "http://127.0.0.1:9999"}, {ID: "ollama", Base: "http://169.254.169.254"}, {ID: "deepseek", Base: "https://api.deepseek.com@evil.test"}} {
		if allowedEndpoint(p) {
			t.Fatal("unsafe endpoint accepted", p)
		}
	}
}
func TestRealCodexHandshake(t *testing.T) {
	if os.Getenv("MYAI_TEST_CODEX") != "1" {
		t.Skip("explicit local Codex protocol test")
	}
	a, e := startAgent(t.TempDir(), t.TempDir(), "")
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	ctx, c := context.WithTimeout(context.Background(), 25*time.Second)
	defer c()
	if e = a.Initialize(ctx); e != nil {
		t.Fatal(e)
	}
	b, e := a.Call(ctx, "model/list", map[string]any{"limit": 100})
	if e != nil {
		t.Fatal(e)
	}
	var v struct {
		Data []any `json:"data"`
	}
	if json.Unmarshal(b, &v) != nil || len(v.Data) == 0 {
		t.Fatalf("invalid model/list %s", string(b))
	}
	t.Log("Official app-server handshake and model discovery passed; no provider turn or credentials used")
}

func TestTruncatedStreamIsInterrupted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"partial\"}\n\n"))
	}))
	defer srv.Close()
	a := &App{}
	out, e := a.chat(context.Background(), Provider{ID: "openai", Base: srv.URL}, &Session{Model: "fixture-model"}, func(string, any) {})
	if out != "partial" || e == nil {
		t.Fatal("Truncated output was incorrectly completed", out, e)
	}
}
