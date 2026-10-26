package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	syncatomic "sync/atomic"
	"testing"
	"time"

	"github.com/oneworkspacestudio/my-ai/internal/searchruntime"
)

func modelText(w http.ResponseWriter, provider, text string) {
	if provider == "ollama" {
		json.NewEncoder(w).Encode(map[string]any{"message": map[string]string{"content": text}, "done": true})
		return
	}
	if provider == "openai" {
		b, _ := json.Marshal(map[string]any{"type": "response.output_text.delta", "delta": text})
		fmt.Fprintf(w, "data: %s\n\ndata: {\"type\":\"response.completed\"}\n\n", b)
		return
	}
	b, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"delta": map[string]string{"content": text}}}})
	fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", b)
}
func modelBody(r *http.Request) string {
	raw, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(raw))
	return string(raw)
}
func snippetOnlyRead(context.Context, string) ([]byte, string, string, error) {
	return nil, "", "", errors.New("fixture page unavailable")
}

// Existing protocol tests use a deliberately small controller fixture; the
// research-specific tests below exercise retries, reads and failed verification.
func fixtureResearchControl(w http.ResponseWriter, r *http.Request) bool {
	raw := modelBody(r)
	switch {
	case strings.Contains(raw, "research controller / plan"):
		modelText(w, "ollama", `{"search":true,"queries":["Public question"]}`)
	case strings.Contains(raw, "research controller / evaluate"):
		modelText(w, "ollama", `{"sufficient":true,"facts":[{"claim":"Verified reference","url":"https://example.com/source","quote":"verified reference"}]}`)
	case strings.Contains(raw, "research controller / verify"):
		modelText(w, "ollama", `{"supported":true}`)
	default:
		return false
	}
	return true
}

