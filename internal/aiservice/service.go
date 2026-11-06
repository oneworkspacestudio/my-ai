// Package aiservice owns shared local inference. Prompts and replies are never persisted.
package aiservice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
)

const MaxQueue = 64
const MaxPerUser = 8

var modelName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:/+-]{0,159}$`)
var appName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,39}$`)

type Settings struct {
	Enabled bool   `json:"enabled"`
	Model   string `json:"model"`
}
type ticket struct {
	app, user string
	ready     chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
	started   time.Time
}
type Service struct {
	H                 *apphost.Host
	mu                sync.Mutex
	configMu          sync.Mutex
	settings          Settings
	waiting           []*ticket
	active            *ticket
	completed, failed uint64
	phase             string
	backend           string
	client            *http.Client
	managed           map[string]string
	wake              chan struct{}
	stopped           bool
}

func New(h *apphost.Host) (*Service, error) {
	s := &Service{H: h, backend: "http://127.0.0.1:11434", client: &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, phase: "Off", managed: map[string]string{}, wake: make(chan struct{}, 1)}
	if endpoint := os.Getenv("AISERVICE_OLLAMA_URL"); endpoint != "" {
		u, e := url.Parse(endpoint)
		if e != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
			return nil, errors.New("AI Service requires a loopback Ollama endpoint")
		}
		s.backend = endpoint
	}
	if _, e := h.DB.Exec(`CREATE TABLE IF NOT EXISTS ai_service_settings(id INTEGER PRIMARY KEY CHECK(id=1),value TEXT NOT NULL); CREATE TABLE IF NOT EXISTS ai_service_clients(id TEXT PRIMARY KEY, hash TEXT NOT NULL);`); e != nil {
		return nil, e
	}
	if _, e := h.DB.Exec(`CREATE TABLE IF NOT EXISTS ai_service_remote_clients(id TEXT PRIMARY KEY REFERENCES ai_service_clients(id) ON DELETE CASCADE)`); e != nil {
		return nil, e
	}
	var raw string
	if e := h.DB.QueryRow(`SELECT value FROM ai_service_settings WHERE id=1`).Scan(&raw); e == nil {
		if e = json.Unmarshal([]byte(raw), &s.settings); e != nil {
			return nil, e
		}
	}
	if s.settings.Model != "" && !modelName.MatchString(s.settings.Model) {
		return nil, errors.New("invalid saved AI Service model")
	}
	if s.settings.Enabled {
		s.phase = "Waiting for Ollama"
	}
	if key := os.Getenv("AISERVICE_MYAI_TOKEN"); key != "" {
		s.managed["myai"] = apphost.Hash(key)
	}
	h.Mux.HandleFunc("GET /api/service", h.Require(s.status))
	h.Mux.HandleFunc("GET /api/service/models", h.Require(s.models))
	h.Mux.HandleFunc("POST /api/service/settings", h.Owner(s.configure))
	h.Mux.HandleFunc("GET /api/service/clients", h.Owner(s.clients))
	h.Mux.HandleFunc("POST /api/service/clients", h.Owner(s.createClient))
	h.Mux.HandleFunc("DELETE /api/service/clients/{id}", h.Owner(s.deleteClient))
	h.Mux.HandleFunc("POST /api/chat", h.Require(func(w http.ResponseWriter, r *http.Request) {
		u := apphost.Current(r).User.ID
		s.infer(w, r, "browser", u, func() bool {
			c, e := r.Cookie("aiservice_session")
			if e != nil {
				return false
			}
			var expires, role string
			e = s.H.DB.QueryRow(`SELECT x.expires,u.role FROM host_sessions x JOIN host_users u ON u.id=x.user_id WHERE x.id=? AND u.id=?`, apphost.Hash(c.Value), u).Scan(&expires, &role)
			at, err := time.Parse(time.RFC3339Nano, expires)
			return e == nil && err == nil && role != "disabled" && time.Now().Before(at)
		})
	}))
	h.Mux.HandleFunc("POST /_internal/chat", s.internalChat)
	h.Mux.HandleFunc("POST /api/client/chat", s.remoteChat)
	h.Mux.HandleFunc("GET /_internal/models", s.internalModels)
	h.Mux.HandleFunc("GET /_internal/version", s.metadata)
	h.Mux.HandleFunc("GET /_internal/ps", s.metadata)
	h.Mux.HandleFunc("POST /_internal/show", s.metadata)
	return s, nil
}

