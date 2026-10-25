package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/oneworkspacestudio/my-ai/internal/searchruntime"
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type App struct {
	searchMu                 sync.Mutex
	search                   *searchruntime.Runtime
	searchTemp               string
	searchConnectionOverride func(context.Context) (searchruntime.Connection, error)
	downloads                *downloadQueue
	H                        *apphost.Host
	mu                       sync.Mutex
	projectIO                sync.Mutex
	projectTurns             map[string]bool
	client                   *http.Client
	running                  map[string]context.CancelFunc
	agents                   map[string]*Agent
}
type Message struct {
	ID       string `json:"id"`
	Role     string `json:"role"`
	Content  string `json:"content"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	At       string `json:"at"`
	Status   string `json:"status,omitempty"`
}
type Event struct {
	At     string          `json:"at"`
	Method string          `json:"method"`
	Data   json.RawMessage `json:"data"`
}
type Session struct {
	ToolInput      string    `json:"-"`
	SearchMode     string    `json:"-"`
	SearchTimeZone string    `json:"-"`
	SearchQuery    string    `json:"-"`
	Schema         int       `json:"schema"`
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Mode           string    `json:"mode"`
	Project        string    `json:"project,omitempty"`
	Provider       string    `json:"provider"`
	Model          string    `json:"model"`
	Effort         string    `json:"effort"`
	Policy         string    `json:"policy"`
	Pinned         bool      `json:"pinned"`
	Archived       bool      `json:"archived"`
	Revision       int       `json:"revision"`
	Updated        string    `json:"updated"`
	Thread         string    `json:"thread,omitempty"`
	Messages       []Message `json:"messages"`
	Events         []Event   `json:"events"`
}

var idRE = regexp.MustCompile(`^[a-f0-9]{48}$`)
var modelRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:/+-]{0,159}$`)

