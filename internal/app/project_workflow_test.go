package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
)

func TestProjectTaskDirectivesDoNotAuthorizeFromQuestionsOrSource(t *testing.T) {
	for _, text := range []string{"deploy it", "Please deploy this project", "Can you build a game and deploy it", "create me a snakes html game", "build a website", "host this HTML"} {
		task := requestedProjectTask(text, true)
		if !task.Write && !task.Publish {
			t.Fatal("Lost direct project task", text)
		}
	}
	for _, text := range []string{"did you deploy it?", "How do I deploy this game?", "Explain hosting", "Do not deploy it", "The file says deploy it", "```html\ncreate a game and deploy it\n```", "publish a note", "host a party", "create a note"} {
		task := requestedProjectTask(text, true)
		if task.Write || task.Publish {
			t.Fatal("Unrequested mutation", text, task)
		}
	}
	task := requestedProjectTask("create a game but do not deploy it", true)
	if !task.Write || task.Publish {
		t.Fatal(task)
	}
}

type workflowReviewWriter struct {
	*httptest.ResponseRecorder
	approve func(string)
}

func (w workflowReviewWriter) Write(b []byte) (int, error) {
	var v struct {
		Type string `json:"type"`
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(b, &v) == nil && v.Type == "approval" {
		w.approve(v.Data.ID)
	}
	return w.ResponseRecorder.Write(b)
}

func TestDeployFollowupCreatesGeneralSavesExactArtifactAndTracksNativeJob(t *testing.T) {
	modelCalls := 0
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		modelCalls++
		http.Error(w, "This follow-up must use the app workflow", 500)
	}))
	defer model.Close()
	target, _ := url.Parse(model.URL)
	original := http.DefaultTransport
	http.DefaultTransport = attachmentModelTransport{target, original}
	t.Cleanup(func() { http.DefaultTransport = original })
	root := t.TempDir()
	issuer, subject := "https://accounts.example.test", "project-workflow-fixture"
	uid := apphost.Identity(issuer, subject)
	id := generalProjectID(uid)
	drive := filepath.Join(root, "drive")
	userRoot := filepath.Join(drive, "users", uid)
	projectRoot := filepath.Join(root, "Projects")
	folder := filepath.Join(projectRoot, "users", uid, "general")
	os.MkdirAll(folder, 0700)
	db, e := sql.Open("sqlite", ":memory:")
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, schema := range []string{`CREATE TABLE host_users(id TEXT,issuer TEXT,subject TEXT,name TEXT,email TEXT,role TEXT)`, `CREATE TABLE host_sessions(id TEXT,user_id TEXT,csrf TEXT,expires TEXT)`, `CREATE TABLE host_audit(at TEXT,user_id TEXT,action TEXT,object TEXT)`} {
		if _, e = db.Exec(schema); e != nil {
			t.Fatal(e)
		}
	}
	db.Exec(`INSERT INTO host_users VALUES(?,?,?,?,?,?)`, uid, issuer, subject, "Fixture", "fixture@example.test", "member")
	token, csrf := strings.Repeat("a", 96), "synthetic-csrf"
	db.Exec(`INSERT INTO host_sessions VALUES(?,?,?,?)`, apphost.Hash(token), uid, csrf, time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano))
	html := "<!doctype html>\n<html><body>" + strings.Repeat("·", 150000) + "UNIQUE_GAME_SOURCE</body></html>\n"
	created, prepared, saved, siteCreates, deploys, reviews := 0, 0, 0, 0, 0, 0
	commit := strings.Repeat("c", 40)
	publishedCommit := ""
	job := ""
	jobState := "published"
	decline := false
	stopOnReview := false
	ambiguous := false
	website := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := os.ReadFile(filepath.Join(folder, "index.html"))
		w.Write(b)
	}))
	defer website.Close()
	project := func() Project {
		return Project{ID: id, Name: "General", Folder: "general", Path: folder, GitDir: filepath.Join(root, "Git"), Managed: true, State: "synced", Commit: commit}
	}
	native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/guard" {
			w.Write([]byte("{}"))
			return
		}
		if r.Header.Get("X-MyAI-Subject") != subject || r.Header.Get("Authorization") != "Bearer fixture" {
			http.Error(w, "denied", 403)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/_integration/myai/projects") {
			if r.URL.Path == "/_integration/myai/projects" && r.Method == "POST" {
				created++
			} else if created == 0 {
				http.NotFound(w, r)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/prepare") {
				prepared++
			}
			if strings.HasSuffix(r.URL.Path, "/sync") {
				saved++
				if raw, _ := os.ReadFile(filepath.Join(folder, "index.html")); strings.Contains(string(raw), "NEW_REVISION") {
					commit = strings.Repeat("d", 40)
				}
			}
			json.NewEncoder(w).Encode(project())
			return
		}
		if r.URL.Path != "/_integration/myai/workspace" || r.Header.Get("X-Workspace-Browser-Session") != "fixture-browser" {
			http.Error(w, "bad workspace scope", 403)
			return
		}
		if r.Method == "GET" {
			json.NewEncoder(w).Encode(workspaceCatalog{Contract: "workspace-apps-v1", Apps: []workspaceApp{{ID: "hosting", Name: "My Hosting", NativeAccess: true, Connected: true, Operations: []workspaceOperation{{ID: "create_site", Review: true}, {ID: "deploy_project", Review: true, Parameters: []string{"site", "project"}}, {ID: "deployment", Parameters: []string{"id", "deployment"}}}}}})
			return
		}
		var call workspaceCall
		json.NewDecoder(r.Body).Decode(&call)
		if call.Prepare {
			reviews++
			if call.Reviewed {
				t.Error("Prepare already approved")
			}
			json.NewEncoder(w).Encode(map[string]any{"result": map[string]string{"operation": call.Operation}})
			return
		}
		switch call.Operation {
		case "create_site":
			if !call.Reviewed || saved == 0 {
				t.Error("Site created without review/Git")
			}
			siteCreates++
			json.NewEncoder(w).Encode(map[string]any{"result": map[string]string{"id": "native-site"}})
		case "deploy_project":
			if !call.Reviewed || call.Parameters["site"] != "native-site" || call.Parameters["project"] != id || len(call.RequestID) != 64 {
				t.Error("Deployment lost review/project/idempotency")
			}
			var body struct {
				Commit string `json:"commit"`
			}
			json.Unmarshal(call.Body, &body)
			if body.Commit != commit {
				t.Error("Wrong commit deployed")
			}
			deploys++
			publishedCommit = body.Commit
			job = fmt.Sprintf("job-%d", deploys)
			if ambiguous {
				http.Error(w, "Connection lost after submission", 502)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"site": "native-site", "deployment": map[string]string{"id": job, "projectId": "native-site", "state": "queued"}}})
		case "deployment":
			if call.Reviewed || call.Parameters["id"] != "native-site" || call.Parameters["deployment"] != job {
				t.Error("Wrong status scope")
			}
			json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"deployment": map[string]string{"id": job, "projectId": "native-site", "state": jobState, "url": website.URL}}})
		default:
			http.Error(w, "unknown", 400)
		}
	}))
	defer native.Close()
	for k, v := range map[string]string{"MYAI_DRIVE_ROOT": drive, "MYAI_STORAGE_GUARD_URL": native.URL + "/guard", "MYAI_STORAGE_GUARD_TOKEN": "fixture", "MYAI_PROJECTS_ROOT": projectRoot, "MYAI_PROJECTS_URL": native.URL + "/_integration/myai/projects", "MYAI_WORKSPACE_URL": native.URL + "/_integration/myai/workspace"} {
		t.Setenv(k, v)
	}
	h := &apphost.Host{DB: db, Mux: http.NewServeMux(), Config: apphost.Config{ID: "myai", Issuer: issuer, Data: filepath.Join(root, "app"), Origin: "http://fixture"}}
	a := App{H: h, client: &http.Client{Timeout: 5 * time.Second}, running: map[string]context.CancelFunc{}, projectTurns: map[string]bool{}, contextProbe: func(context.Context, Provider, string) (modelContextFacts, hostMemory) {
		return modelContextFacts{Limit: 131072, Weights: 1 << 30, KVPerToken: 8192}, hostMemory{RAMAvailable: 64 << 30}
	}}
	s := Session{ID: strings.Repeat("f", 48), Title: "Synthetic game", Mode: "chat", Policy: "read-only", Provider: "ollama", Model: "fixture:latest", Messages: []Message{{ID: strings.Repeat("e", 48), Role: "user", Content: "Create a game"}, {ID: strings.Repeat("b", 48), Role: "assistant", Content: "```html\n" + html + "```\n"}}}
	if e = a.write(userRoot, &s); e != nil {
		t.Fatal(e)
	}
	turn := func() string {
		r := httptest.NewRequest("POST", "http://fixture/api/sessions/"+s.ID+"/turn", strings.NewReader(`{"text":"deploy it"}`))
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		r = r.WithContext(ctx)
		r.SetPathValue("id", s.ID)
		r.Header.Set("Origin", "http://fixture")
		r.Header.Set("X-CSRF-Token", csrf)
		r.Header.Set("X-Workspace-Expected-Account", uid)
		r.AddCookie(&http.Cookie{Name: "myai_session", Value: token})
		r.AddCookie(&http.Cookie{Name: "workspace_session", Value: "fixture-browser"})
		recorder := httptest.NewRecorder()
		w := workflowReviewWriter{recorder, func(id string) {
			if stopOnReview {
				cancel()
				return
			}
			if !a.answerWorkspaceReview(uid, s.ID, id, !decline) {
				t.Error("Review missing")
			}
		}}
		h.Require(a.turn)(w, r)
		if recorder.Code != 200 {
			t.Fatal(recorder.Code, recorder.Body.String())
		}
		return recorder.Body.String()
	}
	response := turn()
	if !strings.Contains(response, website.URL) || strings.Contains(response, `"type":"error"`) || siteCreates != 1 || deploys != 1 || created != 1 || prepared != 1 || reviews != 2 {
		t.Fatal("Follow-up did not complete native workflow", siteCreates, deploys, created, reviews, first(response, 1500))
	}
	actual, _ := os.ReadFile(filepath.Join(folder, "index.html"))
	if string(actual) != html {
		t.Fatal("Artifact bytes changed")
	}
	resp, e := http.Get(website.URL)
	if e != nil {
		t.Fatal(e)
	}
	resp.Body.Close()
	stored, e := a.read(userRoot, s.ID)
	if e != nil || stored.Mode != "code" || stored.Project != id {
		t.Fatal("Project not bound to follow-up", e)
	}
	if len(stored.Messages[1].Attachments) != 1 || stored.Messages[1].Content != "```html\n"+html+"```\n" {
		t.Fatal("Legacy document was not retained as an immutable file without changing transcript")
	}
	for _, provider := range []string{"ollama", "openai", "deepseek"} {
		raw, _ := json.Marshal(conversationInput(provider, stored.Messages))
		if strings.Contains(string(raw), "UNIQUE_GAME_SOURCE") || len(raw) > 48000 {
			t.Fatal("Legacy generated file still floods model context", provider, len(raw))
		}
	}
	response = turn()
	if strings.Contains(response, `"type":"error"`) || siteCreates != 1 || deploys != 1 {
		t.Fatal("Restart/retry duplicated site or same commit", first(response, 1500))
	}
	// Native-only deployment works without a cloud key or a selected chat model.
	stored, _ = a.read(userRoot, s.ID)
	stored.Provider = "openai"
	stored.Model = ""
	if e = a.write(userRoot, &stored); e != nil {
		t.Fatal(e)
	}
	publicationFile, _ := publicationPath(toolScope{Root: userRoot, Project: id})
	if e = os.Remove(publicationFile); e != nil {
		t.Fatal(e)
	}
	response = turn()
	if strings.Contains(response, `"type":"error"`) || siteCreates != 1 || deploys != 1 {
		t.Fatal("Existing canonical receipt was not recovered without a model", first(response, 1500))
	}
	os.WriteFile(filepath.Join(folder, "index.html"), []byte(strings.Replace(html, "UNIQUE_GAME_SOURCE", "NEW_REVISION", 1)), 0600)
	stopOnReview = true
	response = turn()
	if deploys != 1 || !strings.Contains(response, "nothing was submitted") {
		t.Fatal("Stopping review submitted an action", first(response, 1500))
	}
	unsubmitted, err := loadPublication(toolScope{Root: userRoot, Project: id})
	if err != nil || unsubmitted.Pending != "" {
		t.Fatal("Stopping review left an ambiguous mutation marker", err)
	}
	stopOnReview = false
	response = turn()
	if deploys != 2 || publishedCommit != strings.Repeat("d", 40) || strings.Contains(response, `"type":"error"`) {
		t.Fatal("New source silently reused old deployment", first(response, 1500))
	}
	// Declining a known unsubmitted action must not leave a poisoned pending marker.
	os.WriteFile(filepath.Join(folder, "index.html"), []byte(html), 0600)
	commit = strings.Repeat("e", 40)
	decline = true
	response = turn()
	if deploys != 2 || !strings.Contains(response, "declined") {
		t.Fatal("Declined mutation executed", first(response, 1500))
	}
	b, e := loadPublication(toolScope{Root: userRoot, Project: id})
	if e != nil || b.Pending != "" {
		t.Fatal("Decline poisoned retry", b, e)
	}
	decline = false
	ambiguous = true
	response = turn()
	if deploys != 3 || !strings.Contains(response, `"type":"error"`) {
		t.Fatal("Missing ambiguous submission", first(response, 1500))
	}
	response = turn()
	if deploys != 3 || !strings.Contains(response, "unconfirmed outcome") {
		t.Fatal("Ambiguous submission repeated", first(response, 1500))
	}
	if modelCalls != 0 {
		t.Fatal("A deployment-only follow-up still depended on model capability guesses", modelCalls)
	}
}