func (s *Service) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}
func (s *Service) status(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	own := 0
	uid := apphost.Current(r).User.ID
	for _, t := range s.waiting {
		if t.user == uid {
			own++
		}
	}
	busy := s.active != nil
	app := ""
	if busy && apphost.Current(r).User.Role == "owner" {
		app = s.active.app
	}
	apphost.JSON(w, 200, map[string]any{"settings": s.settings, "phase": s.phase, "busy": busy, "activeApp": app, "queued": len(s.waiting), "yourQueued": own, "completed": s.completed, "failed": s.failed, "queueLimit": MaxQueue, "perUserLimit": MaxPerUser, "dependency": "Existing host Ollama at " + s.backend, "queuePolicy": "FIFO · one request at a time · queued requests are interrupted on restart"})
}

func (s *Service) backendRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var in io.Reader
	if body != nil {
		b, e := json.Marshal(body)
		if e != nil {
			return nil, e
		}
		in = bytes.NewReader(b)
	}
	r, e := http.NewRequestWithContext(ctx, method, s.backend+path, in)
	if e != nil {
		return nil, e
	}
	r.Header.Set("Content-Type", "application/json")
	return s.client.Do(r)
}
func (s *Service) inventory(ctx context.Context) ([]map[string]any, error) {
	r, e := s.backendRequest(ctx, "GET", "/api/tags", nil)
	if e != nil {
		return nil, e
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return nil, errors.New("Ollama is unavailable")
	}
	var data struct {
		Models []map[string]any `json:"models"`
	}
	e = json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&data)
	return data.Models, e
}
func (s *Service) models(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	models, e := s.inventory(ctx)
	if e != nil {
		apphost.Problem(w, 503, "Start the host Ollama service to choose an installed model")
		return
	}
	apphost.JSON(w, 200, map[string]any{"models": models})
}
func (s *Service) configure(w http.ResponseWriter, r *http.Request) {
	var next Settings
	if !apphost.Decode(w, r, &next) {
		return
	}
	if next.Model != "" && !modelName.MatchString(next.Model) || next.Enabled && next.Model == "" {
		apphost.Problem(w, 400, "Choose an installed model")
		return
	}
	s.configMu.Lock()
	defer s.configMu.Unlock()
	if next.Enabled {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		models, e := s.inventory(ctx)
		cancel()
		found := false
		for _, m := range models {
			if m["name"] == next.Model {
				found = true
			}
		}
		if e != nil || !found {
			apphost.Problem(w, 400, "The model must already be installed in host Ollama")
			return
		}
	}
	s.mu.Lock()
	old := s.settings
	if next.Enabled && old.Model != next.Model && (s.active != nil || len(s.waiting) > 0) {
		s.mu.Unlock()
		apphost.Problem(w, 409, "Turn the service off before changing a busy model")
		return
	}
	b, _ := json.Marshal(next)
	if _, e := s.H.DB.Exec(`INSERT INTO ai_service_settings(id,value)VALUES(1,?) ON CONFLICT(id)DO UPDATE SET value=excluded.value`, string(b)); e != nil {
		s.mu.Unlock()
		s.H.DBError(w, e)
		return
	}
	s.settings = next
	if !next.Enabled {
		s.phase = "Stopping"
		if s.active != nil {
			s.active.cancel()
		}
		for _, t := range s.waiting {
			t.cancel()
		}
	}
	if next.Enabled {
		s.phase = "Loading model"
	}
	s.mu.Unlock()
	// Never unload an active inference. The resident worker owns model changes too.
	s.signal()
	s.H.Audit(apphost.Current(r).User.ID, "ai-service-configure", "settings")
	apphost.JSON(w, 200, map[string]bool{"saved": true})
}

