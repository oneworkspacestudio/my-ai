package app

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Opt-in live acceptance: isolated retained public weights on port39735, real
// bundled SearXNG/pages, and no owner account, private Drive or cloud credentials.
func TestNativeResearch(t *testing.T) {
	if os.Getenv("MYAI_TEST_RESEARCH") != "1" {
		t.Skip("Explicit isolated live research acceptance")
	}
	var trace []map[string]string
	a := App{clock: func() time.Time { return time.Date(2026, 10, 26, 8, 20, 0, 0, time.UTC) }, researchTrace: func(phase, output string) {
		trace = append(trace, map[string]string{"phase": phase, "output": output})
		if path := os.Getenv("MYAI_TEST_RECEIPT"); path != "" {
			raw, _ := json.MarshalIndent(trace, "", "  ")
			_ = os.WriteFile(path+".trace.json", raw, 0600)
		}
	}}
	defer a.closeSearch()
	provider := Provider{ID: "ollama", Base: "http://127.0.0.1:39735"}
	var receipts []map[string]any
	var previousSession []byte
	for _, tc := range []struct {
		question string
		want     []string
	}{
		{"what was the Liverpool score", []string{"2", "0", "Brighton"}},
		{"What was the Liverpool result yesterday?", []string{"2", "0", "Brighton"}},
		{"Who won the 2014 World Cup final?", []string{"Germany"}},
		{"What is the latest stable Go release? Please check official release notes.", []string{"Go", "1.27.3"}},
		{"What about yesterday?", []string{"Brighton", "25", "2026"}},
	} {
		if selected := os.Getenv("MYAI_RESEARCH_CASE"); selected != "" && selected != tc.question {
			continue
		}
		t.Run(tc.question, func(t *testing.T) {
			trace = nil
			ctx, cancel := context.WithTimeout(context.Background(), 9*time.Minute)
			defer cancel()
			s := Session{Model: "eriktonon/bonsai-2-uncensored:4b", ToolInput: tc.question, SearchTimeZone: "Europe/London", Messages: []Message{{Role: "user", Content: tc.question, PublicQuestion: tc.question}}}
			if tc.question == "What about yesterday?" {
				if len(previousSession) == 0 {
					t.Fatal("Missing persisted public context")
				}
				if err := json.Unmarshal(previousSession, &s); err != nil {
					t.Fatal(err)
				}
				s.ToolInput = tc.question
				s.Events = nil
				s.Messages = append(s.Messages, Message{Role: "user", Content: tc.question, PublicQuestion: tc.question})
			}
			var displayed strings.Builder
			start := time.Now()
			answer, err := a.chatWithTools(ctx, provider, &s, func(k string, v any) {
				if k == "delta" {
					displayed.WriteString(v.(string))
				}
			}, toolScope{})
			if tc.question == "what was the Liverpool score" {
				s.Messages = append(s.Messages, Message{Role: "assistant", Content: answer})
				previousSession, _ = json.Marshal(s)
			}
			queries := []string{}
			readPages := 0
			for _, event := range s.Events {
				if event.Method != "tool/result" {
					continue
				}
				var v struct {
					Name   string         `json:"name"`
					Failed bool           `json:"failed"`
					Output searchResponse `json:"output"`
				}
				json.Unmarshal(event.Data, &v)
				if v.Name == "web_search" && !v.Failed && v.Output.Query != "" && len(v.Output.Sources) > 0 {
					queries = append(queries, v.Output.Query)
					for _, source := range v.Output.Sources {
						if source.Text != "" {
							readPages++
						}
					}
				}
			}
			receipts = append(receipts, map[string]any{"question": tc.question, "answer": answer, "error": func() string {
				if err != nil {
					return err.Error()
				}
				return ""
			}(), "queries": queries, "readPages": readPages, "elapsedSeconds": int(time.Since(start).Seconds()), "events": s.Events, "controlTrace": trace, "ownerFixtureUsed": false, "globalModelLibraryChanged": false})
			if path := os.Getenv("MYAI_TEST_RECEIPT"); path != "" {
				raw, _ := json.MarshalIndent(receipts, "", "  ")
				if e := os.WriteFile(path, raw, 0600); e != nil {
					t.Fatal(e)
				}
			}
			if err != nil || answer != displayed.String() || searchDenial.MatchString(answer) || len(queries) == 0 || !strings.Contains(answer, "https://") {
				t.Fatal(answer, err, queries)
			}
			visible := citationURL.ReplaceAllString(answer, "")
			if strings.Contains(tc.question, "2014") && (strings.Contains(visible, "Thomas") || strings.Contains(visible, "110")) {
				t.Errorf("Invented history: %s", answer)
			}
			if (strings.Contains(strings.ToLower(tc.question), "liverpool") || tc.question == "What about yesterday?") && !regexp.MustCompile(`(?i)(?:Liverpool\s+(?:beat |defeated |won |vs\.? |v\.? )?Brighton[^\n]{0,180}2\s*[-–:]\s*0|Liverpool[^\n]{0,180}2\s*[-–:]\s*0[^\n]{0,35}Brighton|2\s*[-–:]\s*0[^\n]{0,35}Brighton)`).MatchString(visible) {
				t.Errorf("No actual score-bearing answer: %s", answer)
			}
			for _, want := range tc.want {
				if !strings.Contains(strings.ToLower(visible), strings.ToLower(want)) {
					t.Errorf("Expected %q in supported answer: %s", want, answer)
				}
			}
			t.Logf("queries=%v, readPages=%d, answer=%s", queries, readPages, answer)
		})
	}
}
