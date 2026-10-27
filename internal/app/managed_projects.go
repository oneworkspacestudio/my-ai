package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
)

// A stable per-account ID makes creation safe to retry across tabs and restarts.
// The Workspace broker still owns storage, authorization and private Git setup.
func generalProjectID(uid string) string {
	sum := sha256.Sum256([]byte("my-ai/general-project/v1/" + uid))
	return fmt.Sprintf("%x", sum[:24])
}

func (a *App) ensureCodeProject(ctx context.Context, uid string, s *Session) error {
	if s.Mode != "code" || s.Project != "" {
		return nil
	}
	a.projectSetup.Lock()
	defer a.projectSetup.Unlock()
	id := generalProjectID(uid)
	var p Project
	// Reusing General must not sync another conversation's unfinished edits.
	if e := a.projectBroker(ctx, uid, "GET", "/"+id, nil, &p); e != nil || p.Commit == "" || p.State == "creating" || p.State == "setup-needed" {
		if e := a.projectBroker(ctx, uid, "POST", "", map[string]string{"id": id, "name": "General"}, &p); e != nil {
			return fmt.Errorf("General project could not be prepared. Your message has not been sent. %w", e)
		}
	}
	if p.ID != id || !p.Managed || p.Commit == "" || p.State == "creating" || p.State == "setup-needed" {
		return errors.New("General project setup is incomplete. Your message has not been sent; try again shortly")
	}
	s.Project = p.ID
	s.Thread = ""
	return nil
}