func (s *Service) acquire(ctx context.Context, app, user string, requireEnabled bool) (*ticket, int) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	t := &ticket{app: app, user: user, ready: make(chan struct{}), ctx: ctx, cancel: cancel, started: time.Now()}
	s.mu.Lock()
	if s.stopped || requireEnabled && !s.settings.Enabled {
		s.mu.Unlock()
		cancel()
		return nil, 503
	}
	n := 0
	if s.active != nil && s.active.app == app && s.active.user == user {
		n++
	}
	for _, q := range s.waiting {
		if q.app == app && q.user == user {
			n++
		}
	}
	if len(s.waiting) >= MaxQueue || n >= MaxPerUser {
		s.mu.Unlock()
		cancel()
		return nil, 429
	}
	if s.active == nil {
		s.active = t
		close(t.ready)
	} else {
		s.waiting = append(s.waiting, t)
	}
	s.mu.Unlock()
	select {
	case <-t.ready:
		if ctx.Err() != nil {
			s.release(t, false)
			return nil, 408
		}
		return t, 0
	case <-ctx.Done():
		s.mu.Lock()
		if s.active == t {
			s.mu.Unlock()
			s.release(t, false)
		} else {
			for i, q := range s.waiting {
				if q == t {
					s.waiting = append(s.waiting[:i], s.waiting[i+1:]...)
					break
				}
			}
			s.mu.Unlock()
		}
		cancel()
		return nil, 408
	}
}
func (s *Service) release(t *ticket, ok bool) {
	t.cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != t {
		return
	}
	if t.app != "resident" {
		if ok {
			s.completed++
		} else {
			s.failed++
		}
	}
	s.active = nil
	for len(s.waiting) > 0 {
		q := s.waiting[0]
		s.waiting = s.waiting[1:]
		if q.ctx.Err() != nil {
			continue
		}
		s.active = q
		close(q.ready)
		break
	}
	if t.app != "resident" {
		s.signal()
	}
}
func (s *Service) infer(w http.ResponseWriter, r *http.Request, app, user string, authorized func() bool) {
	var body map[string]any
	if e := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&body); e != nil || body == nil {
		apphost.Problem(w, 400, "Submit an Ollama chat request under 4 MB")
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	done := make(chan struct{})
	defer close(done)
	// Recheck credentials/session while waiting and while streaming.
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !authorized() {
					cancel()
					return
				}
			}
		}
	}()
	t, code := s.acquire(ctx, app, user, true)
	if code != 0 {
		apphost.Problem(w, code, map[int]string{503: "AI Service is off. Ask the owner to enable it", 429: "The AI queue is full for this user or server. Try shortly", 408: "Request cancelled or queue wait expired"}[code])
		return
	}
	ok := false
	defer func() { s.release(t, ok) }()
	s.mu.Lock()
	settings := s.settings
	s.mu.Unlock()
	if !settings.Enabled {
		apphost.Problem(w, 503, "AI Service is off")
		return
	}
	if model, exists := body["model"]; exists && model != "" && model != settings.Model {
		apphost.Problem(w, 409, "Choose the active AI Service model: "+settings.Model)
		return
	}
	body["model"] = settings.Model
	body["keep_alive"] = -1
	// GPU/host resource controls belong to the owner, not arbitrary API clients.
	options, exists := body["options"].(map[string]any)
	if !exists {
		if body["options"] != nil {
			apphost.Problem(w, 400, "Model options must be an object")
			return
		}
		options = map[string]any{}
	}
	delete(options, "num_gpu")
	delete(options, "main_gpu")
	delete(options, "num_thread")
	for key, bounds := range map[string][2]float64{"num_ctx": {512, 32768}, "num_predict": {1, 32768}, "num_batch": {1, 512}, "num_keep": {0, 32768}} {
		if raw, ok := options[key]; ok {
			v, number := raw.(float64)
			if !number || math.Trunc(v) != v || v < bounds[0] || v > bounds[1] {
				apphost.Problem(w, 400, "Shared model limits: context512–32768, output1–32768, batch1–512 and kept tokens0–32768")
				return
			}
		}
	}
	if options["num_ctx"] == nil {
		options["num_ctx"] = 8192
	}
	if options["num_predict"] == nil {
		options["num_predict"] = 4096
	}
	body["options"] = options
	resp, e := s.backendRequest(t.ctx, "POST", "/api/chat", body)
	if e != nil {
		apphost.Problem(w, 503, "Host Ollama could not serve the request")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		apphost.Problem(w, 502, "Host Ollama rejected the request. Check the selected model and available memory")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(200)
	buf := make([]byte, 32<<10)
	total := int64(0)
	for {
		n, e := resp.Body.Read(buf)
		if n > 0 {
			total += int64(n)
			if total > 32<<20 {
				return
			}
			if _, err := w.Write(buf[:n]); err != nil {
				return
			}
			if f, yes := w.(http.Flusher); yes {
				f.Flush()
			}
		}
		if e != nil {
			ok = e == io.EOF
			return
		}
	}
}

