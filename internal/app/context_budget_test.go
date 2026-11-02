package app

import (
	"context"
	"encoding/json"
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestContextSizingModelMemoryAndOverrides(t *testing.T) {
	facts := modelContextFacts{Limit: 131072, Weights: 2 << 30, KVPerToken: 8192}
	rich := hostMemory{RAMTotal: 128 << 30, RAMAvailable: 96 << 30, GPUTotal: 24 << 30, GPUFree: 22 << 30}
	b := contextChoice(facts, rich, nil, "code")
	if b.Chat != 32768 || b.Code != 65536 || b.Selected != 65536 || b.Reply != 8192 {
		t.Fatal(b)
	}
	low := contextChoice(modelContextFacts{Limit: 8192, Weights: 6 << 30, KVPerToken: 128 << 10}, hostMemory{RAMAvailable: 8 << 30}, nil, "code")
	if low.Code != 512 || low.Selected > 8192 {
		t.Fatal(low)
	}
	small := contextChoice(modelContextFacts{Limit: 4096, Weights: 1 << 30, KVPerToken: 8192}, rich, nil, "code")
	if small.Code != 4096 || small.Chat != 4096 {
		t.Fatal(small)
	}
	manual := LocalModelSettings{ContextTokens: tuneInt(8192), CodeContextTokens: tuneInt(262144), MaxOutputTokens: tuneInt(32768)}
	b = contextChoice(facts, rich, &manual, "code")
	if b.Chat != 8192 || b.Code != 131072 || !strings.Contains(b.Notice, "capped") {
		t.Fatal(b)
	}
	if manual.CodeContextTokens == nil || *manual.CodeContextTokens != 262144 {
		t.Fatal("Changed saved override")
	}
	unknown := contextChoice(modelContextFacts{}, hostMemory{}, &manual, "code")
	if unknown.Code != 8192 || unknown.ModelMax != 0 || !strings.Contains(unknown.Notice, "unavailable") {
		t.Fatal(unknown)
	}
}

func TestEffectiveContextReachesEveryLocalWireAndRejectsOverflow(t *testing.T) {
	requests := 0
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var v map[string]any
		json.NewDecoder(r.Body).Decode(&v)
		o := v["options"].(map[string]any)
		want := 65536
		if requests == 2 {
			want = 16384
		}
		if int(o["num_ctx"].(float64)) != want {
			t.Error("Budget not applied", o)
		}
		w.Write([]byte("{\"message\":{\"content\":\"ok\"},\"done\":true}\n"))
	}))
	defer up.Close()
	a := App{}
	b := contextChoice(modelContextFacts{Limit: 131072, Weights: 1 << 30, KVPerToken: 8192}, hostMemory{RAMAvailable: 64 << 30}, nil, "code")
	p := Provider{ID: "ollama", Base: up.URL, Context: &b, Generation: &LocalModelSettings{ContextTokens: tuneInt(1024)}}
	messages := []any{map[string]string{"role": "user", "content": "Build a page"}}
	if _, err := a.generate(context.Background(), p, "fixture", messages, toolScope{}, false, func(string, any) {}); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), researchGenerationKey{}, true)
	if _, err := a.generate(ctx, p, "fixture", messages, toolScope{}, false, func(string, any) {}); err != nil {
		t.Fatal(err)
	}
	b.Selected = 1024
	if _, err := a.generate(context.Background(), p, "fixture", []any{map[string]string{"role": "user", "content": strings.Repeat("x", 5000)}}, toolScope{}, false, func(string, any) {}); err == nil || requests != 2 {
		t.Fatal("Overflow sent to provider", err, requests)
	}
}

func TestCodeContextProfilePersistsIndependently(t *testing.T) {
	a := App{H: &apphost.Host{Config: apphost.Config{Data: t.TempDir()}}}
	s := LocalModelSettings{ContextTokens: tuneInt(8192), CodeContextTokens: tuneInt(65536), MaxOutputTokens: tuneInt(1024)}
	if err := a.saveLocalSettings("fixture-one", "fixture:latest", s); err != nil {
		t.Fatal(err)
	}
	restarted := App{H: a.H}
	one, err := restarted.readLocalSettings("fixture-one")
	if err != nil {
		t.Fatal(err)
	}
	got := one.Models["fixture:latest"]
	if got.CodeContextTokens == nil || *got.CodeContextTokens != 65536 || *got.ContextTokens != 8192 {
		t.Fatal("Separate contexts were lost")
	}
	two, err := restarted.readLocalSettings("fixture-two")
	if err != nil || len(two.Models) != 0 {
		t.Fatal("Profile leaked across accounts", err)
	}
	if err := a.saveLocalSettings("fixture-one", "fixture:latest", LocalModelSettings{}); err != nil {
		t.Fatal(err)
	}
	one, _ = a.readLocalSettings("fixture-one")
	if len(one.Models) != 0 {
		t.Fatal("Automatic reset retained a context override")
	}
}

func TestContextEndpointRequiresAccount(t *testing.T) {
	h := &apphost.Host{Mux: http.NewServeMux(), Config: apphost.Config{Data: t.TempDir()}}
	a := App{H: h}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://localhost/api/model-context?model=fixture", nil)
	h.Require(a.localModelContext)(w, r)
	if w.Code != 401 && w.Code != 403 {
		t.Fatal("Anonymous context probe accepted", w.Code)
	}
}
