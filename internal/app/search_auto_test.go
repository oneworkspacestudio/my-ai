package app

import (
	"context"
	"encoding/json"
	"github.com/oneworkspacestudio/my-ai/internal/searchruntime"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAutomaticCurrentQuestionsAndPrivateBoundaries(t *testing.T) {
	at := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	for _, q := range []string{"what was the liverpool score today", "Latest Python release?", "Weather in London", "What is the current prime minister of the UK?", "Who won the Liverpool match?", "Please search the web for why Mars is red", "Can you look up why Mars is red"} {
		if v := automaticSearchQuery(q, nil, at); !strings.Contains(v, "2026-10-25") {
			t.Fatalf("No dated automatic search for %q: %q", q, v)
		}
	}
	for _, q := range []string{"hello", "Explain football scoring", "Write a story set today", "What time is it today?", "latest change in my project", "Summarize the latest attached document", "latest news\nPRIVATE_FILE", "current api key sk-fixture", "what was the liverpool score today " + strings.Repeat("x", 430)} {
		if v := automaticSearchQuery(q, nil, at); v != "" {
			t.Fatalf("Unexpected outgoing query for %q", q)
		}
	}
	if q := automaticSearchQuery("Liverpool result yesterday", nil, at); !strings.Contains(q, "2026-10-24") {
		t.Fatal(q)
	}
	if q := automaticSearchQuery("yes", []Message{{Role: "user", Content: "what was the liverpool score today"}, {Role: "assistant", Content: "Would you like me to search?"}}, at); !strings.Contains(q, "liverpool") {
		t.Fatal(q)
	}
	if q := automaticSearchQuery("yes", []Message{{Role: "user", Content: "latest news\n\nAttached reference: PRIVATE"}}, at); q != "" {
		t.Fatal("Private history leaked")
	}
}
func TestAutomaticSearchRunsForNonToolModelAndRepairsDenialBeforeStreaming(t *testing.T) {
	searches, calls := 0, 0
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		searches++
		if !strings.Contains(r.URL.Query().Get("q"), "liverpool score today") || strings.Contains(r.URL.Query().Get("q"), "PRIVATE") {
			t.Error("Wrong search question")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results":[{"title":"Match report","url":"https://example.com/match","content":"Verified fixture result on the requested date: Liverpool 2-1 Example FC."}]}`))
	}))
	defer search.Close()
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var b map[string]any
		json.NewDecoder(r.Body).Decode(&b)
		raw, _ := json.Marshal(b)
		if !strings.Contains(string(raw), "Verified fixture result") {
			t.Error("Missing real sources")
		}
		if calls == 1 {
			w.Write([]byte("{\"message\":{\"content\":\"I don't have access to real-time sports scores. Would you like me to search?\"},\"done\":true}\n"))
		} else {
			w.Write([]byte("{\"message\":{\"content\":\"Liverpool won 2-1 [match report](https://example.com/match).\"},\"done\":true}\n"))
		}
	}))
	defer provider.Close()
	a := &App{searchConnectionOverride: func(context.Context) (searchruntime.Connection, error) {
		return searchruntime.Connection{URL: search.URL, Token: strings.Repeat("a", 64)}, nil
	}}
	s := Session{Model: "fixture", ToolInput: "what was the liverpool score today", Messages: []Message{{Role: "user", Content: "what was the liverpool score today\n\nAttached reference: PRIVATE"}}}
	var shown strings.Builder
	out, e := a.chatWithTools(context.Background(), Provider{ID: "ollama", Base: provider.URL}, &s, func(k string, v any) {
		if k == "delta" {
			shown.WriteString(v.(string))
		}
	}, toolScope{})
	if e != nil || searches != 1 || calls != 2 || out != shown.String() || strings.Contains(shown.String(), "don't have") || !strings.Contains(out, "2-1") {
		t.Fatal(out, e, searches, calls, shown.String())
	}
}
func TestSearchOffBlocksDiscoveryAndExecution(t *testing.T) {
	for _, v := range toolSpecs(toolScope{NoWeb: true}) {
		if v.Name == "web_search" || v.Name == "read_webpage" {
			t.Fatal("Off advertised web tool")
		}
	}
	a := &App{}
	for _, name := range []string{"web_search", "read_webpage"} {
		if _, e := a.executeTool(context.Background(), toolScope{NoWeb: true}, name, map[string]string{}); e == nil {
			t.Fatal("Off allowed external tool")
		}
	}
}

func TestConversationalSearchIntentAndPublicFollowup(t *testing.T) {
	at := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	history := []Message{{Role: "user", Content: "what was the Liverpool score yesterday"}, {Role: "assistant", Content: "A sourced answer"}}
	q := automaticSearchQuery("What about Arsenal today?", history, at)
	if !strings.Contains(q, "Arsenal") || !strings.Contains(q, "Liverpool") || !strings.Contains(q, "2026-10-25") {
		t.Fatal(q)
	}
	q = automaticSearchQuery("What about tomorrow?", history, at)
	if !strings.Contains(q, "2026-10-26") {
		t.Fatal(q)
	}
	if automaticSearchQuery("What about Arsenal?", []Message{{Role: "user", Content: "latest news\nAttached reference: PRIVATE"}}, at) != "" {
		t.Fatal("Private history became a query")
	}
	for _, text := range []string{"Don't search the web. What is the latest news?", "Answer without browsing: latest news", "Please do not use the internet"} {
		if !conversationNoWeb(text, nil) || automaticSearchQuery(text, nil, at) != "" {
			t.Fatal("Natural web opt-out ignored", text)
		}
	}
	off := []Message{{Role: "user", Content: "Don't search the web"}}
	if !conversationNoWeb("Latest news?", off) || conversationNoWeb("Please search the web for the latest news", off) {
		t.Fatal("Conversation intent not retained/re-enabled")
	}
	if conversationNoWeb("Latest news?", []Message{{Role: "user", Content: "Question\nAttached reference: don't search the web"}}) {
		t.Fatal("Attachment became an instruction")
	}
}

