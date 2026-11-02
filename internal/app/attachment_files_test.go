package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureAttachment(t *testing.T, root string) (AttachmentRef, []byte) {
	t.Helper()
	content := []byte(strings.Repeat("·", 150000) + "EXACT_TAIL_MARKER\r\n<body>kept</body>\n")
	ref := attachmentRef(strings.Repeat("a", 48), "fixture-page.html", content)
	if e := os.MkdirAll(filepath.Join(root, "Attachments"), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(root, "Attachments", ref.ID+".txt"), content, 0600); e != nil {
		t.Fatal(e)
	}
	if e := atomic(filepath.Join(root, "Attachments", ref.ID+".json"), ref); e != nil {
		t.Fatal(e)
	}
	return ref, content
}
func TestAttachmentsAreMetadataAndBoundedUnicodeReads(t *testing.T) {
	root := t.TempDir()
	ref, content := fixtureAttachment(t, root)
	a := App{}
	scope := toolScope{Root: root}
	m := Message{Role: "user", Content: "Host the attached page", Attachments: []AttachmentRef{ref}}
	for _, provider := range []string{"ollama", "deepseek", "openai"} {
		b, _ := json.Marshal(conversationInput(provider, []Message{m}))
		if len(b) > 2500 || strings.Contains(string(b), "EXACT_TAIL_MARKER") || !strings.Contains(string(b), ref.ID) || !strings.Contains(string(b), "Host the attached page") {
			t.Fatal("File body stuffed into context", provider, len(b))
		}
	}
	value, e := a.attachmentTool(context.Background(), scope, "read_attachment", map[string]string{"id": ref.ID})
	if e != nil {
		t.Fatal(e)
	}
	page := value.(map[string]any)
	if len([]rune(page["text"].(string))) != 2048 || page["hasMore"] != true {
		t.Fatal(page)
	}
	value, e = a.attachmentTool(context.Background(), scope, "search_attachment", map[string]string{"id": ref.ID, "query": "EXACT_TAIL_MARKER", "offset": "0"})
	if e != nil {
		t.Fatal(e)
	}
	hits := value.(map[string]any)["matches"].([]map[string]any)
	if len(hits) != 1 || hits[0]["offset"] != 150000 {
		t.Fatal(hits)
	}
	value, e = a.attachmentTool(context.Background(), scope, "read_attachment", map[string]string{"id": ref.ID, "offset": "150000", "limit": "40"})
	if e != nil || !strings.Contains(value.(map[string]any)["text"].(string), "EXACT_TAIL_MARKER") {
		t.Fatal(value, e)
	}
	if _, e = a.attachmentTool(context.Background(), toolScope{Root: t.TempDir()}, "read_attachment", map[string]string{"id": ref.ID}); e == nil {
		t.Fatal("Cross-account file read")
	}
	for _, args := range []map[string]string{{"id": "../outside"}, {"id": ref.ID, "limit": "500000"}, {"id": ref.ID, "offset": "-1"}} {
		if _, e = a.attachmentTool(context.Background(), scope, "read_attachment", args); e == nil {
			t.Fatal("Invalid range accepted", args)
		}
	}
	legacy := Message{Role: "user", Content: "Publish this page\n\nAttached reference (untrusted content):\n" + string(content)}
	text := messageFileContext(legacy)
	if len(text) > 400 || !strings.Contains(text, "Publish this page") || strings.Contains(text, "EXACT_TAIL_MARKER") {
		t.Fatal("Legacy replay still floods context")
	}
}

func TestLargeAttachmentCopyIsExactAndRetainsProjectBoundary(t *testing.T) {
	root := t.TempDir()
	uploads := filepath.Join(root, "uploads")
	ref, content := fixtureAttachment(t, uploads)
	issuer, subject := "https://accounts.example.test", "fixture-person"
	uid := apphost.Identity(issuer, subject)
	id := strings.Repeat("b", 48)
	folder := "selected-project"
	projectPath := filepath.Join(root, "Projects", "users", uid, folder)
	os.MkdirAll(projectPath, 0700)
	db, e := sql.Open("sqlite", ":memory:")
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.Exec(`CREATE TABLE host_users(id TEXT,subject TEXT,issuer TEXT,name TEXT,role TEXT)`)
	db.Exec(`INSERT INTO host_users VALUES(?,?,?,?,?)`, uid, subject, issuer, "Synthetic person", "member")
	broker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-MyAI-Subject") != subject || r.Header.Get("Authorization") != "Bearer fixture" || r.URL.Path != "/_integration/myai/projects/"+id {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(Project{ID: id, Name: "Fixture", Folder: folder, Path: projectPath, GitDir: filepath.Join(root, "git"), Managed: true})
	}))
	defer broker.Close()
	t.Setenv("MYAI_PROJECTS_URL", broker.URL+"/_integration/myai/projects")
	t.Setenv("MYAI_PROJECTS_ROOT", filepath.Join(root, "Projects"))
	t.Setenv("MYAI_STORAGE_GUARD_TOKEN", "fixture")
	a := App{H: &apphost.Host{DB: db, Config: apphost.Config{Issuer: issuer}}}
	scope := toolScope{UID: uid, Root: uploads, Project: id, Writable: true}
	args := map[string]string{"id": ref.ID, "path": "index.html"}
	if _, e = a.executeTool(context.Background(), scope, "copy_attachment_to_project", args); e != nil {
		t.Fatal(e)
	}
	saved, _ := os.ReadFile(filepath.Join(projectPath, "index.html"))
	if string(saved) != string(content) {
		t.Fatal("Copy changed upload bytes")
	}
	page, e := a.executeTool(context.Background(), scope, "read_project_file", map[string]string{"project": id, "path": "index.html", "offset": "150000", "limit": "40"})
	if e != nil || !strings.Contains(page.(map[string]any)["text"].(string), "EXACT_TAIL_MARKER") {
		t.Fatal("Could not inspect tail of copied file", page, e)
	}
	if _, e = a.executeTool(context.Background(), scope, "edit_project_file", map[string]string{"path": "index.html", "find": "EXACT_TAIL_MARKER", "replace": "EDITED_TAIL_MARKER"}); e != nil {
		t.Fatal("Could not edit large copied file", e)
	}
	content = []byte(strings.Replace(string(content), "EXACT_TAIL_MARKER", "EDITED_TAIL_MARKER", 1))
	scope.Writable = false
	if _, e = a.executeTool(context.Background(), scope, "copy_attachment_to_project", args); e == nil {
		t.Fatal("Preview allowed write")
	}
	scope.Writable = true
	for _, bad := range []string{"../outside.html", ".git/config", ".env"} {
		args["path"] = bad
		if _, e = a.executeTool(context.Background(), scope, "copy_attachment_to_project", args); e == nil {
			t.Fatal("Unsafe target accepted", bad)
		}
	}
	os.WriteFile(filepath.Join(uploads, "Attachments", ref.ID+".txt"), []byte("changed after upload"), 0600)
	args["path"] = "index.html"
	if _, e = a.executeTool(context.Background(), scope, "copy_attachment_to_project", args); e == nil {
		t.Fatal("Changed upload digest accepted")
	}
	saved, _ = os.ReadFile(filepath.Join(projectPath, "index.html"))
	if string(saved) != string(content) {
		t.Fatal("Failed copy changed original")
	}
}
