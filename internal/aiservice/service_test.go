package aiservice

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func fixture() *Service {
	return &Service{settings: Settings{Enabled: true, Model: "fixture:9b"}, wake: make(chan struct{}, 1), managed: map[string]string{}}
}
func awaitQueue(t *testing.T, s *Service, n int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		got := len(s.waiting)
		s.mu.Unlock()
		if got == n {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("queue did not reach %d", n)
}
func TestQueueSerialFIFOAndCancellation(t *testing.T) {
	s := fixture()
	first, code := s.acquire(context.Background(), "one", "alice", true)
	if code != 0 {
		t.Fatal(code)
	}
	second := make(chan *ticket, 1)
	go func() {
		v, c := s.acquire(context.Background(), "two", "bob", true)
		if c == 0 {
			second <- v
		}
	}()
	awaitQueue(t, s, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancelled := make(chan int, 1)
	go func() { _, c := s.acquire(ctx, "three", "carol", true); cancelled <- c }()
	awaitQueue(t, s, 2)
	cancel()
	if <-cancelled != 408 {
		t.Fatal("cancelled admission")
	}
	awaitQueue(t, s, 1)
	select {
	case <-second:
		t.Fatal("parallel admission")
	default:
	}
	s.release(first, true)
	select {
	case next := <-second:
		s.release(next, true)
	case <-time.After(time.Second):
		t.Fatal("next request not admitted")
	}
	if s.completed != 2 || s.active != nil || len(s.waiting) != 0 {
		t.Fatal("queue did not drain")
	}
}
func TestQueuePerUserAndGlobalBounds(t *testing.T) {
	s := fixture()
	first, _ := s.acquire(context.Background(), "myapp", "alice", true)
	defer s.release(first, true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for i := 0; i < MaxPerUser-1; i++ {
		go s.acquire(ctx, "myapp", "alice", true)
	}
	awaitQueue(t, s, MaxPerUser-1)
	if _, c := s.acquire(ctx, "myapp", "alice", true); c != 429 {
		t.Fatal(c)
	}
	s.mu.Lock()
	s.waiting = make([]*ticket, MaxQueue)
	for i := range s.waiting {
		s.waiting[i] = &ticket{app: "other", user: "other", ctx: ctx, cancel: func() {}}
	}
	s.mu.Unlock()
	if _, c := s.acquire(ctx, "new", "bob", true); c != 429 {
		t.Fatal(c)
	}
	// Restore synthetic placeholders before releasing the active ticket.
	s.mu.Lock()
	s.waiting = nil
	s.mu.Unlock()
}
func TestInferencePinsModelAndStreamsWithoutPersisting(t *testing.T) {
	s := fixture()
	var calls atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/api/chat" {
			t.Error(r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["keep_alive"] != float64(-1) || body["model"] != "fixture:9b" {
			t.Error(body)
		}
		if _, ok := body["options"].(map[string]any)["num_gpu"]; ok {
			t.Error("untrusted hardware override")
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Write([]byte("{\"message\":{\"content\":\"fixture\"},\"done\":true}\n"))
	}))
	defer backend.Close()
	s.backend = backend.URL
	s.client = backend.Client()
	request := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/chat", strings.NewReader(body))
		w := httptest.NewRecorder()
		s.infer(w, r, "myapp", "alice", func() bool { return true })
		return w
	}
	if w := request(`{"model":"fixture:9b","messages":[],"keep_alive":0,"options":{"num_gpu":99,"num_ctx":8192}}`); w.Code != 200 || !strings.Contains(w.Body.String(), "fixture") {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := request(`{"model":"other:9b"}`); w.Code != 409 {
		t.Fatal(w.Code)
	}
	s.settings.Enabled = false
	if w := request(`{"model":"fixture:9b"}`); w.Code != 503 {
		t.Fatal(w.Code)
	}
	if calls.Load() != 1 {
		t.Fatal("unexpected backend call")
	}
}
func TestRevocationCancelsWaitingRequest(t *testing.T) {
	s := fixture()
	first, _ := s.acquire(context.Background(), "one", "alice", true)
	defer s.release(first, true)
	done := make(chan int, 1)
	go func() {
		r := httptest.NewRequest("POST", "/api/chat", strings.NewReader(`{"messages":[]}`))
		w := httptest.NewRecorder()
		s.infer(w, r, "two", "bob", func() bool { return false })
		done <- w.Code
	}()
	awaitQueue(t, s, 1)
	select {
	case c := <-done:
		if c != 408 {
			t.Fatal(c)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("revocation did not cancel queued inference")
	}
	awaitQueue(t, s, 0)
}
