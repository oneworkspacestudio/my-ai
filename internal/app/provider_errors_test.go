package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProviderFailuresExplainLocalCausesWithoutLeakingResponse(t *testing.T) {
	for _, tt := range []struct {
		provider      string
		status        int
		message, want string
	}{
		{"ollama", 500, "error loading model: unknown tensor type: 42 PRIVATE_PROMPT secret-api-key", "Update Ollama"},
		{"ollama", 500, "model requires more system memory (8 GB) than is available PRIVATE_PROMPT", "available memory"},
		{"ollama", 404, "model 'PRIVATE_PROMPT' not found", "not installed"},
		{"ollama", 500, "chat prompt error: no user query found in messages PRIVATE_PROMPT", "too long"},
		{"ollama", 500, "runner exited PRIVATE_PROMPT secret-api-key", "could not load or run"},
		{"openai", 401, "PRIVATE_PROMPT secret-api-key", "API key"},
		{"deepseek", 500, "PRIVATE_PROMPT secret-api-key", "Try again shortly"},
	} {
		t.Run(tt.provider+tt.want, func(t *testing.T) {
			raw, _ := json.Marshal(map[string]string{"error": tt.message})
			e := providerFailure(tt.provider, tt.status, raw)
			got := e.Error()
			if !strings.Contains(got, tt.want) || strings.Contains(got, "PRIVATE_PROMPT") || strings.Contains(got, "secret-api-key") {
				t.Fatal(got)
			}
			if tt.provider == "ollama" && (strings.Contains(got, "quota") || strings.Contains(got, "credentials")) {
				t.Fatal("local failure mislabelled", got)
			}
		})
	}
	for _, raw := range []string{`<html>unknown tensor type secret-api-key</html>`, `{"error":{"message":"unknown tensor type secret-api-key"}}`, `"unknown tensor type secret-api-key"`} {
		if strings.Contains(providerFailure("ollama", 500, []byte(raw)).Error(), "secret-api-key") {
			t.Fatal("raw upstream response leaked")
		}
	}
}

func TestStreamedOllamaLoadingFailureExplainsRuntimeUpgrade(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{\"error\":\"error loading model: unknown tensor type: 42 secret-api-key\"}\n"))
	}))
	defer srv.Close()
	_, e := (&App{}).generate(context.Background(), Provider{ID: "ollama", Base: srv.URL}, "fixture", nil, toolScope{}, true, func(string, any) {})
	if e == nil || !strings.Contains(e.Error(), "Update Ollama") || strings.Contains(e.Error(), "secret-api-key") {
		t.Fatal(e)
	}
}

func TestOllamaToolsUnsupportedStillFallsBackOnlyInChat(t *testing.T) {
	for _, project := range []string{"", "selected"} {
		requests := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests++
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["tools"] != nil {
				w.WriteHeader(400)
				w.Write([]byte(`{"error":"model does not support tools"}`))
				return
			}
			w.Write([]byte("{\"message\":{\"content\":\"hello\"},\"done\":true}\n"))
		}))
		s := Session{Model: "fixture", Messages: []Message{{Role: "user", Content: "hello"}}}
		out, e := (&App{}).chatWithTools(context.Background(), Provider{ID: "ollama", Base: srv.URL}, &s, func(string, any) {}, toolScope{Project: project})
		srv.Close()
		if project == "" {
			if e != nil || out != "hello" || requests != 2 {
				t.Fatal(out, e, requests)
			}
		} else if e == nil || requests != 1 {
			t.Fatal("project tools were silently bypassed", e, requests)
		}
	}
}

func TestOllamaVersionStatusRejectsUntrustedText(t *testing.T) {
	for _, v := range []string{"0.40.11", "0.40.11-rc.1", "0.40.11+cpu"} {
		if !validOllamaVersion(v) {
			t.Fatal(v)
		}
	}
	for _, v := range []string{"", "0.40", "0.40.11 secret", "<script>", "0.40.11\n", strings.Repeat("1", 49) + ".1.1"} {
		if validOllamaVersion(v) {
			t.Fatal("Unsafe version accepted")
		}
	}
}
