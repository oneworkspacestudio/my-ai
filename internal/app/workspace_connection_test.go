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

func TestWorkspaceConnectionUsesBoundReadWithoutActionAuthority(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	issuer, subject := "https://accounts.example.test", "sso-fixture"
	uid := apphost.Identity(issuer, subject)
	if _, err = db.Exec("CREATE TABLE host_users(id TEXT,subject TEXT,issuer TEXT,name TEXT,role TEXT); INSERT INTO host_users VALUES(?,?,?,?,?)", uid, subject, issuer, "Fixture", "member"); err != nil {
		t.Fatal(err)
	}
	connected, changesOnly := false, false
	reads, requests := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("X-MyAI-Subject") != subject || r.Header.Get("X-MyAI-Issuer") != issuer || r.Header.Get("X-Workspace-Browser-Session") != "bound-browser" {
			t.Error("lost browser/identity binding")
		}
		if r.Method == "GET" {
			ops := []workspaceOperation{{ID: "create_site", Review: true}}
			if !changesOnly {
				ops = append(ops, workspaceOperation{ID: "sites"})
			}
			json.NewEncoder(w).Encode(workspaceCatalog{Contract: "workspace-apps-v1", Apps: []workspaceApp{{ID: "hosting", NativeAccess: true, Connected: false, Operations: ops}}})
			return
		}
		var call workspaceCall
		json.NewDecoder(r.Body).Decode(&call)
		if call.App != "hosting" || call.Operation != "sites" || call.Prepare || call.Reviewed || (len(call.Body) != 0 && string(call.Body) != "null") {
			t.Error("sign-in granted mutation authority", call)
		}
		reads++
		if !connected {
			w.WriteHeader(409)
			json.NewEncoder(w).Encode(map[string]any{"connectionRequired": true, "error": "native session missing"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"result": []any{map[string]any{"id": "private-fixture-site"}}})
	}))
	defer server.Close()
	t.Setenv("MYAI_WORKSPACE_URL", server.URL+"/_integration/myai/workspace")
	t.Setenv("MYAI_STORAGE_GUARD_TOKEN", "fixture-capability")
	a := &App{H: &apphost.Host{DB: db, Config: apphost.Config{Issuer: issuer}}}
	scope := toolScope{UID: uid, WorkspaceSession: "bound-browser", WorkspaceKnown: map[string]bool{}}
	if ok, err := a.verifyWorkspaceConnection(context.Background(), scope, "hosting"); err != nil || ok {
		t.Fatal("cold session not detected", ok, err)
	}
	connected = true
	if ok, err := a.verifyWorkspaceConnection(context.Background(), scope, "hosting"); err != nil || !ok {
		t.Fatal("native sign-in not verified", ok, err)
	}
	if reads != 2 {
		t.Fatal(reads)
	}
	before := reads
	if ok, err := a.verifyWorkspaceConnection(context.Background(), scope, "missing"); err == nil || ok || reads != before {
		t.Fatal("uninstalled app accepted", ok, err)
	}
	changesOnly = true
	if ok, err := a.verifyWorkspaceConnection(context.Background(), scope, "hosting"); err == nil || ok || reads != before {
		t.Fatal("mutation used as identity probe", ok, err)
	}
	changesOnly = false
	discovery, err := a.workspaceDiscovery(context.Background(), scope, "describe_workspace_app", "hosting")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(discovery)
	if strings.Contains(string(raw), "connected") || !strings.Contains(string(raw), "automatically") {
		t.Fatal("model receives cold-session failure metadata", string(raw))
	}
	before = requests
	if _, err = db.Exec("UPDATE host_users SET role='disabled'"); err != nil {
		t.Fatal(err)
	}
	if ok, err := a.verifyWorkspaceConnection(context.Background(), scope, "hosting"); err == nil || ok || requests != before {
		t.Fatal("disabled identity reached broker", ok, err)
	}
}
func TestWorkspaceConnectionRouteRejectsAnonymous(t *testing.T) {
	h := &apphost.Host{Mux: http.NewServeMux()}
	a := &App{H: h}
	r := httptest.NewRequest("POST", "http://localhost/api/workspace/connection", strings.NewReader("{\"app\":\"hosting\"}"))
	w := httptest.NewRecorder()
	h.Require(a.workspaceConnection)(w, r)
	if w.Code != 401 && w.Code != 403 {
		t.Fatal(w.Code)
	}
}
