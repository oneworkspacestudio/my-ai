package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type attachmentModelTransport struct {
	target   *url.URL
	original http.RoundTripper
}

func (t attachmentModelTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host == "127.0.0.1:11434" {
		r = r.Clone(r.Context())
		u := *r.URL
		u.Scheme, u.Host = t.target.Scheme, t.target.Host
		r.URL = &u
	}
	return t.original.RoundTrip(r)
}
func TestAttachedChatTurnUsesGeneralAndCopiesWithoutPromptStuffing(t *testing.T) {
	root := t.TempDir()
	issuer, subject := "https://fixture-accounts.example.test", "fixture"
	uid := apphost.Identity(issuer, subject)
	projectID := generalProjectID(uid)
	drive := filepath.Join(root, "drive")
	userRoot := filepath.Join(drive, "users", uid)
	ref, content := fixtureAttachment(t, userRoot)
	projectPath := filepath.Join(root, "Projects", "users", uid, "general")
	os.MkdirAll(projectPath, 0700)
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, schema := range []string{`CREATE TABLE host_users(id TEXT,issuer TEXT,subject TEXT,name TEXT,email TEXT,role TEXT)`, `CREATE TABLE host_sessions(id TEXT,user_id TEXT,csrf TEXT,expires TEXT)`, `CREATE TABLE host_audit(at TEXT,user_id TEXT,action TEXT,object TEXT)`} {
		if _, err = db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	db.Exec(`INSERT INTO host_users VALUES(?,?,?,?,?,?)`, uid, issuer, subject, "Synthetic person", "fixture@example.test", "member")
	token, csrf := strings.Repeat("a", 96), "fixture-csrf"
	db.Exec(`INSERT INTO host_sessions VALUES(?,?,?,?)`, apphost.Hash(token), uid, csrf, time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano))
	prepared, synced := false, false
	native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/guard" {
			w.Write([]byte("{}"))
			return
		}
		if r.Header.Get("X-MyAI-Subject") != subject || r.Header.Get("Authorization") != "Bearer fixture" {
			http.Error(w, "denied", 403)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/prepare") {
			prepared = true
		}
		if strings.HasSuffix(r.URL.Path, "/sync") {
			synced = true
		}
		json.NewEncoder(w).Encode(Project{ID: projectID, Name: "General", Folder: "general", Path: projectPath, GitDir: filepath.Join(root, "git"), Managed: true, State: "synced", Commit: strings.Repeat("c", 40)})
	}))
	defer native.Close()
	t.Setenv("MYAI_DRIVE_ROOT", drive)
	t.Setenv("MYAI_STORAGE_GUARD_URL", native.URL+"/guard")
	t.Setenv("MYAI_STORAGE_GUARD_TOKEN", "fixture")
	t.Setenv("MYAI_PROJECTS_ROOT", filepath.Join(root, "Projects"))
	t.Setenv("MYAI_PROJECTS_URL", native.URL+"/_integration/myai/projects")
	t.Setenv("MYAI_WORKSPACE_URL", "")
	modelCalls := 0
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			t.Error("Bad generation body")
		}
		raw, _ := json.Marshal(body)
		if len(raw) > 32000 || strings.Contains(string(raw), "EXACT_TAIL_MARKER") || !strings.Contains(string(raw), "Host the attached file") || !strings.Contains(string(raw), ref.ID) {
			t.Error("Attachment flooded request or lost instruction", len(raw))
		}
		if body["options"].(map[string]any)["num_ctx"] != float64(65536) {
			t.Error("Code budget not selected")
		}
		modelCalls++
		if modelCalls == 1 {
			json.NewEncoder(w).Encode(map[string]any{"done": true, "message": map[string]any{"content": "I will copy the file.", "tool_calls": []any{map[string]any{"function": map[string]any{"name": "copy_attachment_to_project", "arguments": map[string]string{"id": ref.ID, "path": "index.html"}}}}}})
		} else {
			w.Write([]byte("{\"message\":{\"content\":\"The file is saved.\"},\"done\":true}\n"))
		}
	}))
	defer model.Close()
	target, _ := url.Parse(model.URL)
	original := http.DefaultTransport
	http.DefaultTransport = attachmentModelTransport{target, original}
	t.Cleanup(func() { http.DefaultTransport = original })
	h := &apphost.Host{DB: db, Mux: http.NewServeMux(), Config: apphost.Config{ID: "myai", Issuer: issuer, Data: filepath.Join(root, "app"), Origin: "http://fixture"}}
	a := App{H: h, client: &http.Client{Timeout: 5 * time.Second}, running: map[string]context.CancelFunc{}, projectTurns: map[string]bool{}, contextProbe: func(context.Context, Provider, string) (modelContextFacts, hostMemory) {
		return modelContextFacts{Limit: 131072, Weights: 1 << 30, KVPerToken: 8192}, hostMemory{RAMAvailable: 64 << 30}
	}}
	s := Session{ID: strings.Repeat("d", 48), Title: "New conversation", Mode: "chat", Policy: "read-only", Provider: "ollama", Model: "fixture:latest"}
	if err = a.write(userRoot, &s); err != nil {
		t.Fatal(err)
	}
	input, _ := json.Marshal(map[string]any{"text": "Host the attached file", "attachments": []string{ref.ID}})
	r := httptest.NewRequest("POST", "http://fixture/api/sessions/"+s.ID+"/turn", strings.NewReader(string(input)))
	r.SetPathValue("id", s.ID)
	r.Header.Set("Origin", "http://fixture")
	r.Header.Set("X-CSRF-Token", csrf)
	r.Header.Set("X-Workspace-Expected-Account", uid)
	r.AddCookie(&http.Cookie{Name: "myai_session", Value: token})
	w := httptest.NewRecorder()
	h.Require(a.turn)(w, r)
	if w.Code != 200 || modelCalls != 2 || !prepared || !synced {
		t.Fatal(w.Code, modelCalls, prepared, synced, w.Body.String())
	}
	stored, err := a.read(userRoot, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Mode != "code" || stored.Project != projectID || stored.Policy != "workspace-write" || len(stored.Messages[0].Attachments) != 1 || len(stored.Messages[0].Content) > 100 {
		t.Fatal("Turn not converted or saved as file reference")
	}
	saved, _ := os.ReadFile(filepath.Join(projectPath, "index.html"))
	if string(saved) != string(content) {
		t.Fatal("Exact file was not saved")
	}
}