func (s *Service) clientAuthorized(r *http.Request) (string, bool) {
	host, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return "", false
	}
	app := r.Header.Get("X-AI-App")
	user := r.Header.Get("X-AI-User")
	key := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !appName.MatchString(app) || len(user) < 1 || len(user) > 128 || len(key) < 32 || len(key) > 200 {
		return "", false
	}
	hash := apphost.Hash(key)
	if expected := s.managed[app]; expected != "" {
		return app, apphost.Equal(hash, expected)
	}
	var expected string
	e = s.H.DB.QueryRow(`SELECT hash FROM ai_service_clients WHERE id=?`, app).Scan(&expected)
	return app, e == nil && apphost.Equal(expected, hash)
}
func (s *Service) internalChat(w http.ResponseWriter, r *http.Request) {
	app, ok := s.clientAuthorized(r)
	if !ok {
		apphost.Problem(w, 401, "Use an authorized app credential")
		return
	}
	s.infer(w, r, app, r.Header.Get("X-AI-User"), func() bool { _, ok := s.clientAuthorized(r); return ok })
}

// Remote credentials are explicit owner grants. Existing local and managed My AI
// credentials never gain remote access. Server apps relay requests; browser
// origins/cookies are neither credentials nor an accepted client transport here.
func (s *Service) remoteAuthorized(r *http.Request) (string, bool) {
	if r.Header.Get("Origin") != "" || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		return "", false
	}
	key := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if len(key) < 32 || len(key) > 200 {
		return "", false
	}
	var id string
	e := s.H.DB.QueryRow(`SELECT c.id FROM ai_service_clients c JOIN ai_service_remote_clients x ON x.id=c.id WHERE c.hash=?`, apphost.Hash(key)).Scan(&id)
	return id, e == nil
}
func (s *Service) remoteChat(w http.ResponseWriter, r *http.Request) {
	app, ok := s.remoteAuthorized(r)
	if !ok {
		apphost.Problem(w, 401, "Use an owner-issued remote app credential from a server app")
		return
	}
	// All requests sharing a remote capability share one quota. Caller supplied
	// user/app headers cannot manufacture additional admission slots.
	s.infer(w, r, app, "remote-client", func() bool { id, ok := s.remoteAuthorized(r); return ok && id == app })
}
func (s *Service) internalModels(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.clientAuthorized(r); !ok {
		apphost.Problem(w, 401, "Use an authorized app credential")
		return
	}
	s.mu.Lock()
	settings := s.settings
	s.mu.Unlock()
	models := []map[string]any{}
	if settings.Enabled {
		models = append(models, map[string]any{"name": settings.Model, "capabilities": []string{"completion", "tools"}})
	}
	apphost.JSON(w, 200, map[string]any{"models": models, "enabled": settings.Enabled})
}