func TestAuthoritativeClockTimezoneMidnightDSTAndLeapDay(t *testing.T) {
	for _, tc := range []struct{ utc, zone, date, yesterday string }{
		{"2026-10-25T00:30:00Z", "Europe/London", "2026-10-25", "2026-10-24"},
		{"2026-10-25T01:30:00Z", "Europe/London", "2026-10-25", "2026-10-24"},
		{"2026-10-26T00:30:00Z", "America/New_York", "2026-10-25", "2026-10-24"},
		{"2028-03-01T00:30:00Z", "UTC", "2028-03-01", "2028-02-29"},
		{"2027-01-01T00:30:00Z", "UTC", "2027-01-01", "2026-12-31"},
	} {
		t.Run(tc.utc+tc.zone, func(t *testing.T) {
			instant, _ := time.Parse(time.RFC3339, tc.utc)
			calls := 0
			a := App{clock: func() time.Time { calls++; return instant }}
			s := Session{SearchTimeZone: tc.zone}
			one, two := a.clockInstructions(&s), a.clockInstructions(&s)
			if one != two || calls != 1 || !strings.Contains(one, tc.date) || !strings.Contains(one, tc.yesterday) || !strings.Contains(one, tc.zone) || !strings.Contains(one, tc.utc) {
				t.Fatal(one, calls)
			}
		})
	}
}
func TestResearchDatesPreserveLatestHistoryAndComparisons(t *testing.T) {
	at := time.Date(2026, 10, 26, 0, 30, 0, 0, time.UTC)
	for _, tc := range []struct{ question, query, want string }{
		{"What was the Liverpool score?", "Liverpool latest match result", "Liverpool latest match result"},
		{"Who won the 2014 World Cup final?", "2014 World Cup final score", "2014 World Cup final score"},
		{"Liverpool result yesterday", "Liverpool result today", "Liverpool result 25 October 2026"},
		{"What about yesterday?", "Atlas result 26 October 2026", "Atlas result 25 October 2026"},
		{"Compare 2026-10-24 and 2026-10-25", "weather 2026-10-24 2026-10-25", "weather 2026-10-24 2026-10-25"},
		{"Compare today and yesterday", "weather today yesterday", "weather today yesterday"},
		{"What is Beethoven's Fifth Symphony score?", "Beethoven Symphony 5 score", "Beethoven Symphony 5 score"},
	} {
		if got := normalizeResearchQuery(tc.query, tc.question, at); got != tc.want {
			t.Errorf("%s: %q != %q", tc.question, got, tc.want)
		}
	}
	q := boundedQuery(strings.Repeat("日本語", 300))
	if len(q) > 500 || !validSearchQuery(q) {
		t.Fatal("Invalid UTF8 query boundary")
	}
}
func TestPublicFollowupNeverUsesPrivateHistoryOrAttachments(t *testing.T) {
	if !safePublicQuestion("Compare the latest stable Python and Go releases.\nUse official release notes and explain the differences.") {
		t.Fatal("Raw public multiline question blocked")
	}
	history := []Message{{Role: "user", Content: "Atlas score today\nAttached reference (untrusted content): PRIVATE_ATTACHMENT", PublicQuestion: "Atlas score today"}, {Role: "assistant", Content: "PRIVATE_ASSISTANT"}}
	q := researchQuestion("What about yesterday?", history)
	if strings.Contains(q, "PRIVATE") || strings.Contains(q, "today") || !strings.Contains(q, "Atlas") || !strings.Contains(q, "yesterday") {
		t.Fatal(q)
	}
	for _, input := range []string{"latest news\nPRIVATE_FILE", "latest attached document", "current api key sk-fixture-key-123456", "latest change in my project"} {
		if researchQuestion(input, nil) != "" {
			t.Fatal("Private query allowed", input)
		}
	}
	if researchQuestion("What about yesterday?", []Message{{Role: "user", Content: "latest news\nAttached reference: PRIVATE"}}) != "" {
		t.Fatal("Expanded legacy attachment entered a follow-up")
	}
	if researchQuestion("Who won the 2014 World Cup final?", nil) == "" {
		t.Fatal("Standalone historical question mistaken for a follow-up")
	}
	s := Session{SearchTimeZone: "Europe/London", Messages: history}
	raw, _ := json.Marshal(s)
	var restored Session
	json.Unmarshal(raw, &restored)
	if restored.SearchTimeZone != s.SearchTimeZone || restored.Messages[0].PublicQuestion != "Atlas score today" {
		t.Fatal("Timezone/public question did not survive reload")
	}
}
func TestWebDocumentKeepsRelevantDatesAndSkipsNavigation(t *testing.T) {
	html := `<html><head><title>Official results</title><meta property="article:published_time" content="2026-10-26"></head><body><nav>NAV_PRIVATE_TRAP</nav><main><p>` + strings.Repeat("Unrelated navigation and older material. ", 800) + `</p><article><time datetime="2026-10-25">Sunday</time><p>Atlas defeated Beacon 2–0 on 25 October 2026. Final.</p><a href="/match">Match report</a><script>INJECTED_SCRIPT</script></article></main><footer>FOOTER_TRAP</footer></body></html>`
	d := extractWebDocument(html, "text/html", "https://example.com/results", "Atlas Beacon 25 October")
	if !strings.Contains(d.Text, "2026-10-25") || !strings.Contains(d.Text, "Atlas defeated Beacon") || strings.Contains(d.Text, "TRAP") || strings.Contains(d.Text, "INJECTED_SCRIPT") || d.Published != "2026-10-26" || len(d.Links) != 1 || d.Links[0].URL != "https://example.com/match" {
		t.Fatal(d)
	}
}

