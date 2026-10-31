package app

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOllamaUpgradeVersionOrder(t *testing.T) {
	for _, v := range []struct {
		a, b string
		want bool
	}{{"0.40.13", "0.40.11", true}, {"0.40.13", "0.40.11+cpu", true}, {"0.40.13", "0.40.13", false}, {"0.40.9", "0.40.13", false}, {"0.40.13", "invalid", false}} {
		if newerOllama(v.a, v.b) != v.want {
			t.Fatalf("wrong order: %s %s", v.a, v.b)
		}
	}
}
func TestOfficialOllamaInstallerMetadata(t *testing.T) {
	m := map[string]any{"tag_name": "v0.40.13", "assets": []any{map[string]any{"name": "OllamaSetup.exe", "size": 123, "digest": "sha256:" + strings.Repeat("a", 64), "browser_download_url": "https://github.com/ollama/ollama/releases/download/v0.40.13/OllamaSetup.exe"}}}
	raw, _ := json.Marshal(m)
	if _, e := officialOllamaAsset(raw); e != nil {
		t.Fatal(e)
	}
	for _, bad := range []string{strings.ReplaceAll(string(raw), "github.com/ollama/ollama", "evil.example/ollama/ollama"), strings.ReplaceAll(string(raw), strings.Repeat("a", 64), "bad"), strings.ReplaceAll(string(raw), "v0.40.13", "v0.40.13-rc.1")} {
		if _, e := officialOllamaAsset([]byte(bad)); e == nil {
			t.Fatal("accepted unverified installer")
		}
	}
}
func TestVersionRequirementExplainsUpgradeWithoutLeakingPrompt(t *testing.T) {
	e := providerFailure("ollama", 500, []byte(`{"error":"model requires ollama version v0.40.13 or newer (current version is v0.40.11) PRIVATE_PROMPT"}`))
	if !strings.Contains(e.Error(), "Update Ollama") || strings.Contains(e.Error(), "PRIVATE_PROMPT") {
		t.Fatal(e)
	}
}

type upgradeTransport func(*http.Request) (*http.Response, error)

func (f upgradeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestUpgradeNeverInstallsCorruptDownloadsAndPreservesModels(t *testing.T) {
	for _, bad := range []bool{false, true} {
		t.Run(fmt.Sprint(bad), func(t *testing.T) {
			dir := t.TempDir()
			model := filepath.Join(dir, "existing-model")
			os.WriteFile(model, []byte("fixture"), 0600)
			payload := "verified installer fixture"
			sum := fmt.Sprintf("%x", sha256.Sum256([]byte(payload)))
			installed := false
			a := &App{H: &apphost.Host{Config: apphost.Config{Data: dir}}, ollamaUpdate: &ollamaUpgrade{Busy: true}, upgradeContext: context.Background()}
			a.upgradeVersion = func(context.Context) string {
				if installed {
					return "0.40.13"
				}
				return "0.40.11"
			}
			a.upgradeSpace = func(string, int64) error { return nil }
			a.upgradeInstall = func(_ context.Context, p string) error {
				b, e := os.ReadFile(p)
				if e != nil || string(b) != payload {
					t.Fatal("unverified bytes reached installer")
				}
				installed = true
				return nil
			}
			a.upgradeClient = &http.Client{Transport: upgradeTransport(func(r *http.Request) (*http.Response, error) {
				body := ""
				switch r.URL.String() {
				case "https://api.github.com/repos/ollama/ollama/releases/latest":
					b, _ := json.Marshal(map[string]any{"tag_name": "v0.40.13", "assets": []any{map[string]any{"name": "OllamaSetup.exe", "size": len(payload), "digest": "sha256:" + sum, "browser_download_url": "https://github.com/ollama/ollama/releases/download/v0.40.13/OllamaSetup.exe"}}})
					body = string(b)
				case "https://github.com/ollama/ollama/releases/download/v0.40.13/OllamaSetup.exe":
					body = payload
					if bad {
						body = "corrupt"
					}
				default:
					return nil, errors.New("unexpected network request")
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
			})}
			a.runOllamaUpgrade()
			if installed == bad || a.ollamaUpdate.Busy {
				t.Fatal("incorrect upgrade result")
			}
			if !bad && !strings.Contains(a.ollamaUpdate.Phase, "upgraded to 0.40.13") {
				t.Fatal(a.ollamaUpdate.Phase)
			}
			if bad && !strings.Contains(a.ollamaUpdate.Phase, "verification failed") {
				t.Fatal(a.ollamaUpdate.Phase)
			}
			b, _ := os.ReadFile(model)
			if string(b) != "fixture" {
				t.Fatal("model changed")
			}
			matches, _ := filepath.Glob(filepath.Join(dir, ".ollama-upgrade-*"))
			if len(matches) > 0 {
				t.Fatal("installer files left behind")
			}
		})
	}
}

func TestUpgradeRequiresConfirmationBeforeAnyWork(t *testing.T) {
	a := &App{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/ollama/upgrade", strings.NewReader(`{"confirm":false}`))
	a.ollamaUpgradeStart(w, r)
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
}
func TestUpgradeRoutesRejectAnonymousRequests(t *testing.T) {
	h := &apphost.Host{Mux: http.NewServeMux()}
	a := &App{H: h}
	for _, method := range []string{"GET", "POST"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "http://localhost/api/ollama/upgrade", strings.NewReader(`{"confirm":true}`))
		fn := a.ollamaUpgradeStatus
		if method == "POST" {
			fn = a.ollamaUpgradeStart
		}
		h.Owner(fn)(w, r)
		if w.Code != 401 && w.Code != 403 {
			t.Fatal(w.Code)
		}
	}
}
func TestUpgradeRejectsForeignRedirects(t *testing.T) {
	c := ollamaHTTP(1)
	for _, raw := range []string{"https://evil.example/OllamaSetup.exe", "http://github.com/ollama/ollama"} {
		r, _ := http.NewRequest("GET", raw, nil)
		if c.CheckRedirect(r, nil) == nil {
			t.Fatal("accepted untrusted redirect")
		}
	}
}
