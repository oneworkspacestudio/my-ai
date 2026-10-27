package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
)

func TestDeploymentTrackingPersistsReceiptBeforeReadAndUsesExactJob(t *testing.T) {
	db, e := sql.Open("sqlite", ":memory:")
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	issuer, subject := "https://accounts.example.test", "job-fixture"
	uid := apphost.Identity(issuer, subject)
	if _, e = db.Exec(`CREATE TABLE host_users(id TEXT,subject TEXT,issuer TEXT,name TEXT,role TEXT); INSERT INTO host_users VALUES(?,?,?,?,?)`, uid, subject, issuer, "Fixture", "member"); e != nil {
		t.Fatal(e)
	}
	a := &App{H: &apphost.Host{DB: db, Config: apphost.Config{Issuer: issuer}}}
	root := t.TempDir()
	s := &Session{ID: strings.Repeat("a", 48), Messages: []Message{{ID: strings.Repeat("b", 48), Role: "user", Content: "Deploy"}, {Role: "assistant", Calls: []functionCall{{ID: "job-call", Name: "workspace_hosting_deploy_project", Arguments: `{}`}}}}}
	scope := toolScope{UID: uid, Root: root, WorkspaceSession: "fixture", CallID: "job-call", InvokedTool: "workspace_hosting_deploy_project"}
	reads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads++
		var call workspaceCall
		_ = json.NewDecoder(r.Body).Decode(&call)
		if call.Operation != "deployment" || call.Parameters["id"] != "site-a" || call.Parameters["deployment"] != "job-b" || call.Reviewed {
			t.Error("Polling changed authority or reversed site/job IDs")
		}
		stored, err := a.read(root, s.ID)
		if err != nil {
			t.Error("Receipt not saved before tracking", err)
		}
		if !strings.Contains(stored.Messages[len(stored.Messages)-1].Content, `"state":"queued"`) {
			t.Error("Queued receipt missing before read")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"deployment": map[string]any{"id": "job-b", "projectId": "site-a", "state": "published", "url": "http://site-a.localhost:39680"}}})
	}))
	defer server.Close()
	t.Setenv("MYAI_WORKSPACE_URL", server.URL+"/_integration/myai/workspace")
	t.Setenv("MYAI_STORAGE_GUARD_TOKEN", "fixture")
	app := workspaceApp{ID: "hosting", Operations: []workspaceOperation{{ID: "deployment"}}}
	call := workspaceCall{Parameters: map[string]string{"site": "site-a"}}
	receipt := func() map[string]any {
		return map[string]any{"result": map[string]any{"site": "site-a", "deployment": map[string]any{"id": "job-b", "projectId": "site-a", "state": "queued"}}}
	}
	result := a.trackWorkspaceDeployment(context.Background(), scope, s, func(string, any) {}, app, call, receipt())
	b, _ := json.Marshal(result)
	if reads != 1 || !strings.Contains(string(b), `"state":"published"`) || len(s.Messages) != 3 {
		t.Fatal("Tracking lost outcome or duplicated tool result", reads, string(b), len(s.Messages))
	}
	stored, err := a.read(root, s.ID)
	if err != nil || !strings.Contains(stored.Messages[2].Content, `"state":"published"`) {
		t.Fatal("Final receipt not durable", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result = a.trackWorkspaceDeployment(ctx, scope, s, func(string, any) {}, app, call, receipt())
	b, _ = json.Marshal(result)
	if reads != 1 || !strings.Contains(string(b), `"state":"queued"`) || !strings.Contains(string(b), "trackingNotice") {
		t.Fatal("Stop lost submitted job or pretended it completed", string(b), reads)
	}
}

func TestHostingNativeToolsUseSemanticSiteAndDeploymentFields(t *testing.T) {
	scope := toolScope{WorkspaceSession: "fixture", WorkspaceNative: map[string]workspaceNativeTool{}}
	registerWorkspaceNative(scope, workspaceApp{ID: "hosting", Operations: []workspaceOperation{{ID: "deployment", Parameters: []string{"id", "deployment"}}}})
	tool := scope.WorkspaceNative["workspace_hosting_deployment"]
	b, _ := json.Marshal(tool.Spec)
	if strings.Contains(string(b), `"id":`) || !strings.Contains(string(b), `"site":`) {
		t.Fatal("Ambiguous native ID exposed", string(b))
	}
	args, err := workspaceNativeArguments(tool, `{"site":"site-a","deployment":"job-b"}`)
	if err != nil || !strings.Contains(args["arguments"], `"id":"site-a"`) || !strings.Contains(args["arguments"], `"deployment":"job-b"`) {
		t.Fatal(args, err)
	}
}
