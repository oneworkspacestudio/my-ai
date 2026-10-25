package app

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

type Project struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path,omitempty"`
	GitDir   string `json:"gitDir,omitempty"`
	Folder   string `json:"folder,omitempty"`
	URL      string `json:"url,omitempty"`
	DriveURL string `json:"driveUrl,omitempty"`
	Branch   string `json:"branch,omitempty"`
	State    string `json:"state,omitempty"`
	Commit   string `json:"commit,omitempty"`
	Error    string `json:"error,omitempty"`
	Managed  bool   `json:"managed"`
}

func (a *App) projectPath(uid string) string {
	return filepath.Join(a.H.Config.Data, "approved-projects", uid+".json")
}
func (a *App) allProjects(uid string) []Project {
	out := []Project{}
	b, _ := os.ReadFile(a.projectPath(uid))
	_ = json.Unmarshal(b, &out)
	return out
}
func (a *App) findProject(uid, id string) (Project, error) {
	if os.Getenv("MYAI_PROJECTS_URL") != "" {
		if p, e := a.managedProject(context.Background(), uid, id); e == nil {
			return p, nil
		}
	}
	for _, p := range a.allProjects(uid) {
		if p.ID == id {
			if _, e := safePath(p.Path); e != nil {
				return Project{}, e
			}
			if f, e := os.Stat(p.Path); e != nil || !f.IsDir() {
				return Project{}, errors.New("Project folder is unavailable")
			}
			return p, nil
		}
	}
	return Project{}, errors.New("Choose one of your projects")
}
func (a *App) projects(w http.ResponseWriter, r *http.Request) {
	ps, e := a.loadProjects(r.Context(), apphost.Current(r).User.ID)
	if e != nil {
		apphost.Problem(w, 503, e.Error())
		return
	}
	apphost.JSON(w, 200, publicProjects(ps))
}
func (a *App) projectSave(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if !apphost.Decode(w, r, &in) {
		return
	}
	if in.ID == "" {
		in.ID = apphost.ID()
	}
	if !idRE.MatchString(in.ID) || strings.TrimSpace(in.Name) == "" {
		apphost.Problem(w, 400, "Give your project a name")
		return
	}
	uid := apphost.Current(r).User.ID
	var p Project
	if e := a.projectBroker(r.Context(), uid, "POST", "", in, &p); e != nil {
		apphost.Problem(w, 503, e.Error())
		return
	}
	a.H.Audit(uid, "project-create", p.ID)
	ps, e := a.loadProjects(r.Context(), uid)
	if e != nil {
		apphost.Problem(w, 503, e.Error())
		return
	}
	apphost.JSON(w, 200, publicProjects(ps))
}
func within(root, p string) bool {
	rel, e := filepath.Rel(root, p)
	return e == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
func (a *App) diff(w http.ResponseWriter, r *http.Request) {
	uid := apphost.Current(r).User.ID
	if p, e := a.managedProject(r.Context(), uid, r.PathValue("id")); e == nil {
		var changes map[string]string
		if e = a.projectBroker(r.Context(), uid, "POST", "/"+p.ID+"/changes", map[string]string{}, &changes); e != nil {
			apphost.Problem(w, 503, e.Error())
			return
		}
		apphost.JSON(w, 200, changes)
		return
	}
	if apphost.Current(r).User.Role != "owner" {
		apphost.Problem(w, 403, "Choose one of your managed projects")
		return
	}
	p, e := a.findProject(apphost.Current(r).User.ID, r.PathValue("id"))
	if e != nil {
		apphost.Problem(w, 404, e.Error())
		return
	}
	ctx, c := context.WithTimeout(r.Context(), 15*time.Second)
	defer c()
	run := func(args ...string) string {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-C", p.Path, "--no-pager"}, args...)...)
		cmd.Env = cleanEnv()
		hideCommand(cmd)
		stdout, e := cmd.StdoutPipe()
		if e != nil {
			return ""
		}
		if cmd.Start() != nil {
			return "Git is not available for this project"
		}
		b, _ := io.ReadAll(io.LimitReader(stdout, 4<<20))
		if len(b) >= 4<<20 {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
		return string(b)
	}
	apphost.JSON(w, 200, map[string]string{"status": run("status", "--short", "--untracked-files=normal"), "diff": run("diff", "--no-ext-diff", "--no-textconv", "HEAD", "--", "."), "path": p.Path})
}
func (a *App) upload(w http.ResponseWriter, r *http.Request) {
	root, ok := a.root(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	if r.ParseMultipartForm(1<<20) != nil {
		apphost.Problem(w, 400, "Upload one UTF-8 text file up to 1 MB")
		return
	}
	defer r.MultipartForm.RemoveAll()
	f, h, e := r.FormFile("file")
	if e != nil {
		apphost.Problem(w, 400, "Choose a text file")
		return
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if e != nil || len(b) > 1<<20 || !utf8.Valid(b) || strings.ContainsRune(string(b), 0) {
		apphost.Problem(w, 400, "Only UTF-8 text/code files up to 1 MB are supported")
		return
	}
	id := apphost.ID()
	p, e := safePath(root, "Attachments", id+".txt")
	if e == nil {
		e = os.MkdirAll(filepath.Dir(p), 0700)
	}
	if e == nil {
		e = os.WriteFile(p, b, 0600)
	}
	if e != nil {
		apphost.Problem(w, 503, "My Drive could not save the attachment")
		return
	}
	apphost.JSON(w, 200, map[string]any{"id": id, "name": filepath.Base(h.Filename), "size": len(b)})
}
func (a *App) attachment(w http.ResponseWriter, r *http.Request) {
	root, ok := a.root(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if !idRE.MatchString(id) {
		http.NotFound(w, r)
		return
	}
	b, e := os.ReadFile(filepath.Join(root, "Attachments", id+".txt"))
	if e != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": "attachment.txt"}))
	w.Write(b)
}
