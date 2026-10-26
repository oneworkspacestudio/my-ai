package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/oneworkspacestudio/my-ai/internal/searchruntime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestResearchRetriesEmptyResultsWithoutPublicationFilteringHistory(t *testing.T) {
	searches := 0
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		searches++
		if r.URL.Query().Get("time_range") != "" {
			t.Error("Historical event excluded by publication filter")
		}
		if r.URL.Query().Get("language") != "ja" {
			t.Error("Question language lost")
		}
		if searches == 1 {
			fmt.Fprint(w, `{"results":[],"unresponsive_engines":[["brave","too many requests"]]}`)
			return
		}
		fmt.Fprint(w, `{"results":[{"title":"Official report","url":"https://example.com/report","content":"Germany defeated Argentina 1-0 in the 2014 World Cup final."}]}`)
	}))
	defer search.Close()
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := modelBody(r)
		switch {
		case strings.Contains(raw, "research controller / plan"):
			modelText(w, "ollama", `{"search":true,"queries":["2014 World Cup final"],"language":"ja","time_range":"day"}`)
		case strings.Contains(raw, "research controller / evaluate"):
			if searches == 1 {
				modelText(w, "ollama", `{"sufficient":false,"queries":["2014 World Cup final official match report"]}`)
			} else {
				modelText(w, "ollama", `{"sufficient":true,"facts":[{"claim":"Germany won 1-0","url":"https://example.com/report","quote":"Germany defeated Argentina 1-0 in the 2014 World Cup final."}]}`)
			}
		case strings.Contains(raw, "research controller / verify"):
			modelText(w, "ollama", `{"supported":true}`)
		default:
			modelText(w, "ollama", "Germany beat Argentina 1-0 in the 2014 final. [Report](https://example.com/report)")
		}
	}))
	defer model.Close()
	a := App{webReadOverride: snippetOnlyRead, searchConnectionOverride: func(context.Context) (searchruntime.Connection, error) {
		return searchruntime.Connection{URL: search.URL, Token: strings.Repeat("a", 64)}, nil
	}}
	s := Session{Model: "fixture", ToolInput: "Who won the 2014 World Cup final?", Messages: []Message{{Role: "user", Content: "Who won the 2014 World Cup final?"}}}
	out, err := a.chatWithTools(context.Background(), Provider{ID: "ollama", Base: model.URL}, &s, func(string, any) {}, toolScope{})
	if err != nil || !strings.Contains(out, "Germany") || searches != 2 {
		t.Fatal(out, err, searches)
	}
}

func TestUnsupportedFinalClaimsAreNeverStreamedEvenAfterBadRepair(t *testing.T) {
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"results":[{"title":"Release notes","url":"https://example.com/releases","content":"Comet 8.2 is the stable release."}]}`)
	}))
	defer search.Close()
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := modelBody(r)
		switch {
		case strings.Contains(raw, "research controller / plan"):
			modelText(w, "ollama", `{"search":true,"queries":["Comet stable release"]}`)
		case strings.Contains(raw, "research controller / evaluate"):
			modelText(w, "ollama", `{"sufficient":true,"facts":[{"claim":"Comet 8.2 is stable","url":"https://example.com/releases","quote":"Comet 8.2 is the stable release."}]}`)
		case strings.Contains(raw, "research controller / verify"):
			modelText(w, "ollama", `{"supported":false,"problem":"99 is not in the source"}`)
		default:
			modelText(w, "ollama", "Comet 99 is the latest version. [Invented link](https://invented.example/releases)")
		}
	}))
	defer model.Close()
	a := App{webReadOverride: snippetOnlyRead, searchConnectionOverride: func(context.Context) (searchruntime.Connection, error) {
		return searchruntime.Connection{URL: search.URL, Token: strings.Repeat("a", 64)}, nil
	}}
	s := Session{Model: "fixture", ToolInput: "Latest stable Comet release?", Messages: []Message{{Role: "user", Content: "Latest stable Comet release?"}}}
	var displayed strings.Builder
	out, err := a.chatWithTools(context.Background(), Provider{ID: "ollama", Base: model.URL}, &s, func(k string, v any) {
		if k == "delta" {
			displayed.WriteString(v.(string))
		}
	}, toolScope{})
	if err != nil || out != displayed.String() || strings.Contains(out, "99") || strings.Contains(out, "invented.example") || !strings.Contains(out, "8.2") {
		t.Fatal(out, err)
	}
}

func TestMalformedControllerHasBoundedResearchAndHonestFallback(t *testing.T) {
	searches := 0
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		searches++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"results":[{"title":"Actual reference","url":"https://example.com/source","content":"Only a general reference; no requested fact is verified."}]}`)
	}))
	defer search.Close()
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		modelText(w, "ollama", "An invented answer 999 with no evidence.")
	}))
	defer model.Close()
	a := App{webReadOverride: snippetOnlyRead, searchConnectionOverride: func(context.Context) (searchruntime.Connection, error) {
		return searchruntime.Connection{URL: search.URL, Token: strings.Repeat("a", 64)}, nil
	}}
	s := Session{Model: "fixture", ToolInput: "Latest public news?", Messages: []Message{{Role: "user", Content: "Latest public news?"}}}
	out, err := a.chatWithTools(context.Background(), Provider{ID: "ollama", Base: model.URL}, &s, func(string, any) {}, toolScope{})
	if err != nil || searches < 2 || searches > 6 || strings.Contains(out, "999") || !strings.Contains(out, "could not verify") || !strings.Contains(out, "https://example.com/source") {
		t.Fatal(out, err, searches)
	}
}

