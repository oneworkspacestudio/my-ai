package app

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	benchassets "github.com/oneworkspacestudio/my-ai/tools/agent-bench"
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
)

type benchmarkState struct {
	ID      string         `json:"id"`
	State   string         `json:"state"`
	Phase   string         `json:"phase"`
	Started string         `json:"started,omitempty"`
	Updated string         `json:"updated,omitempty"`
	Results map[string]any `json:"results,omitempty"`
}
type modelBenchmark struct {
	mu         sync.Mutex
	state      benchmarkState
	cancel     context.CancelFunc
	done       chan struct{}
	path, root string
}

func (a *App) initBenchmark() error {
	root, err := safePath(a.H.Config.Data, "model-benchmark")
	if err != nil {
		return err
	}
	b := &modelBenchmark{root: root, path: filepath.Join(root, "state.json"), state: benchmarkState{State: "ready", Phase: "Test this server with Workspace running"}}
	if raw, e := os.ReadFile(b.path); e == nil {
		if json.Unmarshal(raw, &b.state) != nil {
			return errors.New("Model benchmark history is unreadable; preserve it before repair")
		}
		if b.state.State == "running" {
			b.state.State = "interrupted"
			b.state.Phase = "Server restarted. Previous test stopped; start a new test when ready"
			if e = atomic(b.path, b.state); e != nil {
				return e
			}
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	a.benchmark = b
	return nil
}
func (a *App) benchmarkBusy() bool {
	if a.benchmark == nil {
		return false
	}
	b := a.benchmark
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.cancel != nil
}
func (a *App) cancelBenchmark(reason string) {
	if a.benchmark == nil {
		return
	}
	b := a.benchmark
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cancel != nil {
		b.state.State = "interrupted"
		b.state.Phase = reason
		b.state.Updated = now()
		_ = atomic(b.path, b.state)
		b.cancel()
	}
}

func (a *App) waitBenchmark() {
	if a.benchmark == nil {
		return
	}
	b := a.benchmark
	b.mu.Lock()
	done := b.done
	b.mu.Unlock()
	if done != nil {
		select {
		case <-done:
		case <-time.After(30 * time.Second):
		}
	}
}

func benchmarkEndpoint() (string, error) {
	base := os.Getenv("MYAI_BENCHMARK_OLLAMA_URL")
	if base == "" {
		base = defaults()[2].Base
	}
	u, e := url.Parse(base)
	if e != nil || u.Scheme != "http" || !isLoopbackHost(u.Hostname()) || u.User != nil || u.RawQuery != "" || u.Fragment != "" || strings.Trim(u.Path, "/") != "" {
		return "", errors.New("Model tests require a local HTTP Ollama endpoint")
	}
	return strings.TrimRight(base, "/"), nil
}
func (a *App) benchmarkStatus(w http.ResponseWriter, r *http.Request) {
	b := a.benchmark
	b.mu.Lock()
	defer b.mu.Unlock()
	a.readBenchmarkResult(b)
	apphost.JSON(w, 200, b.state)
}
func (a *App) readBenchmarkResult(b *modelBenchmark) {
	if !idRE.MatchString(b.state.ID) {
		return
	}
	p, e := safePath(b.root, b.state.ID, "results", "resource-results.json")
	if e != nil {
		return
	}
	f, e := os.Open(p)
	if e != nil {
		return
	}
	defer f.Close()
	var result map[string]any
	if json.NewDecoder(io.LimitReader(f, 2<<20)).Decode(&result) == nil {
		b.state.Results = result
	}
}
func (a *App) benchmarkStart(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Models  []string `json:"models"`
		Reserve float64  `json:"reserveGB"`
		Quick   bool     `json:"quick"`
	}
	if !apphost.Decode(w, r, &request) {
		return
	}
	if request.Reserve < 0 || request.Reserve > 1024 || len(request.Models) > 8 {
		apphost.Problem(w, 400, "Choose at most8 models and a valid memory reserve")
		return
	}
	if a.ollamaMaintenanceBusy() {
		apphost.Problem(w, 409, "Wait for the Ollama upgrade to finish")
		return
	}
	if a.downloads != nil {
		for _, j := range a.downloads.list() {
			if j.State == "queued" || j.State == "downloading" {
				apphost.Problem(w, 409, "Wait for model downloads to finish before measuring this server")
				return
			}
		}
	}
	a.mu.Lock()
	working := len(a.running) > 0
	parent := a.upgradeContext
	a.mu.Unlock()
	if working {
		apphost.Problem(w, 409, "Let the current My AI request finish before testing models")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	models, e := benchmarkModels(ctx, a.client, request.Models)
	if e != nil {
		apphost.Problem(w, 400, e.Error())
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.running) > 0 || (a.downloads != nil && a.downloads.busy()) {
		apphost.Problem(w, 409, "Wait for active requests and model downloads before testing")
		return
	}
	b := a.benchmark
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cancel != nil {
		apphost.Problem(w, 409, "A model test is already running")
		return
	}
	if parent == nil {
		parent = context.Background()
	}
	runCtx, stop := context.WithCancel(parent)
	b.state = benchmarkState{ID: apphost.ID(), State: "running", Phase: "Preparing the included runtime and checking server headroom", Started: now(), Updated: now()}
	if e = atomic(b.path, b.state); e != nil {
		stop()
		apphost.Problem(w, 500, "Could not save the server test")
		return
	}
	b.cancel = stop
	b.done = make(chan struct{})
	go a.runBenchmark(runCtx, models, request.Reserve, request.Quick)
	apphost.JSON(w, 202, b.state)
}
func benchmarkModels(ctx context.Context, client *http.Client, wanted []string) ([]string, error) {
	base, e := benchmarkEndpoint()
	if e != nil {
		return nil, e
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", base+"/api/tags", nil)
	resp, e := client.Do(req)
	if e != nil {
		return nil, errors.New("Start the local Ollama server first")
	}
	defer resp.Body.Close()
	var tags struct {
		Models []struct {
			Name         string   `json:"name"`
			Size         int64    `json:"size"`
			Capabilities []string `json:"capabilities"`
		} `json:"models"`
	}
	if resp.StatusCode != 200 || json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&tags) != nil {
		return nil, errors.New("Could not read installed local models")
	}
	installed := map[string]int64{}
	for _, m := range tags.Models {
		if modelRE.MatchString(m.Name) && m.Size > 0 && m.Size <= 12<<30 && !strings.Contains(strings.ToLower(m.Name), "embed") {
			installed[m.Name] = m.Size
		}
	}
	if len(wanted) > 0 {
		seen := map[string]bool{}
		for _, m := range wanted {
			if !modelRE.MatchString(m) || installed[m] == 0 || seen[m] {
				return nil, errors.New("Choose unique installed local model tags, each at most12GiB")
			}
			seen[m] = true
		}
		return wanted, nil
	}
	order := []string{"qwen3.5:0.8b", "qwen3.5:2b", "qwen3.5:4b", "qwen3.5:9b", "ministral-3:3b", "granite4:3b"}
	out := []string{}
	seen := map[string]bool{}
	for _, m := range order {
		if installed[m] > 0 {
			out = append(out, m)
			seen[m] = true
		}
	}
	remaining := []string{}
	for m := range installed {
		if !seen[m] {
			remaining = append(remaining, m)
		}
	}
	sort.Slice(remaining, func(i, j int) bool { return installed[remaining[i]] < installed[remaining[j]] })
	for _, m := range remaining {
		if len(out) == 8 {
			break
		}
		out = append(out, m)
	}
	if len(out) == 0 {
		return nil, errors.New("Download a small local model in Models first")
	}
	return out, nil
}
func (a *App) benchmarkStop(w http.ResponseWriter, r *http.Request) {
	a.cancelBenchmark("Stopped by the server owner")
	a.waitBenchmark()
	apphost.JSON(w, 200, map[string]bool{"stopped": true})
}
func (a *App) runBenchmark(ctx context.Context, models []string, reserve float64, quick bool) {
	b := a.benchmark
	b.mu.Lock()
	done := b.done
	b.mu.Unlock()
	defer close(done)
	finish := func(state, phase string) {
		b.mu.Lock()
		defer b.mu.Unlock()
		a.readBenchmarkResult(b)
		if b.state.State == "running" {
			b.state.State = state
			b.state.Phase = phase
		}
		b.state.Updated = now()
		b.cancel = nil
		_ = atomic(b.path, b.state)
	}
	if _, e := a.searchConnection(ctx); e != nil {
		finish("failed", "The included runtime could not start; reinstall My AI if this persists")
		return
	}
	a.searchMu.Lock()
	runtime := a.search
	a.searchMu.Unlock()
	python, e := runtime.Python(ctx)
	if e != nil {
		finish("failed", "The included Python runtime is unavailable")
		return
	}
	b.mu.Lock()
	id := b.state.ID
	b.mu.Unlock()
	root, e := safePath(b.root, id)
	if e != nil || os.MkdirAll(root, 0700) != nil {
		finish("failed", "Could not prepare private benchmark storage")
		return
	}
	for _, name := range []string{"bench.py", "summarize.py", "resources.py", "safe_bench.py"} {
		raw, e := benchassets.Files.ReadFile(name)
		if e != nil {
			finish("failed", "Included benchmark is missing")
			return
		}
		if e = os.WriteFile(filepath.Join(root, name), raw, 0600); e != nil {
			finish("failed", "Could not prepare the benchmark")
			return
		}
	}
	args := []string{"-I", "-B", filepath.Join(root, "safe_bench.py"), "--out", filepath.Join(root, "results"), "--reserve-gb", strconv.FormatFloat(reserve, 'f', 2, 64), "--models"}
	args = append(args, models...)
	endpoint, err := benchmarkEndpoint()
	if err != nil {
		finish("failed", err.Error())
		return
	}
	args = append(args, "--base", endpoint)
	if quick {
		args = append(args, "--quick")
	}
	if health := os.Getenv("MYAI_WORKSPACE_URL"); health != "" {
		u, err := url.Parse(health)
		if err != nil || u.Scheme != "http" || !isLoopbackHost(u.Hostname()) || u.User != nil {
			finish("failed", "Workspace health monitoring needs the configured local manager")
			return
		}
		u.Path = "/healthz"
		u.RawQuery = ""
		u.Fragment = ""
		args = append(args, "--health", u.String())
	}
	cmd := exec.CommandContext(ctx, python, args...)
	cmd.Dir = root
	for _, k := range []string{"SystemRoot", "WINDIR", "SystemDrive", "COMSPEC", "PATH", "PATHEXT", "NUMBER_OF_PROCESSORS", "PROCESSOR_ARCHITECTURE", "ProgramFiles"} {
		if v := os.Getenv(k); v != "" {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	cmd.Env = append(cmd.Env, "PYTHONUTF8=1", "PYTHONIOENCODING=utf-8")
	hideCommand(cmd)
	cmd.Cancel = func() error { killCommand(cmd); return nil }
	cmd.WaitDelay = 10 * time.Second
	out, e := cmd.StdoutPipe()
	if e != nil {
		finish("failed", "Could not start benchmark output")
		return
	}
	log, logErr := os.OpenFile(filepath.Join(root, "runner.log"), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if logErr != nil {
		finish("failed", "Could not prepare private benchmark diagnostics")
		return
	}
	defer log.Close()
	cmd.Stderr = log
	if e = cmd.Start(); e != nil {
		finish("failed", "Could not start the included benchmark")
		return
	}
	s := bufio.NewScanner(out)
	s.Buffer(make([]byte, 4096), 1<<20)
	activeModel := ""
	for s.Scan() {
		var event struct{ Phase, Model, Reason string }
		if json.Unmarshal(s.Bytes(), &event) != nil || event.Phase == "" {
			continue
		}
		if event.Phase == "testing" && event.Model != "" {
			activeModel = event.Model
		} else if event.Phase == "completed" || event.Phase == "stopped" || event.Phase == "incomplete" {
			activeModel = ""
		}
		b.mu.Lock()
		if b.state.State == "running" {
			b.state.Phase = event.Phase
			if event.Model != "" {
				b.state.Phase += " · " + event.Model
			}
			if event.Reason != "" {
				b.state.Phase += " · " + event.Reason
			}
			b.state.Updated = now()
			a.readBenchmarkResult(b)
			_ = atomic(b.path, b.state)
		}
		b.mu.Unlock()
	}
	e = cmd.Wait()
	if ctx.Err() != nil {
		// A killed Python client may leave its inference allocation in Ollama.
		// Foreground callers wait for this cleanup before starting their own turn.
		if activeModel != "" {
			unloadCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
			raw, _ := json.Marshal(map[string]any{"model": activeModel, "keep_alive": 0, "stream": false})
			req, _ := http.NewRequestWithContext(unloadCtx, "POST", endpoint+"/api/generate", bytes.NewReader(raw))
			req.Header.Set("Content-Type", "application/json")
			if resp, err := a.client.Do(req); err == nil {
				resp.Body.Close()
			}
			stop()
		}
		finish("interrupted", "Test interrupted; existing models and preferences are retained")
	} else if e != nil {
		finish("failed", "The test could not complete. Download the evidence or try again when the server is quieter")
	} else {
		finish("completed", "Finished. Compare task success and resource headroom below")
	}
}
func isLoopbackHost(host string) bool {
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}
func (a *App) benchmarkExport(w http.ResponseWriter, r *http.Request) {
	b := a.benchmark
	b.mu.Lock()
	id := b.state.ID
	busy := b.cancel != nil
	b.mu.Unlock()
	if busy || !idRE.MatchString(id) {
		apphost.Problem(w, 409, "Wait for the test to finish before downloading evidence")
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=my-ai-model-test.zip")
	z := zip.NewWriter(w)
	defer z.Close()
	for _, name := range []string{"resource-results.json", "run.json", "samples.jsonl", "summary.json"} {
		p, e := safePath(b.root, id, "results", name)
		if e != nil {
			continue
		}
		f, e := os.Open(p)
		if e != nil {
			continue
		}
		entry, e := z.Create(name)
		if e == nil {
			_, _ = io.Copy(entry, io.LimitReader(f, 32<<20))
		}
		f.Close()
	}
}
