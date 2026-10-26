package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	atomicstate "sync/atomic"
	"testing"
	"time"

	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
)

func TestWorkspaceActionsRequireSameConversationHumanReview(t *testing.T) {
	db, e := sql.Open("sqlite", ":memory:")
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	issuer, subject := "https://accounts.example.test", "member"
	uid := apphost.Identity(issuer, subject)
	if _, e = db.Exec(`CREATE TABLE host_users(id TEXT,subject TEXT,issuer TEXT,name TEXT,role TEXT); INSERT INTO host_users VALUES(?,?,?,?,?)`, uid, subject, issuer, "Fixture", "member"); e != nil {
		t.Fatal(e)
	}
	var submitted atomicstate.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Workspace-Browser-Session") != "private-browser-capability" || r.Header.Get("Authorization") != "Bearer private-bridge-capability" || r.Header.Get("X-MyAI-Subject") != subject || r.Header.Get("Cookie") != "" {
			http.Error(w, "wrong authority", 403)
			return
		}
		if r.Method == "GET" {
			_ = json.NewEncoder(w).Encode(workspaceCatalog{Contract: "workspace-apps-v1", Apps: []workspaceApp{{ID: "hosting", Name: "My Hosting", NativeAccess: true, Operations: []workspaceOperation{{ID: "create_site", Title: "Create a site", Review: true}}}}})
			return
		}
		var call workspaceCall
		if json.NewDecoder(r.Body).Decode(&call) != nil {
			t.Error("invalid app packet")
		}
		if call.Prepare {
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"app": "My Hosting", "title": "Create a site", "name": "Example"}})
			return
		}
		if !call.Reviewed {
			t.Error("human review bypassed")
		}
		submitted.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]string{"id": "native-site", "state": "created"}})
	}))
	defer server.Close()
	t.Setenv("MYAI_WORKSPACE_URL", server.URL+"/_integration/myai/workspace")
	t.Setenv("MYAI_STORAGE_GUARD_TOKEN", "private-bridge-capability")
	a := &App{H: &apphost.Host{DB: db, Config: apphost.Config{Issuer: issuer}}}
	private := false
	declined := false
	scope := toolScope{UID: uid, WorkspaceSession: "private-browser-capability", PrivateAppRead: &private, WorkspaceKnown: map[string]bool{"hosting": true}, WorkspaceDeclined: &declined}
	s := &Session{ID: strings.Repeat("a", 48), Mode: "chat", WorkspaceSession: scope.WorkspaceSession}
	approval := make(chan string, 1)
	done := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go func() {
		_, e := a.useWorkspaceApp(ctx, scope, s, func(kind string, value any) {
			if kind == "approval" {
				raw, _ := json.Marshal(value)
				if strings.Contains(string(raw), "private-browser") || strings.Contains(string(raw), "private-bridge") {
					t.Error("credential exposed in review")
				}
				approval <- value.(map[string]any)["id"].(string)
			}
		}, map[string]string{"app": "hosting", "operation": "create_site", "arguments": `{"body":{"name":"Example"}}`})
		done <- e
	}()
	id := <-approval
	if submitted.Load() != 0 || a.answerWorkspaceReview("someone-else", s.ID, id, true) || a.answerWorkspaceReview(uid, "another-conversation", id, true) {
		t.Fatal("foreign review authority accepted")
	}
	if !a.answerWorkspaceReview(uid, s.ID, id, true) {
		t.Fatal("own review refused")
	}
	if e = <-done; e != nil || submitted.Load() != 1 || !private || !s.WorkspaceContext {
		t.Fatal("native action did not complete", e)
	}
	if a.answerWorkspaceReview(uid, s.ID, id, true) {
		t.Fatal("approval reused")
	}
	if _, e = a.executeTool(ctx, scope, "web_search", map[string]string{"query": "private app content"}); e == nil {
		t.Fatal("private app context leaked to web")
	}
	if _, e = a.useWorkspaceApp(ctx, scope, s, func(string, any) {}, map[string]string{"app": "hosting", "operation": "create_site", "arguments": `{"reviewed":true}`}); e == nil {
		t.Fatal("model forged approval flag")
	}
	go func() {
		_, e := a.useWorkspaceApp(ctx, scope, s, func(kind string, value any) {
			if kind == "approval" {
				approval <- value.(map[string]any)["id"].(string)
			}
		}, map[string]string{"app": "hosting", "operation": "create_site", "arguments": `{"body":{"name":"Declined fixture"}}`})
		done <- e
	}()
	id = <-approval
	if !a.answerWorkspaceReview(uid, s.ID, id, false) {
		t.Fatal("Decline refused")
	}
	if e = <-done; e != errWorkspaceDeclined || !declined {
		t.Fatal("Decline did not end changes", e)
	}
	if _, e = a.useWorkspaceApp(ctx, scope, s, func(string, any) { t.Error("Declined action asked again") }, map[string]string{"app": "hosting", "operation": "create_site", "arguments": `{"body":{"name":"Repeated fixture"}}`}); e != errWorkspaceDeclined || submitted.Load() != 1 {
		t.Fatal("Model retried a declined change", e)
	}
	declined = false // A separate subsequent turn can request a new change.
	s.Mode = "code"
	s.Policy = "read-only"
	if _, e = a.useWorkspaceApp(ctx, scope, s, func(string, any) {}, map[string]string{"app": "hosting", "operation": "create_site", "arguments": `{}`}); e == nil {
		t.Fatal("preview-only changed another app")
	}
	raw, _ := json.Marshal(s)
	if strings.Contains(string(raw), scope.WorkspaceSession) {
		t.Fatal("Workspace browser capability persisted in session")
	}
}

