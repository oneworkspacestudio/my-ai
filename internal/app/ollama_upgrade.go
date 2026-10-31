package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ollamaUpgrade struct {
	mu        sync.Mutex
	Busy      bool   `json:"busy"`
	Phase     string `json:"phase"`
	Version   string `json:"version"`
	Completed int64  `json:"completed"`
	Total     int64  `json:"total"`
}
type ollamaAsset struct {
	Version, URL, SHA string
	Size              int64
}

func newerOllama(a, b string) bool {
	if !validOllamaVersion(a) || !validOllamaVersion(b) {
		return false
	}
	aa := strings.Split(strings.SplitN(strings.ReplaceAll(a, "+", "-"), "-", 2)[0], ".")
	bb := strings.Split(strings.SplitN(strings.ReplaceAll(b, "+", "-"), "-", 2)[0], ".")
	for i := 0; i < 3; i++ {
		x, e := strconv.Atoi(aa[i])
		y, f := strconv.Atoi(bb[i])
		if e != nil || f != nil {
			return false
		}
		if x != y {
			return x > y
		}
	}
	return false
}
func officialOllamaAsset(raw []byte) (ollamaAsset, error) {
	var r struct {
		Tag               string `json:"tag_name"`
		Draft, Prerelease bool
		Assets            []struct {
			Name   string
			Size   int64
			Digest string
			URL    string `json:"browser_download_url"`
		}
	}
	if json.Unmarshal(raw, &r) != nil || r.Draft || r.Prerelease {
		return ollamaAsset{}, errors.New("Cannot verify the official Ollama release")
	}
	version := strings.TrimPrefix(r.Tag, "v")
	if !validOllamaVersion(version) || strings.ContainsAny(version, "-+") {
		return ollamaAsset{}, errors.New("Invalid Ollama release version")
	}
	for _, a := range r.Assets {
		if a.Name != "OllamaSetup.exe" {
			continue
		}
		sha := strings.TrimPrefix(a.Digest, "sha256:")
		sum, e := hex.DecodeString(sha)
		want := "https://github.com/ollama/ollama/releases/download/" + r.Tag + "/OllamaSetup.exe"
		if a.URL != want || e != nil || len(sum) != 32 || a.Size < 1 || a.Size > 2<<30 {
			return ollamaAsset{}, errors.New("Cannot verify the official Windows installer")
		}
		return ollamaAsset{version, a.URL, sha, a.Size}, nil
	}
	return ollamaAsset{}, errors.New("Official Windows installer is unavailable")
}
func ollamaHTTP(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) > 5 {
			return errors.New("Too many redirects")
		}
		host := r.URL.Hostname()
		if r.URL.Scheme != "https" || (host != "github.com" && host != "release-assets.githubusercontent.com" && host != "objects.githubusercontent.com") {
			return errors.New("Untrusted installer redirect")
		}
		return nil
	}}
}
func runningOllamaVersion(ctx context.Context) string {
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:11434/api/version", nil)
	resp, e := (&http.Client{Timeout: 2 * time.Second}).Do(req)
	if e != nil {
		return ""
	}
	defer resp.Body.Close()
	var v struct{ Version string }
	if resp.StatusCode != 200 || json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&v) != nil || !validOllamaVersion(v.Version) {
		return ""
	}
	return v.Version
}
func (a *App) ollamaUpgradeStatus(w http.ResponseWriter, r *http.Request) {
	a.ollamaUpdate.mu.Lock()
	v := map[string]any{"busy": a.ollamaUpdate.Busy, "phase": a.ollamaUpdate.Phase, "version": a.ollamaUpdate.Version, "completed": a.ollamaUpdate.Completed, "total": a.ollamaUpdate.Total}
	a.ollamaUpdate.mu.Unlock()
	v["current"] = runningOllamaVersion(r.Context())
	apphost.JSON(w, 200, v)
}
func (a *App) ollamaUpgradeStart(w http.ResponseWriter, r *http.Request) {
	var in struct{ Confirm bool }
	if json.NewDecoder(io.LimitReader(r.Body, 1024)).Decode(&in) != nil || !in.Confirm {
		apphost.Problem(w, 400, "Confirm upgrading the shared Ollama server first")
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	active := len(a.running) > 0
	a.downloads.mu.Lock()
	downloading := a.downloads.active != ""
	for _, job := range a.downloads.jobs {
		if job.State == "queued" || job.State == "downloading" {
			downloading = true
		}
	}
	a.downloads.mu.Unlock()
	if active || downloading {
		apphost.Problem(w, 409, "Finish My AI requests and model downloads before upgrading Ollama")
		return
	}
	a.ollamaUpdate.mu.Lock()
	if a.ollamaUpdate.Busy {
		a.ollamaUpdate.mu.Unlock()
		apphost.Problem(w, 409, "An Ollama upgrade is already running")
		return
	}
	a.ollamaUpdate.Busy = true
	a.ollamaUpdate.Phase = "Checking the official Ollama release"
	a.ollamaUpdate.Completed = 0
	a.ollamaUpdate.Total = 0
	a.ollamaUpdate.mu.Unlock()
	go a.runOllamaUpgrade()
	apphost.JSON(w, 202, map[string]any{"accepted": true})
}
func (a *App) runOllamaUpgrade() {
	a.mu.Lock()
	parent := a.upgradeContext
	a.mu.Unlock()
	ctx, cancel := context.WithTimeout(parent, 30*time.Minute)
	defer cancel()
	u := a.ollamaUpdate
	phase := func(s string) { u.mu.Lock(); u.Phase = s; u.mu.Unlock() }
	defer func() { u.mu.Lock(); u.Busy = false; u.mu.Unlock() }()
	fail := func(s string) {
		phase(s + ". Models and settings were not removed. You can retry or use the official Ollama installer on the server")
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/repos/ollama/ollama/releases/latest", nil)
	req.Header.Set("User-Agent", "OneWorkspace-MyAI")
	resp, e := a.ollamaClient(30 * time.Second).Do(req)
	if e != nil {
		fail("Cannot check the official release")
		return
	}
	raw, e := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	resp.Body.Close()
	if e != nil || resp.StatusCode != 200 {
		fail("Cannot check the official release")
		return
	}
	asset, e := officialOllamaAsset(raw)
	if e != nil {
		fail(e.Error())
		return
	}
	current := a.ollamaVersion(ctx)
	if current != "" && !newerOllama(asset.Version, current) {
		phase("Ollama is already up to date")
		return
	}
	if e = a.ollamaSpace(a.H.Config.Data, asset.Size); e != nil {
		fail(e.Error())
		return
	}
	u.mu.Lock()
	u.Version = asset.Version
	u.Total = asset.Size
	u.mu.Unlock()
	dir, e := os.MkdirTemp(a.H.Config.Data, ".ollama-upgrade-")
	if e != nil {
		fail("Cannot create private installer storage")
		return
	}
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "OllamaSetup.exe")
	f, e := os.OpenFile(file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		fail("Cannot create installer file")
		return
	}
	phase("Downloading the verified Ollama installer")
	req, _ = http.NewRequestWithContext(ctx, "GET", asset.URL, nil)
	resp, e = a.ollamaClient(20 * time.Minute).Do(req)
	if e != nil {
		f.Close()
		fail("Installer download failed")
		return
	}
	hash := sha256.New()
	n, e := io.Copy(io.MultiWriter(f, hash, upgradeProgress{u}), io.LimitReader(resp.Body, asset.Size+1))
	resp.Body.Close()
	closeErr := f.Close()
	if e != nil || closeErr != nil || resp.StatusCode != 200 || n != asset.Size || hex.EncodeToString(hash.Sum(nil)) != asset.SHA {
		fail("Installer verification failed")
		return
	}
	phase("Checking the Ollama publisher and installing; local AI may be temporarily unavailable")
	if e = a.ollamaInstall(ctx, file); e != nil {
		fail(e.Error())
		return
	}
	phase("Waiting for the upgraded Ollama server")
	for i := 0; i < 120; i++ {
		v := a.ollamaVersion(ctx)
		if v == asset.Version || newerOllama(v, asset.Version) {
			phase("Ollama upgraded to " + v + ". Retry your model request")
			return
		}
		select {
		case <-ctx.Done():
			fail("Upgrade verification timed out")
			return
		case <-time.After(5 * time.Second):
		}
	}
	fail("The installed server has not reported the new version; restart Ollama on the server and retry")
}

type upgradeProgress struct{ u *ollamaUpgrade }

func (p upgradeProgress) Write(b []byte) (int, error) {
	p.u.mu.Lock()
	p.u.Completed += int64(len(b))
	p.u.mu.Unlock()
	return len(b), nil
}

func (a *App) ollamaMaintenanceBusy() bool {
	if a.ollamaUpdate == nil {
		return false
	}
	a.ollamaUpdate.mu.Lock()
	defer a.ollamaUpdate.mu.Unlock()
	return a.ollamaUpdate.Busy
}

func (a *App) ollamaVersion(ctx context.Context) string {
	if a.upgradeVersion != nil {
		return a.upgradeVersion(ctx)
	}
	return runningOllamaVersion(ctx)
}
func (a *App) ollamaClient(timeout time.Duration) *http.Client {
	if a.upgradeClient != nil {
		return a.upgradeClient
	}
	return ollamaHTTP(timeout)
}
func (a *App) ollamaSpace(path string, size int64) error {
	if a.upgradeSpace != nil {
		return a.upgradeSpace(path, size)
	}
	return ollamaInstallSpace(path, size)
}
func (a *App) ollamaInstall(ctx context.Context, path string) error {
	if a.upgradeInstall != nil {
		return a.upgradeInstall(ctx, path)
	}
	return installOllama(ctx, path)
}