func TestResearchReadsRefinesAndVerifiesWithoutModelToolSupport(t *testing.T) {
	for _, provider := range []string{"ollama", "openai", "deepseek"} {
		t.Run(provider, func(t *testing.T) {
			searches, evaluations, mainCalls := 0, 0, 0
			var reads syncatomic.Int32
			search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				searches++
				q := r.URL.Query().Get("q")
				if strings.Contains(q, "PRIVATE") || strings.Contains(q, "reference date") || strings.Contains(q, "26 October") {
					t.Error("Wrong public/latest query", q)
				}
				fmt.Fprint(w, `{"results":[{"title":"Old event","url":"https://example.com/old","content":"Atlas won 4-1 on 11 October 2026."},{"url":"https://example.com/empty1"},{"url":"https://example.com/empty2"},{"title":"Official results","url":"https://example.com/results","content":"Full season results and reports."}]}`)
			}))
			defer search.Close()
			model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				raw := modelBody(r)
				if strings.Contains(raw, "research controller /") && strings.Contains(raw, "PRIVATE") {
					t.Error("Private material reached research controller")
				}
				switch {
				case strings.Contains(raw, "research controller / plan"):
					modelText(w, provider, `{"search":true,"queries":["Atlas latest match result"]}`)
				case strings.Contains(raw, "research controller / evaluate"):
					evaluations++
					if evaluations == 1 {
						modelText(w, provider, `{"sufficient":false,"open":["https://example.com/results","http://127.0.0.1/private"]}`)
					} else {
						modelText(w, provider, `{"sufficient":true,"facts":[{"claim":"Atlas won 2-0 on 25 October 2026","url":"https://example.com/results","quote":"Atlas defeated Beacon 2-0 on 25 October 2026. Final."}]}`)
					}
				case strings.Contains(raw, "research controller / verify"):
					modelText(w, provider, `{"supported":true}`)
				default:
					mainCalls++
					if strings.Contains(raw, `"tools"`) {
						w.WriteHeader(400)
						fmt.Fprint(w, `{"error":"model does not support tools"}`)
						return
					}
					modelText(w, provider, "Atlas beat Beacon 2-0 on 25 October 2026. [Official results](https://example.com/results)")
				}
			}))
			defer model.Close()
			a := App{clock: func() time.Time { return time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC) }, searchConnectionOverride: func(context.Context) (searchruntime.Connection, error) {
				return searchruntime.Connection{URL: search.URL, Token: strings.Repeat("a", 64)}, nil
			}, webReadOverride: func(ctx context.Context, url string) ([]byte, string, string, error) {
				reads.Add(1)
				if url == "http://127.0.0.1/private" {
					t.Error("Unretrieved URL was fetched")
				}
				if url == "https://example.com/results" {
					return []byte(`<main><time datetime="2026-10-25"></time><p>Atlas defeated Beacon 2-0 on 25 October 2026. Final.</p></main>`), "text/html", url, nil
				}
				return []byte(`<main>Old reference only.</main>`), "text/html", url, nil
			}}
			s := Session{Model: "fixture", ToolInput: "What was the Atlas score?", SearchTimeZone: "Europe/London", Messages: []Message{{Role: "user", Content: "What was the Atlas score?\nAttached reference: PRIVATE_ATTACHMENT"}}}
			var displayed strings.Builder
			answer, err := a.chatWithTools(context.Background(), Provider{ID: provider, Base: model.URL}, &s, func(k string, v any) {
				if k == "delta" {
					displayed.WriteString(v.(string))
				}
			}, toolScope{})
			if err != nil || answer != displayed.String() || !strings.Contains(answer, "2-0") || strings.Contains(answer, "4-1") || searches != 1 || reads.Load() != 4 || evaluations < 2 || mainCalls != 1 {
				t.Fatal(answer, err, searches, reads.Load(), evaluations, mainCalls)
			}
		})
	}
}

func TestResearchRejectsFabricatedQuotesAndUnknownCitations(t *testing.T) {
	sources := []webResult{{URL: "https://example.com/report", Snippet: "Atlas defeated Beacon 2-0 on 25 October 2026."}}
	if len(verifiedFacts([]researchFact{{Claim: "Wrong", URL: sources[0].URL, Quote: "Atlas won 9-0 yesterday."}}, sources)) != 0 {
		t.Fatal("Fabricated evidence accepted")
	}
	if !knownCitations("2-0 [report](https://example.com/report)", sources) || knownCitations("2-0 [fake](https://invented.example/report)", sources) {
		t.Fatal("Citation provenance broken")
	}
}

func TestUnresponsiveEnginesAndPublishedDatesAreNotEventEvidence(t *testing.T) {
	r, err := searxResults("query", []byte(`{"unresponsive_engines":[["bing","timeout"]],"results":[{"url":"https://example.com/report","title":"Old event","content":"Event occurred in 2014","publishedDate":"2026-10-26","engines":["brave"]}]}`))
	if err != nil || len(r.Warnings) != 1 || r.Sources[0].Published != "2026-10-26" || r.Sources[0].Snippet != "Event occurred in 2014" {
		t.Fatal(r, err)
	}
}