func TestMissingLocationIsClarifiedWithoutInventedSearch(t *testing.T) {
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(modelBody(r), "research controller / plan") {
			t.Error("Unexpected answer generation")
		}
		modelText(w, "ollama", `{"search":false,"clarification":"Which city should I check the weather for?"}`)
	}))
	defer model.Close()
	a := App{searchConnectionOverride: func(context.Context) (searchruntime.Connection, error) {
		t.Fatal("Search before location clarified")
		return searchruntime.Connection{}, nil
	}}
	s := Session{Model: "fixture", ToolInput: "Weather near me?", Messages: []Message{{Role: "user", Content: "Weather near me?"}}}
	out, err := a.chatWithTools(context.Background(), Provider{ID: "ollama", Base: model.URL}, &s, func(string, any) {}, toolScope{})
	if err != nil || out != "Which city should I check the weather for?" || s.ResearchContext == nil {
		t.Fatal(out, err)
	}
}

func TestNaturalURLReadDoesNotRequireIndexingOrToolCalls(t *testing.T) {
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := modelBody(r)
		switch {
		case strings.Contains(raw, "research controller / plan"):
			modelText(w, "ollama", `{"search":true,"queries":[]}`)
		case strings.Contains(raw, "research controller / evaluate"):
			modelText(w, "ollama", `{"sufficient":true,"facts":[{"claim":"Comet 8.2 is stable","url":"https://example.com/article","quote":"Comet 8.2 is the stable release."}]}`)
		case strings.Contains(raw, "research controller / verify"):
			modelText(w, "ollama", `{"supported":true}`)
		default:
			modelText(w, "ollama", "The article announces stable Comet 8.2. [Article](https://example.com/article)")
		}
	}))
	defer model.Close()
	a := App{searchConnectionOverride: func(context.Context) (searchruntime.Connection, error) {
		t.Fatal("Indexed search was unnecessary for supplied page")
		return searchruntime.Connection{}, nil
	}, webReadOverride: func(ctx context.Context, address string) ([]byte, string, string, error) {
		if address != "https://example.com/article" {
			t.Fatal(address)
		}
		return []byte(`<main>Comet 8.2 is the stable release.</main>`), "text/html", address, nil
	}}
	s := Session{Model: "fixture", ToolInput: "Summarize https://example.com/article", Messages: []Message{{Role: "user", Content: "Summarize https://example.com/article"}}}
	out, err := a.chatWithTools(context.Background(), Provider{ID: "ollama", Base: model.URL}, &s, func(string, any) {}, toolScope{})
	if err != nil || !strings.Contains(out, "8.2") {
		t.Fatal(out, err)
	}
}

func TestPublicResearchMemoryIsBoundedAndSurvivesReopening(t *testing.T) {
	a := App{clock: func() time.Time { return time.Date(2026, 10, 26, 0, 30, 0, 0, time.UTC) }}
	s := Session{SearchTimeZone: "Europe/London"}
	a.rememberPublicResearch(&s, researchBundle{Question: "What was the Atlas score?", ResolvedQuestion: "Latest Atlas match result", Facts: []researchFact{{Claim: "Atlas won 2-0 on 25 October", URL: "https://example.com/report", Quote: "Atlas defeated Beacon 2-0."}}})
	raw, _ := json.Marshal(s)
	var restored Session
	json.Unmarshal(raw, &restored)
	if restored.ResearchContext == nil || len(restored.ResearchContext.Facts) != 1 || restored.ResearchContext.At == "" || restored.ResearchContext.TimeZone != "Europe/London" {
		t.Fatal(string(raw))
	}
}
