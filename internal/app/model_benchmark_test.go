package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
)

func TestBenchmarkHistoryRestartDoesNotResumeLoad(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "model-benchmark", "state.json")
	if e := atomic(path, benchmarkState{ID: strings.Repeat("a", 48), State: "running", Phase: "testing"}); e != nil {
		t.Fatal(e)
	}
	a := &App{H: &apphost.Host{Config: apphost.Config{Data: root}}}
	if e := a.initBenchmark(); e != nil {
		t.Fatal(e)
	}
	if a.benchmarkBusy() || a.benchmark.state.State != "interrupted" {
		t.Fatal(a.benchmark.state)
	}
}
func TestBenchmarkExportCannotEscapePrivateRuns(t *testing.T) {
	root := t.TempDir()
	a := &App{benchmark: &modelBenchmark{root: root, state: benchmarkState{ID: "../../owner"}}}
	w := httptest.NewRecorder()
	a.benchmarkExport(w, httptest.NewRequest("GET", "/api/model-benchmark/export", nil))
	if w.Code != 409 {
		t.Fatal(w.Code)
	}
}
func TestBenchmarkCancellationPersistsAndCancelsOnlyWorker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	root := t.TempDir()
	a := &App{benchmark: &modelBenchmark{root: root, path: filepath.Join(root, "state.json"), cancel: cancel, state: benchmarkState{State: "running"}}}
	a.cancelBenchmark("Normal chat requested")
	if ctx.Err() == nil || a.benchmark.state.State != "interrupted" {
		t.Fatal("worker not interrupted")
	}
	if _, e := os.Stat(filepath.Join(root, "state.json")); e != nil {
		t.Fatal(e)
	}
}

type benchRoundTrip func(*http.Request) (*http.Response, error)

func (f benchRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestBenchmarkModelsUsesOnlyInstalledLocalCandidates(t *testing.T) {
	c := &http.Client{Transport: benchRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "127.0.0.1:11434" || r.URL.Path != "/api/tags" {
			t.Fatal(r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"models":[{"name":"qwen3.5:9b","size":6500000000},{"name":"nomic-embed-text:latest","size":300000000},{"name":"huge:latest","size":90000000000}]}`)), Header: http.Header{}}, nil
	})}
	models, e := benchmarkModels(context.Background(), c, nil)
	if e != nil || len(models) != 1 || models[0] != "qwen3.5:9b" {
		t.Fatal(models, e)
	}
	if _, e = benchmarkModels(context.Background(), c, []string{"missing:latest"}); e == nil {
		t.Fatal("uninstalled model accepted")
	}
}

func TestBenchmarkEndpointRejectsRemoteAndCredentialledURLs(t *testing.T) {
	for _, base := range []string{"https://example.com", "http://127.0.0.1.evil.test", "http://owner:password@localhost:11434", "http://127.0.0.1:11434/private", "http://127.0.0.1:11434?key=secret"} {
		t.Setenv("MYAI_BENCHMARK_OLLAMA_URL", base)
		if _, err := benchmarkEndpoint(); err == nil {
			t.Fatal("accepted non-local benchmark endpoint")
		}
	}
	t.Setenv("MYAI_BENCHMARK_OLLAMA_URL", "http://127.0.0.1:40190")
	if base, err := benchmarkEndpoint(); err != nil || base != "http://127.0.0.1:40190" {
		t.Fatal(base, err)
	}
}

func TestBenchmarkRechecksMaintenanceAfterModelDiscovery(t *testing.T) {
	a := &App{H: &apphost.Host{Config: apphost.Config{Data: t.TempDir()}}, ollamaUpdate: &ollamaUpgrade{}, running: map[string]context.CancelFunc{}}
	if err := a.initBenchmark(); err != nil {
		t.Fatal(err)
	}
	a.client = &http.Client{Transport: benchRoundTrip(func(r *http.Request) (*http.Response, error) {
		// Another owner tab begins maintenance while tag discovery is in flight.
		a.ollamaUpdate.mu.Lock()
		a.ollamaUpdate.Busy = true
		a.ollamaUpdate.mu.Unlock()
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"models":[{"name":"qwen3.5:9b","size":6500000000}]}`)), Header: http.Header{}}, nil
	})}
	w := httptest.NewRecorder()
	a.benchmarkStart(w, httptest.NewRequest("POST", "/api/model-benchmark", strings.NewReader(`{"models":["qwen3.5:9b"]}`)))
	if w.Code != 409 || a.benchmarkBusy() {
		t.Fatal("benchmark overlapped a confirmed runtime upgrade", w.Code)
	}
}
