package aiservice

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
)

func TestRemoteGrantsAreExplicitAndRevocable(t *testing.T) {
	h, err := apphost.New(apphost.Config{ID: "aiservice", Name: "AI Service", Version: "test", Data: filepath.Join(t.TempDir(), "data"), Listen: "127.0.0.1:28942", Origin: "http://localhost:28942"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { h.Close() })
	s, err := New(h)
	if err != nil {
		t.Fatal(err)
	}
	setup := httptest.NewRequest("POST", h.Config.Origin+"/api/setup", strings.NewReader(`{"email":"parent@example.test","password":"synthetic-test-password"}`))
	setup.RemoteAddr = "127.0.0.1:1234"
	setup.Header.Set("Origin", h.Config.Origin)
	setup.Header.Set("X-Setup-Token", strings.Split(h.SetupURL(), "#setup/")[1])
	w := httptest.NewRecorder()
	h.Handler().ServeHTTP(w, setup)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	cookie := w.Result().Cookies()[0]
	call := func(method, route, body, csrf string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, h.Config.Origin+route, strings.NewReader(body))
		r.RemoteAddr = "127.0.0.1:1234"
		r.AddCookie(cookie)
		r.Header.Set("Origin", h.Config.Origin)
		r.Header.Set("X-CSRF-Token", csrf)
		result := httptest.NewRecorder()
		h.Handler().ServeHTTP(result, r)
		return result
	}
	var session apphost.Session
	json.Unmarshal(call("GET", "/api/me", "", "").Body.Bytes(), &session)
	issue := func(id string, remote bool) string {
		body, _ := json.Marshal(map[string]any{"id": id, "remote": remote})
		result := call("POST", "/api/service/clients", string(body), session.CSRF)
		if result.Code != 201 {
			t.Fatal(result.Code, result.Body.String())
		}
		var value map[string]any
		json.Unmarshal(result.Body.Bytes(), &value)
		return value["key"].(string)
	}
	local := issue("local-app", false)
	remote := issue("wonderpal", true)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "fixture:9b" || body["keep_alive"] != float64(-1) {
			t.Error(body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":{"content":"synthetic reply"},"done":true}`))
	}))
	defer backend.Close()
	s.backend = backend.URL
	s.client = backend.Client()
	s.settings = Settings{Enabled: true, Model: "fixture:9b"}
	request := func(key, origin string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", h.Config.Origin+"/api/client/chat", strings.NewReader(`{"messages":[{"role":"user","content":"synthetic question"}],"stream":false}`))
		r.RemoteAddr = "198.51.100.23:4567"
		r.Header.Set("Authorization", "Bearer "+key)
		r.Header.Set("Origin", origin)
		result := httptest.NewRecorder()
		h.Handler().ServeHTTP(result, r)
		return result
	}
	if w := request(local, ""); w.Code != 401 {
		t.Fatal("local credential gained remote access", w.Code)
	}
	if w := request(remote, "https://browser.example.test"); w.Code != 401 {
		t.Fatal("browser origin accepted", w.Code)
	}
	if w := request(remote, ""); w.Code != 200 || !strings.Contains(w.Body.String(), "synthetic reply") {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := call("DELETE", "/api/service/clients/wonderpal", "", session.CSRF); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := request(remote, ""); w.Code != 401 {
		t.Fatal("revoked grant accepted", w.Code)
	}
	newLocal := issue("wonderpal", false)
	if w := request(newLocal, ""); w.Code != 401 {
		t.Fatal("remote grant survived replacement", w.Code)
	}
	if s.completed != 1 {
		t.Fatal("remote request did not share inference accounting")
	}
}