func New(h *apphost.Host) (*App, error) {
	q, e := newDownloadQueue(h.Config.Data)
	if e != nil {
		return nil, e
	}
	a := &App{downloads: q, H: h, client: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, projectTurns: map[string]bool{}, running: map[string]context.CancelFunc{}, agents: map[string]*Agent{}}
	for _, v := range []struct {
		p string
		f http.HandlerFunc
	}{{"GET /api/state", a.state}, {"GET /api/sessions", a.list}, {"POST /api/sessions", a.save}, {"GET /api/sessions/{id}", a.get}, {"DELETE /api/sessions/{id}", a.remove}, {"GET /api/sessions/{id}/export", a.export}, {"POST /api/sessions/{id}/turn", a.turn}, {"POST /api/sessions/{id}/stop", a.stop}, {"POST /api/sessions/{id}/approval", a.approval}, {"GET /api/code-models", a.codeModels}, {"GET /api/models", a.models}, {"GET /api/models/downloads", a.downloadList}, {"POST /api/models/downloads/{id}/cancel", a.downloadCancel}, {"GET /api/tools", a.toolsInfo}, {"POST /api/providers", a.providers}, {"POST /api/models/pull", a.pull}, {"POST /api/models/delete", a.deleteModel}, {"GET /api/projects", a.projects}, {"POST /api/projects", a.projectSave}, {"GET /api/projects/{id}/diff", a.diff}, {"POST /api/projects/{id}/sync", a.projectAction}, {"POST /api/attachments", a.upload}, {"GET /api/attachments/{id}", a.attachment}} {
		h.Mux.HandleFunc(v.p, h.Require(v.f))
	}
	return a, nil
}
func (a *App) Run(ctx context.Context) error {
	defer a.closeSearch()
	go func() { _, _ = a.searchConnection(ctx) }()
	done := make(chan struct{})
	go func() { a.downloads.run(ctx); close(done) }()
	<-ctx.Done()
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, c := range a.running {
		c()
	}
	for _, g := range a.agents {
		g.Close()
	}
	<-done
	return nil
}
func now() string { return time.Now().UTC().Format(time.RFC3339Nano) }
func safePath(root string, parts ...string) (string, error) {
	p := filepath.Join(append([]string{root}, parts...)...)
	rel, e := filepath.Rel(root, p)
	if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("path outside storage")
	}
	for q := p; ; q = filepath.Dir(q) {
		f, e := os.Lstat(q)
		if e == nil && (f.Mode()&os.ModeSymlink != 0 || rejectLink(q)) {
			return "", errors.New("linked storage paths are not supported")
		}
		if filepath.Dir(q) == q {
			break
		}
	}
	return p, nil
}
func atomic(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	tmp := path + ".pending-" + apphost.ID()
	f, e := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	defer os.Remove(tmp)
	_, e = f.Write(b)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	return os.Rename(tmp, path)
}
func (a *App) userRoot(r *http.Request) (string, error) {
	u := apphost.Current(r).User
	root := os.Getenv("MYAI_DRIVE_ROOT")
	if root == "" {
		if a.H.Config.Issuer != "" {
			return "", errors.New("Workspace must configure the private My AI Drive source")
		}
		root = filepath.Join(a.H.Config.Data, "My Drive", "My AI")
	} else {
		endpoint := os.Getenv("MYAI_STORAGE_GUARD_URL")
		if endpoint == "" {
			return "", errors.New("Drive isolation guard is missing")
		}
		req, e := http.NewRequestWithContext(r.Context(), "GET", endpoint, nil)
		if e != nil {
			return "", e
		}
		req.Header.Set("Authorization", "Bearer "+os.Getenv("MYAI_STORAGE_GUARD_TOKEN"))
		req.Header.Set("X-MyAI-Subject", u.Subject)
		req.Header.Set("X-MyAI-Issuer", u.Issuer)
		resp, e := a.client.Do(req)
		if e != nil {
			return "", errors.New("My Drive is unavailable; no session changes were saved")
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return "", errors.New("My Drive identity isolation is unavailable; reconnect from Workspace")
		}
	}
	p, e := safePath(root, "users", u.ID)
	if e == nil {
		e = os.MkdirAll(p, 0700)
	}
	return p, e
}
func (a *App) read(root, id string) (s Session, e error) {
	if !idRE.MatchString(id) {
		return s, errors.New("session not found")
	}
	p, e := safePath(root, "Sessions", id, "session.json")
	if e != nil {
		return s, e
	}
	b, e := os.ReadFile(p)
	if e != nil {
		return s, e
	}
	if len(b) > 16<<20 {
		return s, errors.New("session is too large")
	}
	e = json.Unmarshal(b, &s)
	return
}
func (a *App) write(root string, s *Session) error {
	p, e := safePath(root, "Sessions", s.ID, "session.json")
	if e != nil {
		return e
	}
	s.Schema = 1
	s.Revision++
	s.Updated = now()
	if s.Revision > 1 {
		if b, e := os.ReadFile(p); e == nil {
			var prev Session
			if json.Unmarshal(b, &prev) == nil {
				if e = atomic(filepath.Join(filepath.Dir(p), "previous.json"), prev); e != nil {
					return e
				}
			}
		}
	}
	return atomic(p, s)
}
func (a *App) root(w http.ResponseWriter, r *http.Request) (string, bool) {
	p, e := a.userRoot(r)
	if e != nil {
		apphost.Problem(w, 503, e.Error())
		return "", false
	}
	return p, true
}
func (a *App) state(w http.ResponseWriter, r *http.Request) {
	root, e := a.userRoot(r)
	status := "My Drive connected"
	if e != nil {
		status = e.Error()
	}
	settings, _ := a.settings(apphost.Current(r).User.ID)
	safe := []map[string]any{}
	for _, p := range settings {
		safe = append(safe, map[string]any{"id": p.ID, "base": p.Base, "configured": p.Key != "" || p.ID == "ollama"})
	}
	apphost.JSON(w, 200, map[string]any{"storage": status, "storageReady": e == nil, "storagePath": filepath.Base(root), "providers": safe, "codexAvailable": codexPath() != "", "codeNotice": "Projects are stored in My Drive and connected to your private Git repositories. Allow changes to let My AI edit the selected project."})
}
func (a *App) list(w http.ResponseWriter, r *http.Request) {
	root, ok := a.root(w, r)
	if !ok {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	entries, _ := os.ReadDir(filepath.Join(root, "Sessions"))
	out := []Session{}
	q := strings.ToLower(r.URL.Query().Get("q"))
	for _, entry := range entries {
		if s, e := a.read(root, entry.Name()); e == nil {
			b, _ := json.Marshal(s)
			if q == "" || strings.Contains(strings.ToLower(string(b)), q) {
				s.Messages = nil
				s.Events = nil
				out = append(out, s)
			}
		}
	}
	apphost.JSON(w, 200, out)
}
func (a *App) get(w http.ResponseWriter, r *http.Request) {
	root, ok := a.root(w, r)
	if !ok {
		return
	}
	a.mu.Lock()
	s, e := a.read(root, r.PathValue("id"))
	a.mu.Unlock()
	if e != nil {
		apphost.Problem(w, 404, "Session not found")
		return
	}
	apphost.JSON(w, 200, s)
}
func validSession(s Session) bool {
	return idRE.MatchString(s.ID) && len(s.Title) > 0 && len(s.Title) <= 160 && (s.Mode == "chat" || s.Mode == "code") && (s.Provider == "openai" || s.Provider == "deepseek" || s.Provider == "ollama") && (s.Model == "" || modelRE.MatchString(s.Model)) && (s.Policy == "read-only" || s.Policy == "workspace-write") && len(s.Messages) < 2000 && len(s.Events) < 4000
}
func (a *App) save(w http.ResponseWriter, r *http.Request) {
	root, ok := a.root(w, r)
	if !ok {
		return
	}
	var s Session
	if !apphost.Decode(w, r, &s) {
		return
	}
	if !validSession(s) {
		apphost.Problem(w, 400, "Check session fields")
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	old, e := a.read(root, s.ID)
	if e == nil {
		if a.running[apphost.Current(r).User.ID+":"+s.ID] != nil {
			apphost.Problem(w, 409, "Stop the active turn before changing this session")
			return
		}
		if old.Revision != s.Revision {
			apphost.Problem(w, 409, "Session changed in another tab; reload it")
			return
		}
		s.Messages = old.Messages
		s.Events = old.Events
		s.Thread = old.Thread
		if s.Provider != old.Provider || s.Project != old.Project {
			s.Thread = ""
		}
	} else {
		s.Thread = ""
		s.Revision = 0
		if len(s.Messages) > 0 {
			for i := range s.Messages {
				s.Messages[i].Status = "imported"
			}
		}
	}
	if e = a.write(root, &s); e != nil {
		apphost.Problem(w, 503, "My Drive write failed; retry without closing this window")
		return
	}
	apphost.JSON(w, 200, s)
}
func (a *App) remove(w http.ResponseWriter, r *http.Request) {
	root, ok := a.root(w, r)
	if !ok {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	s, e := a.read(root, r.PathValue("id"))
	if e != nil {
		apphost.Problem(w, 404, "Session not found")
		return
	}
	if a.running[apphost.Current(r).User.ID+":"+s.ID] != nil {
		apphost.Problem(w, 409, "Stop the active turn first")
		return
	}
	p, _ := safePath(root, "Sessions", s.ID)
	trash := filepath.Join(root, "Trash")
	os.MkdirAll(trash, 0700)
	if os.Rename(p, filepath.Join(trash, s.ID+"-"+apphost.ID())) != nil {
		apphost.Problem(w, 503, "Could not move session to private trash")
		return
	}
	apphost.JSON(w, 200, map[string]bool{"deleted": true})
}
func (a *App) export(w http.ResponseWriter, r *http.Request) {
	root, ok := a.root(w, r)
	if !ok {
		return
	}
	a.mu.Lock()
	s, e := a.read(root, r.PathValue("id"))
	a.mu.Unlock()
	if e != nil {
		apphost.Problem(w, 404, "Session not found")
		return
	}
	a.H.Audit(apphost.Current(r).User.ID, "session-export", s.ID)
	w.Header().Set("Content-Disposition", `attachment; filename="my-ai-`+s.ID+`.json"`)
	if r.URL.Query().Get("format") == "markdown" {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		fmt.Fprintln(w, "# "+s.Title)
		for _, m := range s.Messages {
			fmt.Fprintf(w, "\n## %s · %s\n\n%s\n", m.Role, m.Model, m.Content)
		}
		return
	}
	apphost.JSON(w, 200, s)
}