func (a *App) projectBroker(ctx context.Context, uid, method, path string, body any, out any) error {
	endpoint := os.Getenv("MYAI_PROJECTS_URL")
	if endpoint == "" {
		return errors.New("Projects needs My AI installed with the latest Workspace, My Drive and Git")
	}
	u, e := url.Parse(endpoint)
	if e != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.User != nil || u.Path != "/_integration/myai/projects" || u.RawQuery != "" {
		return errors.New("Projects connection needs a Workspace update")
	}
	var subject, issuer, name string
	if a.H.DB.QueryRow(`SELECT subject,issuer,name FROM host_users WHERE id=? AND role!='disabled'`, uid).Scan(&subject, &issuer, &name) != nil || issuer != a.H.Config.Issuer || apphost.Identity(issuer, subject) != uid {
		return errors.New("Sign in again to open your projects")
	}
	var reader io.Reader
	if body != nil {
		raw, e := json.Marshal(body)
		if e != nil {
			return e
		}
		reader = bytes.NewReader(raw)
	}
	req, e := http.NewRequestWithContext(ctx, method, endpoint+path, reader)
	if e != nil {
		return e
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("MYAI_STORAGE_GUARD_TOKEN"))
	req.Header.Set("X-MyAI-Subject", subject)
	req.Header.Set("X-MyAI-Issuer", issuer)
	req.Header.Set("X-MyAI-Name", name)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 150 * time.Second, Transport: &http.Transport{Proxy: nil, IdleConnTimeout: 10 * time.Second}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	resp, e := client.Do(req)
	if e != nil {
		return errors.New("My Drive or Git is unavailable. Your project files are safe; try again shortly")
	}
	defer resp.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if e != nil {
		return e
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var problem struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &problem)
		if problem.Error != "" {
			return errors.New(first(problem.Error, 300))
		}
		return errors.New("Projects needs the latest Workspace update or a running My Drive and Git")
	}
	if out != nil && json.Unmarshal(raw, out) != nil {
		return errors.New("Projects returned an unreadable result")
	}
	return nil
}
func publicProjects(ps []Project) []Project {
	out := make([]Project, 0, len(ps))
	for _, p := range ps {
		p.Path = ""
		p.GitDir = ""
		out = append(out, p)
	}
	return out
}
func (a *App) loadProjects(ctx context.Context, uid string) ([]Project, error) {
	ps := []Project{}
	if os.Getenv("MYAI_PROJECTS_URL") != "" {
		if e := a.projectBroker(ctx, uid, "GET", "", nil, &ps); e != nil {
			return nil, e
		}
	}
	for _, legacy := range a.allProjects(uid) {
		found := false
		for _, p := range ps {
			found = found || p.ID == legacy.ID
		}
		if !found {
			legacy.State = "existing-folder"
			ps = append(ps, legacy)
		}
	}
	return ps, nil
}
func (a *App) managedProject(ctx context.Context, uid, id string) (Project, error) {
	var p Project
	if !idRE.MatchString(id) {
		return p, errors.New("Project not found")
	}
	if e := a.projectBroker(ctx, uid, "GET", "/"+id, nil, &p); e != nil {
		return p, e
	}
	root := os.Getenv("MYAI_PROJECTS_ROOT")
	want := filepath.Join(root, "users", uid, p.Folder)
	if root == "" || !p.Managed || p.ID != id || p.Path == "" || p.GitDir == "" || !strings.EqualFold(filepath.Clean(p.Path), filepath.Clean(want)) {
		return Project{}, errors.New("The project's private storage needs attention")
	}
	if _, e := safePath(p.Path); e != nil {
		return Project{}, e
	}
	return p, nil
}
func (a *App) projectAction(w http.ResponseWriter, r *http.Request) {
	uid := apphost.Current(r).User.ID
	id := r.PathValue("id")
	if !idRE.MatchString(id) {
		apphost.Problem(w, 404, "Project not found")
		return
	}
	a.mu.Lock()
	working := a.projectTurns[uid+":"+id]
	a.mu.Unlock()
	if working {
		apphost.Problem(w, 409, "My AI is still working on this project. Stop or finish that conversation first")
		return
	}
	var p Project
	if e := a.projectBroker(r.Context(), uid, "POST", "/"+id+"/sync", map[string]string{}, &p); e != nil {
		apphost.Problem(w, 409, e.Error())
		return
	}
	apphost.JSON(w, 200, publicProjects([]Project{p})[0])
}
func projectRelative(root, rel string) (string, error) {
	if rel == "" || strings.HasPrefix(rel, "/") || filepath.IsAbs(rel) || strings.ContainsAny(rel, ":\\\x00") || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", errors.New("Use a file inside the selected project")
	}
	for _, part := range strings.Split(rel, "/") {
		l := strings.ToLower(part)
		if part == "" || part != "." && strings.TrimRight(part, ". ") != part {
			return "", errors.New("Use a regular project file name")
		}
		base := strings.SplitN(l, ".", 2)[0]
		if base == "con" || base == "prn" || base == "aux" || base == "nul" || len(base) == 4 && (strings.HasPrefix(base, "com") || strings.HasPrefix(base, "lpt")) && base[3] >= '1' && base[3] <= '9' {
			return "", errors.New("Device names are not project files")
		}

		if l == ".git" || l == ".ssh" || l == ".codex" || l == ".agents" || strings.HasPrefix(l, ".env") && l != ".env.example" || strings.Contains(l, "credential") || strings.HasSuffix(l, ".pem") || strings.HasSuffix(l, ".key") || strings.HasSuffix(l, ".dpapi") || l == "id_rsa" || l == "id_ed25519" {
			return "", errors.New("Credential and Git configuration files are outside AI file access")
		}
	}
	return safePath(root, filepath.FromSlash(rel))
}
func (a *App) editProjectFile(ctx context.Context, scope toolScope, name string, args map[string]string) (any, error) {
	if scope.Project == "" || !scope.Writable {
		return nil, errors.New("Allow changes in the selected project before editing files")
	}
	p, e := a.managedProject(ctx, scope.UID, scope.Project)
	if e != nil {
		return nil, e
	}
	path, e := projectRelative(p.Path, args["path"])
	if e != nil {
		return nil, e
	}
	if strings.EqualFold(filepath.Clean(path), filepath.Clean(p.Path)) {
		return nil, errors.New("Choose a project file, not the project folder")
	}
	if f, e := os.Stat(path); e == nil && f.IsDir() {
		return nil, errors.New("Choose a file, not an existing folder")
	}

	content := args["content"]
	a.projectIO.Lock()
	defer a.projectIO.Unlock()
	if name == "edit_project_file" {
		b, e := os.ReadFile(path)
		if e != nil {
			return nil, errors.New("The project file could not be opened")
		}
		if len(b) > 256*1024 || !utf8.Valid(b) || args["find"] == "" || strings.Count(string(b), args["find"]) != 1 {
			return nil, errors.New("The text to replace must match exactly once in a UTF-8 file up to256 KB")
		}
		content = strings.Replace(string(b), args["find"], args["replace"], 1)
	}
	if len(content) > 256*1024 || !utf8.ValidString(content) || strings.ContainsRune(content, 0) {
		return nil, errors.New("Write one UTF-8 project file up to256 KB")
	}
	if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return nil, e
	}
	if _, e = projectRelative(p.Path, args["path"]); e != nil {
		return nil, e
	}
	temp := path + ".myai-" + apphost.ID()
	f, e := os.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return nil, errors.New("My Drive could not save the file")
	}
	defer os.Remove(temp)
	_, e = f.WriteString(content)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e == nil {
		e = os.Rename(temp, path)
	}
	if e != nil {
		return nil, errors.New("My Drive could not finish saving the file")
	}
	return map[string]any{"path": args["path"], "bytes": len(content), "saved_in": "My Drive / Projects / " + p.Name, "git": "Changes will be committed and pushed when this turn completes"}, nil
}
func (a *App) projectCode(ctx context.Context, uid, root string, s *Session, p Provider, send writer) (string, error) {
	project, e := a.managedProject(ctx, uid, s.Project)
	if e != nil {
		return "", e
	}
	var prepared Project
	if e = a.projectBroker(ctx, uid, "POST", "/"+project.ID+"/prepare", map[string]string{}, &prepared); e != nil {
		return "", e
	}
	scope := toolScope{UID: uid, Root: root, Project: project.ID, Writable: s.Policy == "workspace-write"}
	out, e := a.chatWithTools(ctx, p, s, send, scope)
	if e != nil {
		return out, e
	}
	if !scope.Writable {
		return out, nil
	}
	_, e = a.saveProjectCheckpoint(ctx, scope, s, send)
	return out, e
}

// The same account/project authorization used for the normal end-of-turn save
// also allows an explicit checkpoint while building and deploying in one turn.
func (a *App) saveProjectCheckpoint(ctx context.Context, scope toolScope, s *Session, send writer) (Project, error) {
	var synced Project
	if scope.Project == "" || !scope.Writable {
		return synced, errors.New("Allow changes in the selected project before saving")
	}
	send("event", map[string]any{"method": "project/saving", "params": map[string]string{"message": "Saving your project to Git…"}})
	if e := a.projectBroker(ctx, scope.UID, "POST", "/"+scope.Project+"/sync", map[string]string{}, &synced); e != nil {
		return synced, fmt.Errorf("Your project files are saved in My Drive, but Git has not confirmed the save. Open Projects and use Sync to retry. %s", e)
	}
	data, _ := json.Marshal(map[string]string{"repository": synced.URL, "commit": synced.Commit, "message": "Saved to Git"})
	s.Events = append(s.Events, Event{At: now(), Method: "project/saved", Data: data})
	send("event", map[string]any{"method": "project/saved", "params": map[string]string{"repository": synced.URL, "commit": synced.Commit, "message": "Saved to Git"}})
	return synced, nil
}
