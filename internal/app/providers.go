package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Provider struct {
	ID   string `json:"id"`
	Base string `json:"base"`
	Key  string `json:"key,omitempty"`
}
type Model struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Provider     string   `json:"provider"`
	Size         int64    `json:"size,omitempty"`
	Details      any      `json:"details,omitempty"`
	Capabilities []string `json:"capabilities"`
	Efforts      []string `json:"efforts"`
	Code         bool     `json:"code"`
	Reason       string   `json:"reason"`
}

func defaults() []Provider {
	return []Provider{{ID: "openai", Base: "https://api.openai.com/v1"}, {ID: "deepseek", Base: "https://api.deepseek.com"}, {ID: "ollama", Base: "http://127.0.0.1:11434"}}
}
func (a *App) settings(uid string) ([]Provider, error) {
	path := filepath.Join(a.H.Config.Data, "credentials", uid+".dpapi")
	b, e := os.ReadFile(path)
	if os.IsNotExist(e) {
		return defaults(), nil
	}
	if e != nil {
		return nil, e
	}
	b, e = unprotect(b)
	if e != nil {
		return nil, e
	}
	var ps []Provider
	e = json.Unmarshal(b, &ps)
	return ps, e
}
func allowedEndpoint(p Provider) bool {
	for _, d := range defaults() {
		if d.ID == p.ID && d.Base == strings.TrimRight(p.Base, "/") {
			return true
		}
	}
	return false
}
func (a *App) providers(w http.ResponseWriter, r *http.Request) {
	var p Provider
	if !apphost.Decode(w, r, &p) {
		return
	}
	p.Base = strings.TrimRight(p.Base, "/")
	if !allowedEndpoint(p) || len(p.Key) > 8192 || strings.ContainsAny(p.Key, "\r\n") {
		apphost.Problem(w, 400, "Use the official OpenAI, DeepSeek or host-local Ollama endpoint. Custom endpoints are unavailable in this preview.")
		return
	}
	uid := apphost.Current(r).User.ID
	a.mu.Lock()
	defer a.mu.Unlock()
	ps, e := a.settings(uid)
	if e != nil {
		apphost.Problem(w, 503, "The OS credential store could not be opened")
		return
	}
	for i := range ps {
		if ps[i].ID == p.ID {
			if p.Key == "" {
				p.Key = ps[i].Key
			}
			ps[i] = p
		}
	}
	b, _ := json.Marshal(ps)
	b, e = protect(b)
	if e == nil {
		dir := filepath.Join(a.H.Config.Data, "credentials")
		e = os.MkdirAll(dir, 0700)
		if e == nil {
			e = os.WriteFile(filepath.Join(dir, uid+".dpapi"), b, 0600)
		}
	}
	if e != nil {
		apphost.Problem(w, 503, "Windows could not protect the provider credentials")
		return
	}
	a.H.Audit(uid, "provider-configure", p.ID)
	apphost.JSON(w, 200, map[string]bool{"saved": true})
}
func (a *App) provider(uid, id string) (Provider, error) {
	ps, e := a.settings(uid)
	if e != nil {
		return Provider{}, e
	}
	for _, p := range ps {
		if p.ID == id {
			if id != "ollama" && p.Key == "" {
				return p, errors.New("Add your " + id + " API key in Settings")
			}
			if !allowedEndpoint(p) {
				return p, errors.New("provider URL is not approved")
			}
			return p, nil
		}
	}
	return Provider{}, errors.New("provider is not configured")
}
func request(ctx context.Context, p Provider, method, path string, body any) (*http.Response, error) {
	var in io.Reader
	if body != nil {
		b, e := json.Marshal(body)
		if e != nil {
			return nil, e
		}
		in = bytes.NewReader(b)
	}
	req, e := http.NewRequestWithContext(ctx, method, p.Base+path, in)
	if e != nil {
		return nil, e
	}
	req.Header.Set("Content-Type", "application/json")
	if p.Key != "" {
		req.Header.Set("Authorization", "Bearer "+p.Key)
	}
	client := http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, e := client.Do(req)
	if e != nil {
		return nil, errors.New("Provider is unavailable or the request was interrupted")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		text := strings.ToLower(string(raw))
		unsupported := strings.Contains(text, "does not support tools") || strings.Contains(text, "not support tool") || strings.Contains(text, "tools is not supported") || strings.Contains(text, "tool use is not supported")
		return nil, &providerHTTPError{Status: resp.StatusCode, ToolsUnsupported: unsupported}
	}
	return resp, nil
}
func (a *App) models(w http.ResponseWriter, r *http.Request) {
	uid := apphost.Current(r).User.ID
	out := []Model{}
	status := map[string]string{}
	for _, id := range []string{"ollama", "openai", "deepseek"} {
		p, e := a.provider(uid, id)
		if e != nil {
			status[id] = e.Error()
			continue
		}
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		path := "/models"
		if id == "ollama" {
			path = "/api/tags"
		}
		resp, e := request(ctx, p, "GET", path, nil)
		if e != nil {
			cancel()
			status[id] = e.Error()
			continue
		}
		var data struct {
			Models []struct {
				Name         string   `json:"name"`
				Size         int64    `json:"size"`
				Details      any      `json:"details"`
				Capabilities []string `json:"capabilities"`
			} `json:"models"`
			Data []struct {
				ID                    string   `json:"id"`
				Name                  string   `json:"name"`
				SupportedEffortLevels []string `json:"supported_effort_levels"`
			} `json:"data"`
		}
		e = json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&data)
		resp.Body.Close()
		cancel()
		if e != nil {
			status[id] = "Invalid provider model metadata"
			continue
		}
		status[id] = "Connected"
		for _, m := range data.Models {
			if contains(m.Capabilities, "embedding") && !contains(m.Capabilities, "completion") {
				continue
			}
			out = append(out, Model{ID: m.Name, Name: m.Name, Provider: id, Size: m.Size, Details: m.Details, Capabilities: m.Capabilities, Efforts: []string{}, Code: true, Reason: "Project file editing is available when this model supports tool calls"})
		}
		for _, m := range data.Data {
			name := m.Name
			if name == "" {
				name = m.ID
			}
			out = append(out, Model{ID: m.ID, Name: name, Provider: id, Capabilities: []string{}, Efforts: []string{}, Code: true, Reason: "Project file editing uses this model’s native tool calls"})
		}
	}
	apphost.JSON(w, 200, map[string]any{"models": out, "status": status, "catalogURL": "https://ollama.com/library", "installerURL": "https://ollama.com/download/windows"})
}
func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