func TestResearchDiscardsInventedSummaryAndUnsupportedNumbers(t *testing.T) {
	sources := []webResult{{URL: "https://example.com/report", Snippet: "Germany won the 2014 FIFA World Cup."}}
	facts := verifiedFacts([]researchFact{{Claim: "Thomas scored in minute 110", URL: sources[0].URL, Quote: sources[0].Snippet}}, sources)
	if len(facts) != 1 || facts[0].Claim != sources[0].Snippet {
		t.Fatal("Unverified model summary retained", facts)
	}
	b := researchBundle{Question: "Who won the 2014 World Cup?", Facts: facts}
	if researchNumbersSupported("Germany won 1-0 in minute 110. [Source](https://example.com/report)", b) {
		t.Fatal("Unsupported score accepted")
	}
	if !researchNumbersSupported("1. Germany won the 2014 World Cup. [Source](https://example.com/report)", b) {
		t.Fatal("List index treated as a claim")
	}
}
func TestResearchSnapshotKeepsLaterRelevantResultsAndNavigableLinks(t *testing.T) {
	b := researchBundle{Question: "latest Comet release"}
	for i := 0; i < 14; i++ {
		b.Sources = append(b.Sources, webResult{Title: "Unrelated headlines", URL: fmt.Sprintf("https://example.com/news/%d", i), Text: strings.Repeat("unrelated ", 300)})
	}
	b.Sources = append(b.Sources, webResult{Title: "Comet downloads", URL: "https://comet.example/download", Snippet: "Comet stable release", Links: []pageLink{{Title: "Release history", URL: "https://comet.example/releases"}}})
	snapshot := researchSnapshot(b)
	if len(snapshot.Sources) == 0 || snapshot.Sources[0].Title != "Comet downloads" || len(snapshot.Sources[0].Links) != 1 {
		t.Fatal("Refined source/link hidden", snapshot)
	}
}

func TestWebDocumentReportsTruncatedInput(t *testing.T) {
	d := extractWebDocument("<main>Current release: Comet 8.2</main>", "text/html; myai-truncated=1", "https://example.com/releases", "Comet release")
	if !d.Truncated || !strings.Contains(d.Text, "Comet 8.2") {
		t.Fatal(d)
	}
}

func TestDraftCitationsBringActualUnselectedEvidenceIntoVerification(t *testing.T) {
	b := researchBundle{Question: "Comet release?", Sources: []webResult{{URL: "https://example.com/releases", Snippet: "Comet 8.2 is the stable release."}}}
	b = citedResearchEvidence(b, "Comet 8.2 is stable. [Release](https://example.com/releases)")
	if len(b.Facts) != 1 || !researchNumbersSupported("Comet 8.2 is stable.", b) {
		t.Fatal("Retrieved cited evidence lost", b)
	}
	if len(citedResearchEvidence(researchBundle{Sources: b.Sources}, "[Fake](https://invented.example/releases)").Facts) != 0 {
		t.Fatal("Unknown citation gained evidence")
	}
}

func TestWebDocumentRetainsArticleHeaderAndAccessibleScore(t *testing.T) {
	d := extractWebDocument(`<main><header><time datetime="2026-10-25">Sunday</time><h1>Final result</h1></header><div><img alt="Atlas">2 - 0<img alt="Beacon"></div></main>`, "text/html", "https://example.com/results", "Atlas result")
	if !strings.Contains(d.Text, "Atlas") || !strings.Contains(d.Text, "Beacon") || !strings.Contains(d.Text, "2026-10-25") || !strings.Contains(d.Text, "2 - 0") {
		t.Fatal(d)
	}
	generic := webResult{Title: "Atlas latest scores", URL: "https://example.com/scores", Snippet: "Latest Atlas results and news."}
	concrete := webResult{Title: "Atlas news", URL: "https://example.com/news", Snippet: "Atlas beat Beacon 2-0 in the latest match."}
	if researchSourceScore(concrete, "what was the Atlas score") <= researchSourceScore(generic, "what was the Atlas score") {
		t.Fatal("Generic index crowded out actual result")
	}
}

func TestClarificationsMustRequestNewInformation(t *testing.T) {
	if !echoedClarification("What was the Liverpool score yesterday?", "what was the Liverpool score; What about yesterday?") {
		t.Fatal("Restated followup accepted as clarification")
	}
	if echoedClarification("Which city are you in?", "What is the weather near me?") {
		t.Fatal("Missing location clarification blocked")
	}
}
