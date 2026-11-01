package app

import (
	"context"
	"encoding/json"
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func tuneInt(n int) *int           { return &n }
func tuneFloat(n float64) *float64 { return &n }
func TestLocalSettingsValidationPersistenceAndIsolation(t *testing.T) {
	a := &App{H: &apphost.Host{Config: apphost.Config{Data: t.TempDir()}}}
	s := LocalModelSettings{ContextTokens: tuneInt(2048), MaxOutputTokens: tuneInt(512), Temperature: tuneFloat(0), TopK: tuneInt(0), Seed: tuneInt(0)}
	if err := a.saveLocalSettings("one", "fixture:small", s); err != nil {
		t.Fatal(err)
	}
	if err := a.saveLocalSettings("one", "fixture:large", LocalModelSettings{ContextTokens: tuneInt(32768)}); err != nil {
		t.Fatal(err)
	}
	restarted := &App{H: a.H}
	one, err := restarted.readLocalSettings("one")
	if err != nil {
		t.Fatal(err)
	}
	two, err := restarted.readLocalSettings("two")
	if err != nil {
		t.Fatal(err)
	}
	if len(one.Models) != 2 || len(two.Models) != 0 || *one.Models["fixture:small"].Temperature != 0 || *one.Models["fixture:small"].Seed != 0 {
		t.Fatal("isolation or zero-value loss")
	}
	path, _ := a.localSettingsPath("one")
	before, _ := os.ReadFile(path)
	for _, bad := range []LocalModelSettings{{ContextTokens: tuneInt(0)}, {MaxOutputTokens: tuneInt(-1)}, {ContextTokens: tuneInt(1024), MaxOutputTokens: tuneInt(1024)}, {Temperature: tuneFloat(math.NaN())}, {TopP: tuneFloat(2)}, {TopK: tuneInt(-1)}, {RepeatPenalty: tuneFloat(math.Inf(1))}, {BatchSize: tuneInt(0)}, {Seed: tuneInt(-1)}, {ChatMode: "unsafe"}, {KeepAlive: "forever"}} {
		if err := a.saveLocalSettings("one", "fixture:small", bad); err == nil {
			t.Fatal("invalid settings accepted", bad)
		}
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(before) {
		t.Fatal("failed validation changed storage")
	}
	if err := a.saveLocalSettings("one", "fixture:small", LocalModelSettings{ChatMode: "auto"}); err != nil {
		t.Fatal(err)
	}
	one, _ = a.readLocalSettings("one")
	if len(one.Models) != 1 || one.Models["fixture:large"].ContextTokens == nil {
		t.Fatal("reset changed another model")
	}
	os.WriteFile(path, []byte("corrupt fixture"), 0600)
	if err := a.saveLocalSettings("one", "fixture:small", s); err == nil {
		t.Fatal("corrupt settings overwritten")
	}
	after, _ = os.ReadFile(path)
	if string(after) != "corrupt fixture" {
		t.Fatal("corrupt store lost")
	}
}
func TestLocalSettingsActualWireOptionsAcrossGenerationPaths(t *testing.T) {
	var body map[string]any
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Error(r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&body)
		w.Write([]byte("{\"message\":{\"content\":\"fixture\"},\"done\":true}\n"))
	}))
	defer up.Close()
	settings := LocalModelSettings{ContextTokens: tuneInt(4096), MaxOutputTokens: tuneInt(2048), Temperature: tuneFloat(0.7), TopK: tuneInt(0), Seed: tuneInt(0), BatchSize: tuneInt(128), KeepAlive: "0"}
	a := &App{}
	provider := Provider{ID: "ollama", Base: up.URL, Generation: &settings}
	for _, kind := range []string{"chat", "code", "research", "answer"} {
		ctx := context.Background()
		scope := toolScope{}
		if kind == "code" {
			scope.Project = "fixture"
		}
		if kind == "research" {
			ctx = context.WithValue(ctx, researchGenerationKey{}, true)
		}
		if kind == "answer" {
			ctx = context.WithValue(ctx, researchAnswerKey{}, true)
		}
		if _, err := a.generate(ctx, provider, "fixture", nil, scope, false, func(string, any) {}); err != nil {
			t.Fatal(err)
		}
		o := body["options"].(map[string]any)
		limit := float64(2048)
		temp := 0.7
		if kind == "research" {
			limit = 800
			temp = 0
		}
		if o["num_ctx"] != float64(4096) || o["num_predict"] != limit || o["temperature"] != temp || o["num_batch"] != float64(128) || body["keep_alive"] != float64(0) {
			t.Fatal(kind, body)
		}
		if kind != "research" && (o["top_k"] != float64(0) || o["seed"] != float64(0)) {
			t.Fatal("zero values lost", body)
		}
	}
	settings = LocalModelSettings{ContextTokens: tuneInt(2048)}
	a.generate(context.Background(), provider, "fixture", nil, toolScope{}, false, func(string, any) {})
	if body["options"].(map[string]any)["num_predict"] != float64(512) {
		t.Fatal(body)
	}
	settings = LocalModelSettings{}
	a.generate(context.Background(), provider, "fixture", nil, toolScope{Project: "fixture"}, false, func(string, any) {})
	if body["options"].(map[string]any)["num_ctx"] != float64(16384) {
		t.Fatal("defaults changed")
	}
}
func TestPlainChatDoesNotUseToolsResearchOrOrphanToolHistory(t *testing.T) {
	t.Setenv("MYAI_WORKSPACE_URL", "http://127.0.0.1:1")
	calls := 0
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var b map[string]any
		json.NewDecoder(r.Body).Decode(&b)
		if b["tools"] != nil || b["format"] != nil {
			t.Error("plain chat used tools", b)
		}
		messages := b["messages"].([]any)
		if len(messages) != 3 {
			t.Error("orphan replay", messages)
		}
		for _, v := range messages {
			m := v.(map[string]any)
			if m["role"] == "tool" || m["tool_calls"] != nil {
				t.Error("tool replay", m)
			}
		}
		w.Write([]byte("{\"message\":{\"content\":\"plain fixture\"},\"done\":true}\n"))
	}))
	defer up.Close()
	a := &App{}
	s := &Session{Model: "fixture", SearchMode: "on", ToolInput: "deploy with My Hosting", Messages: []Message{{Role: "user", Content: "old"}, {Role: "assistant", Calls: []functionCall{{Name: "web_search"}}}, {Role: "tool", Content: "private reference"}, {Role: "user", Content: "deploy with My Hosting"}}}
	p := Provider{ID: "ollama", Base: up.URL, Generation: &LocalModelSettings{ChatMode: "plain", ContextTokens: tuneInt(2048)}}
	answer, err := a.chatWithTools(context.Background(), p, s, func(string, any) {}, toolScope{WorkspaceSession: "fixture"})
	if err != nil || answer != "plain fixture" || calls != 1 {
		t.Fatal(answer, err, calls)
	}
}
func TestLocalSettingsAuthenticationAndProfileLimit(t *testing.T) {
	h := &apphost.Host{Mux: http.NewServeMux(), Config: apphost.Config{Data: t.TempDir()}}
	a := &App{H: h}
	for _, method := range []string{"GET", "POST"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "http://localhost/api/model-settings?model=fixture", strings.NewReader("{}"))
		h.Require(a.localModelSettings)(w, r)
		if w.Code != 401 && w.Code != 403 {
			t.Fatal(w.Code)
		}
	}
	saved := localSettingsFile{Schema: 1, Models: map[string]LocalModelSettings{}}
	for i := 0; i < 128; i++ {
		saved.Models[strings.Repeat("a", i+1)] = LocalModelSettings{Temperature: tuneFloat(0)}
	}
	path, _ := a.localSettingsPath("fixture")
	if err := atomic(path, saved); err != nil {
		t.Fatal(err)
	}
	if err := a.saveLocalSettings("fixture", "extra", LocalModelSettings{Temperature: tuneFloat(0)}); err == nil {
		t.Fatal("profile limit ignored")
	}
}
