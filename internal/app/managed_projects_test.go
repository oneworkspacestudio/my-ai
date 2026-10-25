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

func TestManagedProjectFileBoundary(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"../outside.txt", "/absolute", "C:/Windows/file", "a\\b.txt", ".git/config", "folder/.env", "id_rsa", "cert.key", "credential-store.json"} {
		if _, e := projectRelative(root, rel); e == nil {
			t.Errorf("accepted %q", rel)
		}
	}
	for _, rel := range []string{"README.md", "src/main.ts", ".env.example", "."} {
		if _, e := projectRelative(root, rel); e != nil {
			t.Errorf("rejected %q: %v", rel, e)
		}
	}
	outside := t.TempDir()
	if e := os.Symlink(outside, filepath.Join(root, "linked")); e == nil {
		if _, e = projectRelative(root, "linked/private.txt"); e == nil {
			t.Fatal("linked path accepted")
		}
	}
}
func TestProjectToolsNeedSelectedWritePermission(t *testing.T) {
	names := func(scope toolScope) map[string]bool {
		out := map[string]bool{}
		for _, v := range toolSpecs(scope) {
			out[v.Name] = true
		}
		return out
	}
	for _, s := range []toolScope{{UID: "person"}, {UID: "person", Project: "project"}} {
		n := names(s)
		if n["write_project_file"] || n["edit_project_file"] {
			t.Fatal("edits exposed without permission")
		}
		if !n["read_project_file"] {
			t.Fatal("own reads unavailable")
		}
	}
	if !names(toolScope{UID: "person", Project: "project", Writable: true})["write_project_file"] {
		t.Fatal("selected editing unavailable")
	}
}
func TestPublicProjectMetadataHasNoHostPaths(t *testing.T) {
	p := publicProjects([]Project{{ID: "project", Path: "private-root", GitDir: "protected-git-metadata", Managed: true, URL: "https://example.test/git/user/project"}})[0]
	if p.Path != "" || p.GitDir != "" || p.URL == "" {
		t.Fatal(p)
	}
}

func TestManagedEditsAreAtomicAndStayInSelectedFiles(t *testing.T) {
	root := t.TempDir()
	issuer, subject := "https://accounts.example.test", "one-person"
	uid := apphost.Identity(issuer, subject)
	id := strings.Repeat("b", 48)
	folder := "selected-project"
	path := filepath.Join(root, "Projects", "users", uid, folder)
	if e := os.MkdirAll(path, 0700); e != nil {
		t.Fatal(e)
	}
	db, e := sql.Open("sqlite", ":memory:")
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, e = db.Exec(`CREATE TABLE host_users(id TEXT,subject TEXT,issuer TEXT,name TEXT,role TEXT)`); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`INSERT INTO host_users VALUES(?,?,?,?,?)`, uid, subject, issuer, "One person", "member"); e != nil {
		t.Fatal(e)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-MyAI-Subject") != subject || r.Header.Get("Authorization") != "Bearer fixture-capability" || r.URL.Path != "/_integration/myai/projects/"+id {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(Project{ID: id, Name: "Selected project", Folder: folder, Path: path, GitDir: filepath.Join(root, "protected-git"), Managed: true})
	}))
	defer srv.Close()
	t.Setenv("MYAI_PROJECTS_URL", srv.URL+"/_integration/myai/projects")
	t.Setenv("MYAI_PROJECTS_ROOT", filepath.Join(root, "Projects"))
	t.Setenv("MYAI_STORAGE_GUARD_TOKEN", "fixture-capability")
	a := App{H: &apphost.Host{DB: db, Config: apphost.Config{Issuer: issuer}}}
	scope := toolScope{UID: uid, Project: id, Writable: true}
	if _, e = a.editProjectFile(context.Background(), scope, "write_project_file", map[string]string{"path": "README.md", "content": "before"}); e != nil {
		t.Fatal(e)
	}
	if _, e = a.editProjectFile(context.Background(), scope, "write_project_file", map[string]string{"path": "README.md", "content": "after"}); e != nil {
		t.Fatal(e)
	}
	if b, _ := os.ReadFile(filepath.Join(path, "README.md")); string(b) != "after" {
		t.Fatal("atomic replacement failed")
	}
	if _, e = a.editProjectFile(context.Background(), scope, "edit_project_file", map[string]string{"path": "README.md", "find": "after", "replace": "edited"}); e != nil {
		t.Fatal(e)
	}
	for _, rel := range []string{".", "nested/..", "../outside.txt", ".git/config"} {
		if _, e = a.editProjectFile(context.Background(), scope, "write_project_file", map[string]string{"path": rel, "content": "must not write"}); e == nil {
			t.Fatal("unsafe file target accepted", rel)
		}
	}
	scope.Writable = false
	if _, e = a.editProjectFile(context.Background(), scope, "write_project_file", map[string]string{"path": "README.md", "content": "not allowed"}); e == nil {
		t.Fatal("preview mode wrote a file")
	}
	if b, _ := os.ReadFile(filepath.Join(path, "README.md")); string(b) != "edited" {
		t.Fatal("failed edits changed original file")
	}
}
