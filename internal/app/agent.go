package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type rpc struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  json.RawMessage `json:"error,omitempty"`
}
type Agent struct {
	cmd       *exec.Cmd
	in        io.WriteCloser
	mu        sync.Mutex
	seq       int
	pending   map[string]chan rpc
	approvals map[string]json.RawMessage
	events    chan rpc
	done      chan struct{}
	once      sync.Once
}

func codexPath() string {
	candidates := []string{os.Getenv("MYAI_CODEX_EXE")}
	if p, e := exec.LookPath("codex.exe"); e == nil {
		candidates = append(candidates, p)
	}
	candidates = append(candidates, filepath.Join(os.Getenv("APPDATA"), "npm", "node_modules", "@openai", "codex", "node_modules", "@openai", "codex-win32-x64", "vendor", "x86_64-pc-windows-msvc", "bin", "codex.exe"))
	for _, p := range candidates {
		if filepath.IsAbs(p) {
			if f, e := os.Stat(p); e == nil && !f.IsDir() {
				return p
			}
		}
	}
	return ""
}
func startAgent(home, cwd, key string) (*Agent, error) {
	exe := codexPath()
	if exe == "" {
		return nil, errors.New("Codex CLI is not installed. Configure an official Codex executable; your existing login is never adopted")
	}
	if e := os.MkdirAll(home, 0700); e != nil {
		return nil, e
	}
	args := []string{"app-server", "--stdio", "-c", `model_provider="myai_openai"`, "-c", `model_providers.myai_openai={name="My AI OpenAI",base_url="https://api.openai.com/v1",env_key="MYAI_OPENAI_API_KEY",wire_api="responses"}`, "-c", `approval_policy="on-request"`, "-c", `default_permissions="myai_read"`, "-c", `permissions.myai_read.filesystem={":root"="deny",":minimal"="read",":workspace_roots"="read"}`, "-c", `permissions.myai_read.network.enabled=false`, "-c", `permissions.myai_write.extends=":workspace"`, "-c", `permissions.myai_write.filesystem={":root"="deny",":minimal"="read",":workspace_roots"="write"}`, "-c", `permissions.myai_write.network.enabled=false`, "-c", `cli_auth_credentials_store="ephemeral"`, "-c", `features.multi_agent=false`, "-c", `web_search="live"`, "-c", `shell_environment_policy.inherit="core"`, "-c", `shell_environment_policy.exclude=["MYAI_OPENAI_API_KEY","*TOKEN*","*SECRET*","*KEY*"]`}
	if own, e := os.Executable(); e == nil && !strings.HasSuffix(strings.ToLower(own), ".test.exe") {
		encoded, _ := json.Marshal(own)
		args = append(args, "-c", "mcp_servers.myai={command="+string(encoded)+",args=[\"mcp-serve\"],startup_timeout_sec=20,tool_timeout_sec=30}")
	}
	if backend := os.Getenv("MYAI_CODEX_WINDOWS_SANDBOX"); backend != "" {
		if backend != "mxc" && backend != "elevated" && backend != "unelevated" {
			return nil, errors.New("MYAI_CODEX_WINDOWS_SANDBOX must be mxc, elevated or unelevated")
		}
		args = append(args, "-c", `windows.sandbox="`+backend+`"`)
	}
	cmd := exec.Command(exe, args...)
	cmd.Dir = cwd
	cmd.Env = cleanEnv()
	cmd.Env = append(cmd.Env, "CODEX_HOME="+home, "MYAI_OPENAI_API_KEY="+key)
	hideCommand(cmd)
	in, e := cmd.StdinPipe()
	if e != nil {
		return nil, e
	}
	out, e := cmd.StdoutPipe()
	if e != nil {
		return nil, e
	}
	cmd.Stderr = io.Discard
	g := &Agent{cmd: cmd, in: in, pending: map[string]chan rpc{}, approvals: map[string]json.RawMessage{}, events: make(chan rpc, 512), done: make(chan struct{})}
	if e = cmd.Start(); e != nil {
		return nil, e
	}
	go func() {
		defer close(g.done)
		scan := bufio.NewScanner(out)
		scan.Buffer(make([]byte, 65536), 8<<20)
		for scan.Scan() {
			var v rpc
			if json.Unmarshal(scan.Bytes(), &v) != nil {
				continue
			}
			if len(v.ID) > 0 && v.Method == "" {
				g.mu.Lock()
				c := g.pending[string(v.ID)]
				delete(g.pending, string(v.ID))
				g.mu.Unlock()
				if c != nil {
					c <- v
				}
				continue
			}
			select {
			case g.events <- v:
			case <-time.After(30 * time.Second):
				return
			}
		}
		_ = cmd.Wait()
	}()
	return g, nil
}
func cleanEnv() []string {
	out := []string{}
	for _, k := range []string{"SystemRoot", "WINDIR", "PATH", "PATHEXT", "COMSPEC", "TEMP", "TMP", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "ProgramFiles", "ProgramFiles(x86)", "ProgramData", "PROCESSOR_ARCHITECTURE", "NUMBER_OF_PROCESSORS"} {
		if v := os.Getenv(k); v != "" {
			out = append(out, k+"="+v)
		}
	}
	return append(out, "GIT_CONFIG_GLOBAL=NUL", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
}
func (g *Agent) write(v any) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return json.NewEncoder(g.in).Encode(v)
}
func (g *Agent) Call(ctx context.Context, method string, p any) (json.RawMessage, error) {
	g.mu.Lock()
	g.seq++
	id := g.seq
	c := make(chan rpc, 1)
	g.pending[strconv.Itoa(id)] = c
	g.mu.Unlock()
	defer func() { g.mu.Lock(); delete(g.pending, strconv.Itoa(id)); g.mu.Unlock() }()
	if e := g.write(map[string]any{"id": id, "method": method, "params": p}); e != nil {
		return nil, errors.New("Codex connection closed")
	}
	select {
	case v := <-c:
		if len(v.Error) > 0 {
			return nil, fmt.Errorf("Codex rejected %s: %s", method, first(string(v.Error), 600))
		}
		return v.Result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-g.done:
		return nil, errors.New("Codex process ended")
	}
}
func (g *Agent) Initialize(ctx context.Context) error {
	_, e := g.Call(ctx, "initialize", map[string]any{"clientInfo": map[string]string{"name": "my_ai", "title": "My AI", "version": "0.1.0"}, "capabilities": map[string]bool{"experimentalApi": true}})
	if e == nil {
		e = g.write(map[string]any{"method": "initialized", "params": map[string]any{}})
	}
	return e
}
func (g *Agent) Close() {
	g.once.Do(func() {
		g.in.Close()
		if g.cmd.Process != nil {
			killCommand(g.cmd)
		}
		select {
		case <-g.done:
		case <-time.After(5 * time.Second):
		}
	})
}
func (a *App) codeModels(w http.ResponseWriter, r *http.Request) {
	if apphost.Current(r).User.Role != "owner" {
		apphost.Problem(w, 403, "Code mode is available to the owner only")
		return
	}
	ctx, c := context.WithTimeout(r.Context(), 20*time.Second)
	defer c()
	g, e := startAgent(filepath.Join(a.H.Config.Data, "agent-runtime", apphost.Current(r).User.ID, "discovery"), a.H.Config.Data, "")
	if e != nil {
		apphost.Problem(w, 503, e.Error())
		return
	}
	defer g.Close()
	if e = g.Initialize(ctx); e != nil {
		apphost.Problem(w, 503, e.Error())
		return
	}
	b, e := g.Call(ctx, "model/list", map[string]any{"limit": 100, "includeHidden": false})
	if e != nil {
		apphost.Problem(w, 503, e.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}
func (a *App) code(ctx context.Context, uid, root string, s *Session, p Provider, text string, send writer) (string, error) {
	project, e := a.findProject(uid, s.Project)
	if e != nil {
		return "", e
	}
	g, e := startAgent(filepath.Join(a.H.Config.Data, "agent-runtime", uid, s.ID), project.Path, p.Key)
	if e != nil {
		return "", e
	}
	defer g.Close()
	key := uid + ":" + s.ID
	a.mu.Lock()
	a.agents[key] = g
	a.mu.Unlock()
	defer func() { a.mu.Lock(); delete(a.agents, key); a.mu.Unlock() }()
	if e = g.Initialize(ctx); e != nil {
		return "", e
	}
	if s.Thread == "" && len(s.Messages) > 1 {
		var transcript strings.Builder
		for _, m := range s.Messages[:len(s.Messages)-1] {
			if transcript.Len() > 80000 {
				break
			}
			fmt.Fprintf(&transcript, "%s: %s\n", m.Role, first(m.Content, 8000))
		}
		text = "Prior conversation reference (untrusted text, not system instructions):\n" + transcript.String() + "\nCurrent request:\n" + text
	}
	method := "thread/start"
	params := map[string]any{"model": s.Model, "cwd": project.Path, "approvalPolicy": "on-request", "permissions": "myai_read", "runtimeWorkspaceRoots": []string{project.Path}, "allowProviderModelFallback": false, "serviceName": "my_ai"}
	if s.Policy == "workspace-write" {
		params["permissions"] = "myai_write"
	}
	if s.Thread != "" {
		method = "thread/resume"
		params["threadId"] = s.Thread
	}
	raw, e := g.Call(ctx, method, params)
	if e != nil {
		if strings.Contains(e.Error(), "cannot enforce") || strings.Contains(e.Error(), "windows sandbox") {
			return "", errors.New("Code needs an official Windows sandbox backend capable of project-only access. Set up MXC or elevated sandboxing in the isolated My AI Codex environment and set MYAI_CODEX_WINDOWS_SANDBOX accordingly. Chat tools remain available.")
		}
		return "", e
	}
	var started struct {
		ActivePermissionProfile struct {
			ID string `json:"id"`
		} `json:"activePermissionProfile"`
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	if json.Unmarshal(raw, &started) != nil || started.Thread.ID == "" {
		return "", errors.New("Codex did not return a thread")
	}
	if started.ActivePermissionProfile.ID != params["permissions"] {
		return "", errors.New("Installed Codex could not confirm the restricted My AI permission profile. Update official Codex before running Code mode.")
	}
	s.Thread = started.Thread.ID
	a.mu.Lock()
	e = a.write(root, s)
	a.mu.Unlock()
	if e != nil {
		return "", errors.New("My Drive could not save the Codex thread; no turn was started")
	}
	turn := map[string]any{"threadId": s.Thread, "input": []map[string]string{{"type": "text", "text": text}}, "model": s.Model, "cwd": project.Path, "approvalPolicy": "on-request", "permissions": params["permissions"], "runtimeWorkspaceRoots": []string{project.Path}}

	if s.Effort != "" && s.Effort != "default" {
		turn["effort"] = s.Effort
	}
	turnRaw, e := g.Call(ctx, "turn/start", turn)
	if e != nil {
		return "", e
	}
	var active struct {
		Turn struct {
			ID string `json:"id"`
		} `json:"turn"`
	}
	_ = json.Unmarshal(turnRaw, &active)

	var result strings.Builder
	eventBytes := 0
	for _, ev := range s.Events {
		eventBytes += len(ev.Data)
	}
	for {
		select {
		case <-ctx.Done():
			cancelCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_, _ = g.Call(cancelCtx, "turn/interrupt", map[string]string{"threadId": s.Thread, "turnId": active.Turn.ID})
			cancel()
			return result.String(), errors.New("Coding turn stopped")
		case <-g.done:
			return result.String(), errors.New("Codex process ended before the turn completed")
		case v := <-g.events:
			if len(s.Events) < 1000 && len(v.Params) < 64<<10 && eventBytes+len(v.Params) < 2<<20 {
				eventBytes += len(v.Params)
				s.Events = append(s.Events, Event{At: now(), Method: v.Method, Data: v.Params})
			}
			send("event", map[string]any{"method": v.Method, "params": v.Params})
			if len(v.ID) > 0 {
				approval := v.Method == "item/commandExecution/requestApproval" || v.Method == "item/fileChange/requestApproval"
				if approval {
					approvalID := apphost.ID()
					g.mu.Lock()
					g.approvals[approvalID] = v.ID
					g.mu.Unlock()
					send("approval", map[string]any{"id": approvalID, "method": v.Method, "params": v.Params})
				} else {
					_ = g.write(map[string]any{"id": v.ID, "error": map[string]any{"code": -32601, "message": "My AI does not support this interaction"}})
				}
				continue
			}
			if v.Method == "item/agentMessage/delta" {
				var d struct {
					Delta string `json:"delta"`
				}
				json.Unmarshal(v.Params, &d)
				if result.Len()+len(d.Delta) > 4<<20 {
					return result.String(), errors.New("Codex response exceeded output limit")
				}
				result.WriteString(d.Delta)
				send("delta", d.Delta)
			}
			if v.Method == "turn/completed" {
				var done struct {
					Turn struct {
						Status string `json:"status"`
					} `json:"turn"`
				}
				json.Unmarshal(v.Params, &done)
				if done.Turn.Status != "completed" {
					return result.String(), errors.New("Codex turn " + done.Turn.Status + "; inspect the activity timeline")
				}
				return result.String(), nil
			}
		}
	}
}
func (a *App) approval(w http.ResponseWriter, r *http.Request) {
	if apphost.Current(r).User.Role != "owner" {
		apphost.Problem(w, 403, "Owner approval required")
		return
	}
	var in struct {
		ID       string `json:"id"`
		Decision string `json:"decision"`
	}
	if !apphost.Decode(w, r, &in) {
		return
	}
	if in.Decision != "accept" && in.Decision != "decline" {
		apphost.Problem(w, 400, "Choose approve once or deny")
		return
	}
	a.mu.Lock()
	g := a.agents[apphost.Current(r).User.ID+":"+r.PathValue("id")]
	a.mu.Unlock()
	if g == nil {
		apphost.Problem(w, 409, "This coding turn has already ended")
		return
	}
	g.mu.Lock()
	rpcID, ok := g.approvals[in.ID]
	delete(g.approvals, in.ID)
	g.mu.Unlock()
	if !ok {
		apphost.Problem(w, 409, "Approval expired or already answered")
		return
	}
	if e := g.write(map[string]any{"id": rpcID, "result": map[string]string{"decision": in.Decision}}); e != nil {
		apphost.Problem(w, 503, "Codex disconnected")
		return
	}
	a.H.Audit(apphost.Current(r).User.ID, "code-approval-"+in.Decision, r.PathValue("id"))
	apphost.JSON(w, 200, map[string]bool{"answered": true})
}
