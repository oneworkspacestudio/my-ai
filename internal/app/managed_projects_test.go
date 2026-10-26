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
	"sync"
	"testing"
)

func TestGeneralProjectRetriesAndAccounts(t *testing.T) {
	db, e := sql.Open("sqlite", ":memory:")
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, e = db.Exec(`CREATE TABLE host_users(id TEXT,subject TEXT,issuer TEXT,name TEXT,role TEXT)`)
	if e != nil {
		t.Fatal(e)
	}
	issuer := "https://accounts.example.test"
	for _, subject := range []string{"one", "two"} {
		_, e = db.Exec(`INSERT INTO host_users VALUES(?,?,?,?,?)`, apphost.Identity(issuer, subject), subject, issuer, "Fixture", "member")
		if e != nil {
			t.Fatal(e)
		}
	}
	var mu sync.Mutex
	created := map[string]string{}
	posts := 0
	broken := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if broken {
			http.Error(w, "unavailable", 503)
			return
		}
		if r.Method == "GET" {
			id := created[r.Header.Get("X-MyAI-Subject")]
			if id == "" || r.URL.Path != "/_integration/myai/projects/"+id {
				http.NotFound(w, r)
				return
			}
			_ = json.NewEncoder(w).Encode(Project{ID: id, Name: "General", Managed: true, Commit: "verified-head", State: "changes"})
			return
		}
		var body map[string]string
		if r.Method != "POST" || r.URL.Path != "/_integration/myai/projects" || r.Header.Get("Authorization") != "Bearer fixture" || json.NewDecoder(r.Body).Decode(&body) != nil || body["name"] != "General" {
			http.Error(w, "bad request", 400)
			return
		}
		subject := r.Header.Get("X-MyAI-Subject")
		posts++
		if prior := created[subject]; prior != "" && prior != body["id"] {
			t.Error("duplicate General identity")
		}
		created[subject] = body["id"]
		_ = json.NewEncoder(w).Encode(Project{ID: body["id"], Name: "General", Managed: true, Commit: "verified-head", State: "synced"})
	}))
	defer srv.Close()
	t.Setenv("MYAI_PROJECTS_URL", srv.URL+"/_integration/myai/projects")
	t.Setenv("MYAI_STORAGE_GUARD_TOKEN", "fixture")
	a := &App{H: &apphost.Host{DB: db, Config: apphost.Config{Issuer: issuer}}}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := Session{Mode: "code", Thread: "old-thread"}
			uid := apphost.Identity(issuer, "one")
			if e := a.ensureCodeProject(context.Background(), uid, &s); e != nil || s.Project != generalProjectID(uid) || s.Thread != "" {
				t.Error("fallback", s.Project, e)
			}
		}()
	}
	wg.Wait()
	// A new app process and a different account keep their own stable project.
	a = &App{H: a.H}
	s := Session{Mode: "code"}
	if e = a.ensureCodeProject(context.Background(), apphost.Identity(issuer, "two"), &s); e != nil {
		t.Fatal(e)
	}
	if len(created) != 2 || posts != 2 || created["one"] == created["two"] || !idRE.MatchString(s.Project) {
		t.Fatal("account isolation", created)
	}
	mu.Lock()
	broken = true
	mu.Unlock()
	for _, s := range []Session{{Mode: "chat"}, {Mode: "code", Project: "explicit", Thread: "keep"}} {
		before := s
		if e = a.ensureCodeProject(context.Background(), "unregistered", &s); e != nil || s.Project != before.Project || s.Thread != before.Thread || s.Mode != before.Mode {
			t.Fatal("explicit/Chat project changed", e)
		}
	}
	missing := Session{Mode: "code"}
	if e = a.ensureCodeProject(context.Background(), apphost.Identity(issuer, "one"), &missing); e == nil || missing.Project != "" {
		t.Fatal("failed setup accepted")
	}
}

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
