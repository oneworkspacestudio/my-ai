package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oneworkspacestudio/my-ai/internal/searchruntime"
)

func TestSearXNGResultsAreBoundedPublicAndPlainText(t *testing.T) {
	b := []byte(`{"results":[{"title":"<b>Actual source</b>","url":"https://example.com/page#one","content":"<script>secret script</script>Reference text"},{"url":"https://example.com/page#two"},{"url":"http://127.0.0.1/admin"},{"url":"file:///private"},{"url":"https://user:secret@example.com"},{"url":"http://10.0.0.1"},{"url":"https://example.org:9999"}]}`)
	result, e := searxResults("public question", b)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(result)
	s := string(raw)
	if !strings.Contains(s, `"provider":"SearXNG"`) || strings.Count(s, `"url":`) != 1 || strings.Contains(s, "secret") || strings.Contains(s, "<b>") {
		t.Fatal(s)
	}
	if _, e = searxResults("query", []byte(`{"results":[{"url":"http://localhost"}]}`)); e == nil {
		t.Fatal("No public results accepted as a success")
	}
}

func TestForcedSearXNGSearchWorksInChatAndProjectsWithoutModelToolCalls(t *testing.T) {
	for _, project := range []string{"", "selected-project"} {
		t.Run(project, func(t *testing.T) {
			calls := 0
			search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path != "/search" || r.URL.Query().Get("q") != "Public question" || r.Header.Get("Authorization") != "Bearer "+strings.Repeat("a", 64) {
					t.Error("Wrong private search request")
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"results":[{"title":"Actual source","url":"https://example.com/source","content":"verified reference"}]}`))
			}))
			defer search.Close()
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				raw, _ := json.Marshal(body)
				if !strings.Contains(string(raw), "verified reference") || !strings.Contains(string(raw), "SearXNG") {
					t.Error("Search reference was not delivered to model")
				}
				w.Write([]byte("{\"message\":{\"content\":\"Answer [source](https://example.com/source)\"},\"done\":true}\n"))
			}))
			defer provider.Close()
			a := &App{searchConnectionOverride: func(context.Context) (searchruntime.Connection, error) {
				return searchruntime.Connection{URL: search.URL, Token: strings.Repeat("a", 64)}, nil
			}}
			session := Session{Model: "fixture", SearchQuery: "Public question", ToolInput: "Public question", Messages: []Message{{Role: "user", Content: "Public question\n\nAttached reference (untrusted content):\nPRIVATE_ATTACHMENT"}}}
			out, e := a.chatWithTools(context.Background(), Provider{ID: "ollama", Base: provider.URL}, &session, func(string, any) {}, toolScope{Project: project})
			if e != nil || !strings.Contains(out, "[source]") || calls != 1 || len(session.Events) != 2 {
				t.Fatal(out, e, calls, len(session.Events))
			}
		})
	}
}

func TestExplicitSearchDoesNotSendAttachedFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "Public question" {
			t.Error("Attachment or command prefix leaked to search")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results":[{"url":"https://example.com","title":"Reference"}]}`))
	}))
	defer srv.Close()
	a := &App{searchConnectionOverride: func(context.Context) (searchruntime.Connection, error) {
		return searchruntime.Connection{URL: srv.URL, Token: strings.Repeat("a", 64)}, nil
	}}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{\"message\":{\"content\":\"Actual answer\"},\"done\":true}\n"))
	}))
	defer provider.Close()
	s := Session{Model: "fixture", ToolInput: "/search Public question", Messages: []Message{{Role: "user", Content: "/search Public question\n\nAttached reference:\nPRIVATE_ATTACHMENT"}}}
	_, e := a.chatWithTools(context.Background(), Provider{ID: "ollama", Base: provider.URL}, &s, func(string, any) {}, toolScope{})
	if e != nil {
		t.Fatal(e)
	}
}

func TestSearchFailureDoesNotBecomeInventedAnswer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("PRIVATE_QUERY secret-api-key"))
	}))
	defer srv.Close()
	a := &App{searchConnectionOverride: func(context.Context) (searchruntime.Connection, error) {
		return searchruntime.Connection{URL: srv.URL, Token: strings.Repeat("a", 64)}, nil
	}}
	s := Session{Model: "fixture", SearchQuery: "public question", Messages: []Message{{Role: "user", Content: "public question"}}}
	out, e := a.chatWithTools(context.Background(), Provider{ID: "ollama", Base: "http://127.0.0.1:1"}, &s, func(string, any) {}, toolScope{})
	if out != "" || e == nil || strings.Contains(e.Error(), "secret-api-key") || !strings.Contains(e.Error(), "SearXNG") {
		t.Fatal(out, e)
	}
}