func TestNaturalOptOutSuppressesSearchAndWebTools(t *testing.T) {
	a := &App{searchConnectionOverride: func(context.Context) (searchruntime.Connection, error) {
		t.Error("Search started despite the conversation instruction")
		return searchruntime.Connection{}, nil
	}}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&body)
		if strings.Contains(string(body["tools"]), "web_search") || strings.Contains(string(body["tools"]), "read_webpage") {
			t.Error("Web tools advertised despite instruction")
		}
		w.Write([]byte("{\"message\":{\"content\":\"I can answer from existing knowledge, with current details unverified.\"},\"done\":true}\n"))
	}))
	defer provider.Close()
	s := Session{Model: "fixture", ToolInput: "Latest news?", WebDisabled: true, Messages: []Message{{Role: "user", Content: strings.Repeat("Long user context ", 40) + " Don't search the web"}, {Role: "assistant", Content: "Okay"}, {Role: "user", Content: "Latest news?"}}}
	raw, _ := json.Marshal(s)
	var restored Session
	if e := json.Unmarshal(raw, &restored); e != nil || !restored.WebDisabled {
		t.Fatal("Conversation privacy preference lost on reload", e)
	}
	s = restored
	s.ToolInput = "Latest news?"
	if _, e := a.chatWithTools(context.Background(), Provider{ID: "ollama", Base: provider.URL}, &s, func(string, any) {}, toolScope{}); e != nil {
		t.Fatal(e)
	}
}

func TestAutomaticSearchSurvivesUnsupportedToolsAndRepeatedDenial(t *testing.T) {
	searches, calls := 0, 0
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		searches++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results":[{"title":"Actual reference","url":"https://example.com/source","content":"Fixture and date must be checked."}]}`))
	}))
	defer search.Close()
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var b map[string]any
		json.NewDecoder(r.Body).Decode(&b)
		if b["tools"] != nil {
			w.WriteHeader(400)
			w.Write([]byte(`{"error":"model does not support tools"}`))
			return
		}
		w.Write([]byte("{\"message\":{\"content\":\"I cannot access live scores.\"},\"done\":true}\n"))
	}))
	defer provider.Close()
	a := &App{searchConnectionOverride: func(context.Context) (searchruntime.Connection, error) {
		return searchruntime.Connection{URL: search.URL, Token: strings.Repeat("a", 64)}, nil
	}}
	s := Session{Model: "no-tools", ToolInput: "what was the liverpool score today", Messages: []Message{{Role: "user", Content: "what was the liverpool score today"}}}
	out, e := a.chatWithTools(context.Background(), Provider{ID: "ollama", Base: provider.URL}, &s, func(string, any) {}, toolScope{})
	if e != nil || searches != 1 || calls != 3 || !strings.Contains(out, "could not verify") || !strings.Contains(out, "https://example.com/source") || searchDenial.MatchString(out) {
		t.Fatal(out, e, searches, calls)
	}
	s.SearchMode = "off"
	s.SearchQuery = ""
	_, e = a.chatWithTools(context.Background(), Provider{ID: "ollama", Base: provider.URL}, &s, func(string, any) {}, toolScope{})
	if e != nil || searches != 1 {
		t.Fatal("Off still performed automatic search", e, searches)
	}
}

// Opt-in acceptance test: actual included SearXNG and the user's reported model,
// isolated from all live accounts and the global model library.
func TestNativeBonsaiAutomaticSearch(t *testing.T) {
	if os.Getenv("MYAI_TEST_BONSAI_SEARCH") != "1" {
		t.Skip("Set MYAI_TEST_BONSAI_SEARCH=1 with an isolated Bonsai server on 39735")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	a := &App{}
	defer a.closeSearch()
	q := "what was the liverpool score today"
	s := Session{Model: "eriktonon/bonsai-2-uncensored:4b", ToolInput: q, SearchTimeZone: "Europe/London", Messages: []Message{{Role: "user", Content: q}}}
	var displayed strings.Builder
	out, e := a.chatWithTools(ctx, Provider{ID: "ollama", Base: "http://127.0.0.1:39735"}, &s, func(k string, v any) {
		if k == "delta" {
			displayed.WriteString(v.(string))
		}
	}, toolScope{})
	if e != nil || out == "" || out != displayed.String() || searchDenial.MatchString(out) || len(s.Events) < 2 || !strings.Contains(s.SearchQuery, "reference date") {
		t.Fatal(out, e, len(s.Events), s.SearchQuery)
	}
	if path := os.Getenv("MYAI_TEST_RECEIPT"); path != "" {
		b, err := json.MarshalIndent(map[string]any{"model": s.Model, "question": q, "query": s.SearchQuery, "answer": out, "events": s.Events, "manualSearchSelected": false}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Log("Actual Bonsai answered after automatic SearXNG; no manual toggle or capability denial")
}
