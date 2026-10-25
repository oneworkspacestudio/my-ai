package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	syncatomic "sync/atomic"
	"testing"
	"time"

	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
)

func awaitDownload(t *testing.T, q *downloadQueue, id, state string) Download {
	t.Helper()
	until := time.Now().Add(8 * time.Second)
	for time.Now().Before(until) {
		for _, j := range q.list() {
			if j.ID == id && j.State == state {
				return j
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("job did not reach %s: %+v", state, q.list())
	return Download{}
}
func TestDownloadSurvivesRestartAndSharesQueue(t *testing.T) {
	var requests syncatomic.Int32
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requests.Add(1)
		fmt.Fprintln(w, `{"status":"pulling layer","digest":"fixture","total":1000,"completed":250}`)
		w.(http.Flusher).Flush()
		if n == 1 {
			close(started)
			<-r.Context().Done()
			return
		}
		fmt.Fprintln(w, `{"status":"success"}`)
	}))
	defer server.Close()
	root := t.TempDir()
	q, e := newDownloadQueue(root)
	if e != nil {
		t.Fatal(e)
	}
	q.provider = Provider{Base: server.URL}
	j, e := q.add("fixture:tiny")
	if e != nil {
		t.Fatal(e)
	}
	same, _ := q.add("fixture:tiny")
	if same.ID != j.ID || len(q.list()) != 1 {
		t.Fatal("shared queue did not deduplicate")
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { q.run(ctx); close(done) }()
	<-started
	awaitDownload(t, q, j.ID, "downloading")
	cancel()
	<-done
	awaitDownload(t, q, j.ID, "queued")
	restarted, e := newDownloadQueue(root)
	if e != nil {
		t.Fatal(e)
	}
	restarted.provider = Provider{Base: server.URL}
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	go restarted.run(ctx2)
	completed := awaitDownload(t, restarted, j.ID, "completed")
	if completed.Total != 1000 || completed.Completed != 1000 || requests.Load() != 2 {
		t.Fatal(completed, requests.Load())
	}
	b, _ := os.ReadFile(restarted.path)
	if !bytes.Contains(b, []byte(`"completed"`)) {
		t.Fatal("completion not persisted")
	}
}
func TestDownloadRealUpstreamErrorAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"status":"pulling manifest"}`)
		fmt.Fprintln(w, `{"error":"model fixture does not exist"}`)
	}))
	defer server.Close()
	q, _ := newDownloadQueue(t.TempDir())
	q.provider = Provider{Base: server.URL}
	ctx, c := context.WithCancel(context.Background())
	defer c()
	go q.run(ctx)
	job, _ := q.add("missing:fixture")
	failed := awaitDownload(t, q, job.ID, "failed")
	if !strings.Contains(failed.Error, "does not exist") {
		t.Fatal("upstream failure hidden", failed)
	}
	q2, _ := newDownloadQueue(t.TempDir())
	other, _ := q2.add("cancel:fixture")
	if e := q2.stop(other.ID); e != nil {
		t.Fatal(e)
	}
	awaitDownload(t, q2, other.ID, "cancelled")
	time.Sleep(30 * time.Millisecond)
	awaitDownload(t, q2, other.ID, "cancelled")
}
func TestDownloadConcurrentRequestsDeduplicate(t *testing.T) {
	q, _ := newDownloadQueue(t.TempDir())
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := q.add("shared:one"); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	if len(q.list()) != 1 {
		t.Fatal("more than one server job")
	}
}
func TestChatToolRoundTripsAllProviders(t *testing.T) {
	for _, provider := range []string{"openai", "deepseek", "ollama"} {
		t.Run(provider, func(t *testing.T) {
			count := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				count++
				var body map[string]any
				json.NewDecoder(r.Body).Decode(&body)
				raw, _ := json.Marshal(body)
				if count == 1 {
					if body["tools"] == nil || !bytes.Contains(raw, []byte("Current server time")) {
						t.Error("tools/clock not provided")
					}
					switch provider {
					case "openai":
						fmt.Fprint(w, "data: {\"type\":\"response.output_item.done\",\"item\":{\"type\":\"function_call\",\"call_id\":\"call1\",\"name\":\"calculate\",\"arguments\":\"{\\\"expression\\\":\\\"6*7\\\"}\"}}\n\ndata: {\"type\":\"response.completed\"}\n\n")
					case "deepseek":
						fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call1\",\"function\":{\"name\":\"calculate\",\"arguments\":\"{\\\"expression\\\":\\\"6*7\\\"}\"}}]}}]}\n\ndata: [DONE]\n\n")
					case "ollama":
						fmt.Fprintln(w, `{"message":{"tool_calls":[{"function":{"name":"calculate","arguments":{"expression":"6*7"}}}]},"done":true}`)
					}
					return
				}
				if count != 2 || !bytes.Contains(raw, []byte("42")) {
					t.Error("actual result not returned to model", string(raw))
				}
				switch provider {
				case "openai":
					fmt.Fprint(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"42\"}\n\ndata: {\"type\":\"response.completed\"}\n\n")
				case "deepseek":
					fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"42\"}}]}\n\ndata: [DONE]\n\n")
				case "ollama":
					fmt.Fprintln(w, `{"message":{"content":"42"},"done":true}`)
				}
			}))
			defer server.Close()
			s := &Session{Model: "fixture", Messages: []Message{{Role: "user", Content: "six times seven"}}}
			out, e := (&App{}).chat(context.Background(), Provider{ID: provider, Base: server.URL}, s, func(string, any) {})
			if e != nil || out != "42" || count != 2 || len(s.Events) != 2 {
				t.Fatal(out, e, count, s.Events)
			}
		})
	}
}
func TestToolNetworkAndPrivateScopeBoundaries(t *testing.T) {
	for _, raw := range []string{"http://127.0.0.1/", "http://[::1]/", "http://[::ffff:127.0.0.1]/", "http://169.254.169.254/", "http://100.64.0.1/", "http://10.1.2.3/", "https://localhost/", "file:///secret", "https://example.com:8443/", "https://user:password@example.com/"} {
		if _, e := validateWebURL(raw); e == nil {
			t.Error("unsafe URL accepted", raw)
		}
	}
	if publicIP(net.ParseIP("192.168.1.5")) || !publicIP(net.ParseIP("1.1.1.1")) {
		t.Fatal("IP classification")
	}
	root := t.TempDir()
	a := &App{}
	s := Session{ID: apphost.ID(), Title: "private", Mode: "chat", Policy: "read-only", Provider: "ollama", Messages: []Message{{Role: "user", Content: "OWN_SCOPE_ONLY"}}}
	if e := a.write(root, &s); e != nil {
		t.Fatal(e)
	}
	v, e := a.executeTool(context.Background(), toolScope{Root: root}, "read_conversation", map[string]string{"id": s.ID})
	if e != nil || !strings.Contains(fmt.Sprint(v), "OWN_SCOPE_ONLY") {
		t.Fatal(v, e)
	}
	if _, e = a.executeTool(context.Background(), toolScope{Root: t.TempDir()}, "read_conversation", map[string]string{"id": s.ID}); e == nil {
		t.Fatal("cross-user read succeeded")
	}
	if _, e = a.executeTool(context.Background(), toolScope{Root: root}, "read_attachment", map[string]string{"id": "../../credentials"}); e == nil {
		t.Fatal("traversal accepted")
	}
}
func TestTimeCalculationMCPAndHTML(t *testing.T) {
	a := &App{}
	v, e := a.executeTool(context.Background(), toolScope{}, "get_time", map[string]string{"timezone": "Europe/London"})
	if e != nil || !strings.Contains(fmt.Sprint(v), time.Now().In(time.Local).Format("2006-01-02")) {
		t.Fatal(v, e)
	}
	if v, e := arithmetic("(12+3)*4/2"); e != nil || v != 30 {
		t.Fatal(v, e)
	}
	for _, expr := range []string{"os.Remove(1)", "1/0", "1<<64", "x+1"} {
		if _, e := arithmetic(expr); e == nil {
			t.Fatal("unsafe arithmetic accepted", expr)
		}
	}
	if pageText("<html><script>secret()</script><p>Real text</p></html>", 100) != "Real text" {
		t.Fatal("HTML extraction")
	}
	var output bytes.Buffer
	input := `{"jsonrpc":"2.0","id":1,"method":"initialize"}` + "\n" + `{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n" + `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"calculate","arguments":{"expression":"6*7"}}}` + "\n"
	if e = RunMCP(strings.NewReader(input), &output); e != nil || !strings.Contains(output.String(), "42") || strings.Contains(output.String(), "read_conversation") {
		t.Fatal(output.String(), e)
	}
	credential := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(credential, []byte("secret"), 0600)
	if _, e = boundedTextFile(credential); e == nil {
		t.Fatal("credential file read")
	}
}

func TestModelTagNormalization(t *testing.T) {
	for raw, want := range map[string]string{"qwen3": "qwen3:latest", "https://ollama.com/library/qwen3:0.6b": "qwen3:0.6b", "https://ollama.com/baytout3/qwen3.5-uncensored": "baytout3/qwen3.5-uncensored:latest", "ollama run qwen3:0.6b": "qwen3:0.6b", "ollama pull llama3.2": "llama3.2:latest"} {
		got, e := normalizeModelTag(raw)
		if e != nil || got != want {
			t.Fatal(raw, got, want, e)
		}
	}
	for _, bad := range []string{"https://evil.test/model", "ollama run x --verbose", "../x", "qwen;whoami"} {
		if _, e := normalizeModelTag(bad); e == nil {
			t.Fatal("invalid tag accepted", bad)
		}
	}
}