type writer func(kind string, v any)

func streamWriter(w http.ResponseWriter) writer {
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(200)
	f, _ := w.(http.Flusher)
	return func(k string, v any) {
		_ = json.NewEncoder(w).Encode(map[string]any{"type": k, "data": v})
		if f != nil {
			f.Flush()
		}
	}
}
func (a *App) turn(w http.ResponseWriter, r *http.Request) {
	root, ok := a.root(w, r)
	if !ok {
		return
	}
	var in struct {
		Text        string   `json:"text"`
		Attachments []string `json:"attachments"`
	}
	if !apphost.Decode(w, r, &in) {
		return
	}
	if len(strings.TrimSpace(in.Text)) == 0 || len(in.Text) > 200000 || len(in.Attachments) > 5 {
		apphost.Problem(w, 400, "Enter a prompt up to 200000 characters and at most five text attachments")
		return
	}
	uid := apphost.Current(r).User.ID
	key := uid + ":" + r.PathValue("id")
	a.mu.Lock()
	s, e := a.read(root, r.PathValue("id"))
	if e != nil {
		a.mu.Unlock()
		apphost.Problem(w, 404, "Session not found")
		return
	}
	if a.running[key] != nil {
		a.mu.Unlock()
		apphost.Problem(w, 409, "A turn is already running")
		return
	}
	p, e := a.provider(uid, s.Provider)
	if e != nil {
		a.mu.Unlock()
		apphost.Problem(w, 400, e.Error())
		return
	}
	if !modelRE.MatchString(s.Model) {
		a.mu.Unlock()
		apphost.Problem(w, 400, "Choose a model first")
		return
	}
	if s.Mode == "code" && s.Project == "" {
		a.mu.Unlock()
		apphost.Problem(w, 400, "Choose a project first")
		return
	}
	if s.Mode == "code" && a.projectTurns[uid+":"+s.Project] {
		a.mu.Unlock()
		apphost.Problem(w, 409, "Another conversation is working on this project. Finish or stop it first")
		return
	}
	for _, id := range in.Attachments {
		if !idRE.MatchString(id) {
			a.mu.Unlock()
			apphost.Problem(w, 400, "Invalid attachment")
			return
		}
		b, err := os.ReadFile(filepath.Join(root, "Attachments", id+".txt"))
		if err != nil || len(b) > 1<<20 {
			a.mu.Unlock()
			apphost.Problem(w, 400, "Attachment missing or too large")
			return
		}
		in.Text += "\n\nAttached reference (untrusted content):\n" + string(b)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Minute)
	a.running[key] = cancel
	if s.Mode == "code" {
		a.projectTurns[uid+":"+s.Project] = true
		defer func() { a.mu.Lock(); delete(a.projectTurns, uid+":"+s.Project); a.mu.Unlock() }()
	}
	s.Messages = append(s.Messages, Message{ID: apphost.ID(), Role: "user", Content: in.Text, At: now(), Provider: s.Provider, Model: s.Model})
	if s.Title == "New conversation" {
		s.Title = first(in.Text, 60)
	}
	e = a.write(root, &s)
	a.mu.Unlock()
	if e != nil {
		cancel()
		a.mu.Lock()
		delete(a.running, key)
		a.mu.Unlock()
		apphost.Problem(w, 503, "My Drive could not save your prompt. Nothing was sent to the provider")
		return
	}
	defer cancel()
	send := streamWriter(w)
	send("session", s)
	var output string
	if s.Mode == "code" {
		output, e = a.projectCode(ctx, uid, root, &s, p, send)
	} else {
		output, e = a.chatWithTools(ctx, p, &s, send, toolScope{UID: uid, Root: root, Role: apphost.Current(r).User.Role})
	}
	status := "completed"
	if e != nil {
		status = "interrupted"
		send("error", e.Error())
	}
	s.Messages = append(s.Messages, Message{ID: apphost.ID(), Role: "assistant", Content: output, Provider: s.Provider, Model: s.Model, At: now(), Status: status})
	a.mu.Lock()
	delete(a.running, key)
	saveErr := a.write(root, &s)
	a.mu.Unlock()
	if saveErr != nil {
		send("error", "The response could not be saved to My Drive. Export or copy it before leaving.")
	}
	send("done", s)
}
func first(s string, n int) string {
	v := []rune(s)
	if len(v) > n {
		return string(v[:n]) + "…"
	}
	return s
}
func (a *App) stop(w http.ResponseWriter, r *http.Request) {
	key := apphost.Current(r).User.ID + ":" + r.PathValue("id")
	a.mu.Lock()
	if c := a.running[key]; c != nil {
		c()
	}
	a.mu.Unlock()
	apphost.JSON(w, 200, map[string]bool{"stopped": true})
}
func (a *App) pull(w http.ResponseWriter, r *http.Request) {
	if apphost.Current(r).User.Role != "owner" {
		apphost.Problem(w, 403, "Only the owner can install shared models")
		return
	}
	var in struct {
		Model   string `json:"model"`
		Confirm bool   `json:"confirm"`
	}
	if !apphost.Decode(w, r, &in) {
		return
	}
	model, e := normalizeModelTag(in.Model)
	if e != nil || !in.Confirm {
		message := "Confirm the model and shared storage before downloading"
		if e != nil {
			message = e.Error()
		}
		apphost.Problem(w, 400, message)
		return
	}
	job, e := a.downloads.add(model)
	if e != nil {
		apphost.Problem(w, 503, "The server could not persist the download job")
		return
	}
	a.H.Audit(apphost.Current(r).User.ID, "model-pull", in.Model)
	apphost.JSON(w, http.StatusAccepted, job)
}
func (a *App) deleteModel(w http.ResponseWriter, r *http.Request) {
	if apphost.Current(r).User.Role != "owner" {
		apphost.Problem(w, 403, "Only the owner can remove shared models")
		return
	}
	var in struct {
		Model   string `json:"model"`
		Confirm bool   `json:"confirm"`
	}
	if !apphost.Decode(w, r, &in) {
		return
	}
	if !modelRE.MatchString(in.Model) || !in.Confirm {
		apphost.Problem(w, 400, "Confirm model removal first")
		return
	}
	a.mu.Lock()
	busy := len(a.running) > 0 || a.downloads.busy()
	a.mu.Unlock()
	if busy {
		apphost.Problem(w, 409, "Wait for all active turns and downloads before removing a shared model")
		return
	}
	p, e := a.provider(apphost.Current(r).User.ID, "ollama")
	if e != nil {
		apphost.Problem(w, 400, e.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	resp, e := request(ctx, p, "DELETE", "/api/delete", map[string]string{"model": in.Model})
	if e != nil {
		apphost.Problem(w, 502, e.Error())
		return
	}
	resp.Body.Close()
	a.H.Audit(apphost.Current(r).User.ID, "model-delete", in.Model)
	apphost.JSON(w, 200, map[string]bool{"deleted": true})
}
func officialModelLink(name string) string {
	return "https://ollama.com/library/" + url.PathEscape(strings.Split(name, ":")[0])
}