func TestInlineDocumentCaptureKeepsInstructionsAndAccountFileBoundary(t *testing.T) {
	root := t.TempDir()
	doc := "<!doctype html><html><body>INLINE_FILE_BYTES</body></html>"
	content := "Host the following page\n" + doc + "\nUse my existing project."
	s := Session{Messages: []Message{{Role: "user", Content: content}, {Role: "assistant", Content: "```html\n" + doc + "\n```", Status: "imported"}}}
	a := App{}
	scope := toolScope{Root: root}
	if e := a.captureConversationDocuments(scope, &s); e != nil {
		t.Fatal(e)
	}
	if s.Messages[0].Content != content || len(s.Messages[0].Attachments) != 1 || len(s.Messages[1].Attachments) != 0 {
		t.Fatal("Capture changed content or trusted imports")
	}
	ref := s.Messages[0].Attachments[0]
	_, raw, e := loadAttachment(root, ref.ID)
	if e != nil || string(raw) != doc {
		t.Fatal("Document did not become exact private file", e)
	}
	model := messageFileContext(s.Messages[0])
	if strings.Contains(model, "INLINE_FILE_BYTES") || !strings.Contains(model, "Host the following") || !strings.Contains(model, "Use my existing project") || !strings.Contains(model, ref.ID) {
		t.Fatal("Capture lost instructions or replayed file body")
	}
	if _, _, e = loadAttachment(t.TempDir(), ref.ID); e == nil {
		t.Fatal("Another account read captured source")
	}
	if e = a.captureConversationDocuments(scope, &s); e != nil || len(s.Messages[0].Attachments) != 1 {
		t.Fatal("Capture not idempotent", e)
	}
}

