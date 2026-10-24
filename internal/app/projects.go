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
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
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
	for _, p := range a.allProjects(uid) {
		if p.ID == id {
			if _, e := safePath(p.Path); e != nil {
				return Project{}, e
			}
			if f, e := os.Stat(p.Path); e != nil || !f.IsDir() {
				return Project{}, errors.New("project root is no longer available")
			}
			return p, nil
		}
	}
	return Project{}, errors.New("Register and select an approved project first")
}
func (a *App) projects(w http.ResponseWriter, r *http.Request) {
	apphost.JSON(w, 200, a.allProjects(apphost.Current(r).User.ID))
}
func (a *App) projectSave(w http.ResponseWriter, r *http.Request) {
	if apphost.Current(r).User.Role != "owner" {
		apphost.Problem(w, 403, "Only the owner can approve a host directory")
		return
	}
	var in struct {
		Path    string `json:"path"`
		Name    string `json:"name"`
		Confirm bool   `json:"confirm"`
		Remove  string `json:"remove,omitempty"`
	}
	if !apphost.Decode(w, r, &in) {
		return
	}
	uid := apphost.Current(r).User.ID
	a.mu.Lock()
	defer a.mu.Unlock()
	ps := a.allProjects(uid)
	if in.Remove != "" {
		out := []Project{}
		for _, p := range ps {
			if p.ID != in.Remove {
				out = append(out, p)
			}
		}
		if atomic(a.projectPath(uid), out) != nil {
			apphost.Problem(w, 503, "Could not save projects")
			return
		}
		apphost.JSON(w, 200, out)
		return
	}
	path := filepath.Clean(in.Path)
	if !in.Confirm || !filepath.IsAbs(path) || filepath.Dir(path) == path || len(in.Name) > 100 {
		apphost.Problem(w, 400, "Confirm one dedicated absolute project folder, not a whole drive")
		return
	}
	for _, v := range []string{a.H.Config.Data, os.Getenv("USERPROFILE"), os.Getenv("SystemRoot"), os.Getenv("APPDATA"), os.Getenv("LOCALAPPDATA"), filepath.Join(os.Getenv("USERPROFILE"), ".codex")} {
		if v != "" && (strings.EqualFold(path, v) || within(v, path) && v != os.Getenv("USERPROFILE")) {
			apphost.Problem(w, 400, "Choose a project outside application data, Windows and global Codex folders")
			return
		}
	}
	if _, e := safePath(path); e != nil {
		apphost.Problem(w, 400, e.Error())
		return
	}
	f, e := os.Stat(path)
	if e != nil || !f.IsDir() {
		apphost.Problem(w, 400, "The project folder does not exist on the host machine")
		return
	}
	for _, p := range ps {
		if strings.EqualFold(p.Path, path) {
			apphost.JSON(w, 200, ps)
			return
		}
	}
	name := in.Name
	if name == "" {
		name = filepath.Base(path)
	}
	ps = append(ps, Project{ID: apphost.ID(), Name: name, Path: path})
	if e = atomic(a.projectPath(uid), ps); e != nil {
		apphost.Problem(w, 503, "Could not save projects")
		return
	}
	a.H.Audit(uid, "project-register", path)
	apphost.JSON(w, 200, ps)
}
func within(root, p string) bool {
	rel, e := filepath.Rel(root, p)
	return e == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
func (a *App) diff(w http.ResponseWriter, r *http.Request) {
	if apphost.Current(r).User.Role != "owner" {
		apphost.Problem(w, 403, "Owner project access required")
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