// Only fixed, read-only model metadata is exposed to trusted server apps.
func (s *Service) metadata(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.clientAuthorized(r); !ok {
		apphost.Problem(w, 401, "Use an authorized app credential")
		return
	}
	kind := strings.TrimPrefix(r.URL.Path, "/_internal/")
	var body any
	if kind == "show" {
		var in struct {
			Model   string `json:"model"`
			Verbose bool   `json:"verbose"`
		}
		if !apphost.Decode(w, r, &in) {
			return
		}
		s.mu.Lock()
		selected := s.settings.Model
		s.mu.Unlock()
		if in.Model != selected {
			apphost.Problem(w, 409, "Choose the active AI Service model")
			return
		}
		body = in
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	resp, e := s.backendRequest(ctx, r.Method, "/api/"+kind, body)
	if e != nil {
		apphost.Problem(w, 503, "Host Ollama is unavailable")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		apphost.Problem(w, 502, "Host Ollama metadata is unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	io.Copy(w, io.LimitReader(resp.Body, 4<<20))
}
func (s *Service) clients(w http.ResponseWriter, r *http.Request) {
	rows, e := s.H.DB.Query(`SELECT c.id, x.id IS NOT NULL FROM ai_service_clients c LEFT JOIN ai_service_remote_clients x ON x.id=c.id ORDER BY c.id`)
	if e != nil {
		s.H.DBError(w, e)
		return
	}
	defer rows.Close()
	ids := []string{}
	remote := []string{}
	for rows.Next() {
		var id string
		var enabled bool
		if rows.Scan(&id, &enabled) == nil {
			ids = append(ids, id)
			if enabled {
				remote = append(remote, id)
			}
		}
	}
	apphost.JSON(w, 200, map[string]any{"clients": ids, "remoteClients": remote, "myaiManaged": s.managed["myai"] != ""})
}
func (s *Service) createClient(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID     string `json:"id"`
		Remote bool   `json:"remote"`
	}
	if !apphost.Decode(w, r, &in) {
		return
	}
	if !appName.MatchString(in.ID) || in.ID == "browser" || in.ID == "resident" || s.managed[in.ID] != "" {
		apphost.Problem(w, 400, "Choose another app ID using lowercase letters, digits and hyphens")
		return
	}
	key := apphost.ID() + apphost.ID()
	tx, e := s.H.DB.Begin()
	if e != nil {
		s.H.DBError(w, e)
		return
	}
	defer tx.Rollback()
	if _, e := tx.Exec(`INSERT INTO ai_service_clients(id,hash)VALUES(?,?)`, in.ID, apphost.Hash(key)); e != nil {
		apphost.Problem(w, 409, "That app already has a credential. Revoke it before issuing a replacement")
		return
	}
	if in.Remote {
		if _, e := tx.Exec(`INSERT INTO ai_service_remote_clients(id)VALUES(?)`, in.ID); e != nil {
			s.H.DBError(w, e)
			return
		}
	}
	if e := tx.Commit(); e != nil {
		s.H.DBError(w, e)
		return
	}
	s.H.Audit(apphost.Current(r).User.ID, "ai-service-client-create", in.ID)
	apphost.JSON(w, 201, map[string]any{"id": in.ID, "key": key, "remote": in.Remote})
}
func (s *Service) deleteClient(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tx, e := s.H.DB.Begin()
	if e != nil {
		s.H.DBError(w, e)
		return
	}
	defer tx.Rollback()
	if _, e := tx.Exec(`DELETE FROM ai_service_remote_clients WHERE id=?`, id); e != nil {
		s.H.DBError(w, e)
		return
	}
	if _, e := tx.Exec(`DELETE FROM ai_service_clients WHERE id=?`, id); e != nil {
		s.H.DBError(w, e)
		return
	}
	if e := tx.Commit(); e != nil {
		s.H.DBError(w, e)
		return
	}
	s.H.Audit(apphost.Current(r).User.ID, "ai-service-client-revoke", id)
	apphost.JSON(w, 200, map[string]bool{"revoked": true})
}

func (s *Service) resident(ctx context.Context, model string, keep int) error {
	resp, e := s.backendRequest(ctx, "POST", "/api/generate", map[string]any{"model": model, "keep_alive": keep, "stream": false, "options": map[string]any{"num_ctx": 8192}})
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	_, e = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != 200 {
		return errors.New("Ollama model operation failed")
	}
	return e
}
func (s *Service) Run(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	loaded := ""
	defer func() {
		s.mu.Lock()
		s.stopped = true
		if s.active != nil {
			s.active.cancel()
		}
		for _, q := range s.waiting {
			q.cancel()
		}
		s.mu.Unlock()
	}()
	for {
		s.mu.Lock()
		cfg := s.settings
		idle := s.active == nil && len(s.waiting) == 0
		s.mu.Unlock()
		if idle {
			t, code := s.acquire(ctx, "resident", "resident", false)
			if code == 0 {
				s.mu.Lock()
				cfg = s.settings
				s.mu.Unlock()
				opctx, cancel := context.WithTimeout(t.ctx, 3*time.Minute)
				e := error(nil)
				if loaded != "" && (!cfg.Enabled || loaded != cfg.Model) {
					e = s.resident(opctx, loaded, 0)
					if e == nil {
						loaded = ""
					}
				}
				if e == nil && cfg.Enabled {
					firstPin := loaded != cfg.Model
					loaded = cfg.Model
					resp, err := s.backendRequest(opctx, "GET", "/api/ps", nil)
					present := false
					if err == nil {
						var ps struct {
							Models []struct {
								Name string `json:"name"`
							} `json:"models"`
						}
						if resp.StatusCode == 200 && json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&ps) == nil {
							for _, m := range ps.Models {
								if m.Name == cfg.Model {
									present = true
								}
							}
						}
						resp.Body.Close()
					}
					if !present || firstPin {
						e = s.resident(opctx, cfg.Model, -1)
					}
				}
				cancel()
				s.mu.Lock()
				if s.settings == cfg {
					if e != nil {
						s.phase = "Ollama unavailable or model could not load; retrying"
					} else if cfg.Enabled {
						s.phase = "Ready · model kept loaded"
					} else {
						s.phase = "Off"
					}
				}
				s.mu.Unlock()
				s.release(t, true)
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		case <-s.wake:
		}
	}
}