func TestWorkspacePrivateQuestionsStayOutOfPublicResearch(t *testing.T) {
	for _, q := range []string{"Find my tasks due tomorrow", "Search my drive for invoices", "Deploy this project", "Please save this in My Drive", "What apps are in One Workspace?", "List my notes"} {
		if safePublicQuestion(q) || researchQuestion(q, nil) != "" {
			t.Errorf("Private app request became a public query: %q", q)
		}
	}
	if !safePublicQuestion("How do static website deployments work?") || !safePublicQuestion("What was the Liverpool score yesterday?") {
		t.Fatal("Public questions incorrectly disabled")
	}
	private := true
	for _, spec := range toolSpecs(toolScope{PrivateAppRead: &private, WorkspaceSession: "ephemeral"}) {
		if spec.Name == "web_search" || spec.Name == "read_webpage" {
			t.Fatal("Model with private app history received outgoing web tools")
		}
	}
	var restored Session
	if json.Unmarshal([]byte(`{"workspaceContext":true}`), &restored) != nil || !restored.WorkspaceContext {
		t.Fatal("Private context boundary did not survive a later turn")
	}
}

func TestWorkspaceStructuredModelArguments(t *testing.T) {
	args, e := workspaceModelArguments(`{"app":"tasks","operation":"create_task","parameters":{},"query":{},"body":{"title":"Fixture","version":2,"completed":false}}`)
	if e != nil || args["app"] != "tasks" || args["operation"] != "create_task" {
		t.Fatal("Structured action rejected", e)
	}
	var body workspaceArgs
	if json.Unmarshal([]byte(args["arguments"]), &body) != nil || !strings.Contains(string(body.Body), `"version":2`) {
		t.Fatal("Typed native body changed")
	}
	for _, raw := range []string{`{"app":"tasks","arguments":"{\"operation\":\"lists\"}"}`, `{"app":"tasks","operation":"lists","reviewed":true}`, `{"app":"tasks","operation":"lists","headers":{"Authorization":"forged"}}`, `{"app":"tasks","operation":"lists","query":{"offset":3}}`} {
		if _, e := workspaceModelArguments(raw); e == nil {
			t.Fatal("Ambiguous or untrusted argument accepted", raw)
		}
	}
	if _, e := workspaceModelArguments(`{"app":"tasks","operation":"lists","parameters":{},"query":{},"body":{}}`); e != nil {
		t.Fatal(e)
	}
}

func TestWorkspaceReviewCancellationDoesNotSubmit(t *testing.T) {
	a := &App{}
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan string, 1)
	done := make(chan error, 1)
	go func() {
		done <- a.reviewWorkspaceAction(ctx, "one", "session", func(kind string, value any) { ready <- value.(map[string]any)["id"].(string) }, "My Hosting", "Deploy", map[string]string{"site": "Example"})
	}()
	id := <-ready
	cancel()
	if e := <-done; e == nil {
		t.Fatal("cancelled review completed")
	}
	if a.answerWorkspaceReview("one", "session", id, true) {
		t.Fatal("cancelled review remained executable")
	}
}

func TestWorkspaceToolsRemainAbsentFromPublicMCP(t *testing.T) {
	for _, spec := range toolSpecs(toolScope{}) {
		if strings.Contains(spec.Name, "workspace") {
			t.Fatal("private app authority exposed through public MCP")
		}
	}
	names := map[string]bool{}
	for _, spec := range toolSpecs(toolScope{WorkspaceSession: "ephemeral"}) {
		names[spec.Name] = true
	}
	if !names["get_workspace_apps"] || !names["describe_workspace_app"] || !names["use_workspace_app"] {
		t.Fatal("scoped app tools unavailable")
	}
}

func TestWorkspaceDiscoveryPrecedesNativeUse(t *testing.T) {
	_, e := (&App{}).useWorkspaceApp(context.Background(), toolScope{}, &Session{}, func(string, any) { t.Error("Undiscovered action reached review") }, map[string]string{"app": "tasks", "operation": "create_task", "arguments": `{"body":{"title":"Fixture"}}`})
	if e == nil || !strings.Contains(e.Error(), "First call describe_workspace_app") {
		t.Fatal("Native schema discovery was optional", e)
	}
}

func TestWorkspaceAppReferencesUseInstalledCatalogue(t *testing.T) {
	catalog := workspaceCatalog{Apps: []workspaceApp{{ID: "drive", Name: "Drive"}, {ID: "tasks", Name: "My Tasks"}, {ID: "future", Name: "My Future App"}}}
	for input, want := range map[string]string{"Save it in My Drive": "drive", "Use the connected Drive app": "drive", "Find this in Tasks": "tasks", "Open My Future App": "future"} {
		apps := referencedWorkspaceApps(input, catalog)
		if len(apps) != 1 || apps[0].ID != want {
			t.Fatal("Installed app not resolved", input, apps)
		}
	}
	for _, input := range []string{"How far should I drive today?", "What are background tasks?", "Latest Liverpool score"} {
		if apps := referencedWorkspaceApps(input, catalog); len(apps) != 0 {
			t.Fatal("Public question routed to a private app", input, apps)
		}
	}
}