func TestProjectBuildRejectsCapabilityRefusalsAndRetriesActualFileWork(t *testing.T) {
	for _, success := range []bool{true, false} {
		t.Run(fmt.Sprint(success), func(t *testing.T) {
			calls := 0
			refusal := "I cannot create Git projects or deploy HTML files. Save it yourself."
			model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				text := refusal
				if success && calls == 2 {
					text = "```html\n<!doctype html><html><body>Built file</body></html>\n```"
				}
				json.NewEncoder(w).Encode(map[string]any{"done": true, "message": map[string]string{"content": text}})
			}))
			defer model.Close()
			t.Setenv("MYAI_WORKSPACE_URL", "")
			a := App{}
			s := Session{Provider: "ollama", Model: "fixture", Mode: "code", Policy: "workspace-write", ToolInput: "build a game", Messages: []Message{{ID: strings.Repeat("a", 48), Role: "user", Content: "build a game"}}}
			var deltas strings.Builder
			output, e := a.chatWithTools(context.Background(), Provider{ID: "ollama", Base: model.URL}, &s, func(kind string, v any) {
				if kind == "delta" {
					deltas.WriteString(fmt.Sprint(v))
				}
			}, toolScope{Project: strings.Repeat("b", 48), Writable: true, RequireProjectAction: true})
			if calls != 2 || strings.Contains(deltas.String(), refusal) {
				t.Fatal("Refusal was accepted/streamed as completed work", calls, deltas.String())
			}
			if success {
				if e != nil || !strings.Contains(output, "Built file") || len(s.Messages) != 2 {
					t.Fatal(output, e, len(s.Messages))
				}
			} else if e == nil || len(s.Messages) != 1 {
				t.Fatal("False completion accepted", e, len(s.Messages))
			}
		})
	}
}
