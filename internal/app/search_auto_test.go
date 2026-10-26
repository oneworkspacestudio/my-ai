package app

import (
	"context"
	"encoding/json"
	"github.com/oneworkspacestudio/my-ai/internal/searchruntime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
