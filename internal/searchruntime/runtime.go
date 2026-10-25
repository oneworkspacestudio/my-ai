package searchruntime

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// The independently licensed upstream application and Python runtime are packed
// reproducibly by scripts/build-search-runtime.py. No install-time downloads.
//
//go:embed payload.zip
var payload []byte

type Connection struct{ URL, Token string }
type Runtime struct {
	mu         sync.Mutex
	root, work string
	connection Connection
	cmd        *exec.Cmd
	done       chan struct{}
	closed     bool
}

func New(root string) *Runtime { return &Runtime{root: root} }

func (r *Runtime) Connection(ctx context.Context) (Connection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return Connection{}, errors.New("SearXNG is restarting. Try again shortly")
	}
	if r.cmd != nil {
		select {
		case <-r.done:
			r.cmd = nil
			r.connection = Connection{}
			if r.work != "" && safeTree(r.work) == nil {
				_ = os.RemoveAll(r.work)
			}
		default:
			return r.connection, nil
		}
	}
	if !filepath.IsAbs(r.root) || filepath.Dir(filepath.Clean(r.root)) == filepath.Clean(r.root) {
		return Connection{}, errors.New("Use a dedicated private search runtime folder")
	}
	for p := filepath.Clean(r.root); ; p = filepath.Dir(p) {
		if _, e := os.Lstat(p); e == nil && linked(p) {
			return Connection{}, errors.New("Linked search runtime folders are blocked")
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	if err := os.MkdirAll(r.root, 0700); err != nil {
		return Connection{}, errors.New("My AI could not prepare its private search runtime")
	}
	binaries, err := prepare(r.root)
	if err != nil {
		return Connection{}, errors.New("The included SearXNG runtime could not be prepared. Reinstall My AI")
	}
	work, err := os.MkdirTemp(r.root, "process-")
	if err != nil {
		return Connection{}, err
	}
	r.work = work
	cache := filepath.Join(work, "cache")
	if err = os.Mkdir(cache, 0700); err != nil {
		return Connection{}, err
	}
	var secret [32]byte
	if _, err = rand.Read(secret[:]); err != nil {
		return Connection{}, err
	}
	token := hex.EncodeToString(secret[:])
	cmd := exec.Command(filepath.Join(binaries, "python.exe"), "-I", "-B", filepath.Join(binaries, "server.py"))
	cmd.Dir = binaries
	for _, key := range []string{"SystemRoot", "WINDIR", "SystemDrive", "COMSPEC", "PATH", "PATHEXT", "NUMBER_OF_PROCESSORS", "PROCESSOR_ARCHITECTURE"} {
		if v := os.Getenv(key); v != "" {
			cmd.Env = append(cmd.Env, key+"="+v)
		}
	}
	cmd.Env = append(cmd.Env, "MYAI_SEARCH_PROCESS_TOKEN="+token, "MYAI_SEARCH_WORK="+cache, "PYTHONUTF8=1")
	hide(cmd)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return Connection{}, err
	}
	cmd.Stderr = io.Discard
	if err = cmd.Start(); err != nil {
		return Connection{}, errors.New("Windows could not start the included SearXNG runtime")
	}
	r.cmd = cmd
	r.done = make(chan struct{})
	done := r.done
	go func() { _ = cmd.Wait(); close(done) }()
	ready := make(chan int, 1)
	go func() {
		s := bufio.NewScanner(out)
		s.Buffer(make([]byte, 1024), 4096)
		port := 0
		if s.Scan() {
			var v struct {
				Port     int    `json:"port"`
				Provider string `json:"provider"`
			}
			if json.Unmarshal(s.Bytes(), &v) == nil && v.Provider == "SearXNG" {
				port = v.Port
			}
		}
		ready <- port
		_, _ = io.Copy(io.Discard, out)
	}()
	startup, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	select {
	case port := <-ready:
		if port > 1023 && port < 65536 {
			r.connection = Connection{URL: "http://127.0.0.1:" + strconv.Itoa(port), Token: token}
			return r.connection, nil
		}
	case <-startup.Done():
	}
	stop(cmd)
	return Connection{}, errors.New("The included SearXNG runtime could not start. Restart My AI or reinstall it")
}

func prepare(root string) (string, error) {
	digest := sha256.Sum256(payload)
	dest := filepath.Join(root, "runtime-"+hex.EncodeToString(digest[:16]))
	if _, e := os.Lstat(dest); e == nil {
		if verified(dest) == nil {
			return dest, nil
		}
		// Retain a damaged generated cache for diagnosis; never execute it or
		// silently overwrite unexpected files. The directory stays on this volume.
		if safeTree(dest) != nil {
			return "", errors.New("linked runtime cache")
		}
		old, e := os.MkdirTemp(root, "retained-invalid-")
		if e != nil {
			return "", e
		}
		_ = os.Remove(old)
		if e = os.Rename(dest, old); e != nil {
			return "", e
		}
	}
	stage, e := os.MkdirTemp(root, "prepare-")
	if e != nil {
		return "", e
	}
	if e = unpack(stage, payload); e != nil {
		return "", e
	}
	if e = verified(stage); e != nil {
		return "", e
	}
	if e = os.Rename(stage, dest); e != nil {
		return "", e
	}
	return dest, nil
}

func verified(root string) error {
	if e := safeTree(root); e != nil {
		return e
	}
	z, e := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if e != nil {
		return e
	}
	var expected map[string]string
	for _, f := range z.File {
		if f.Name == "runtime-files.json" {
			r, e := f.Open()
			if e != nil {
				return e
			}
			e = json.NewDecoder(r).Decode(&expected)
			r.Close()
			if e != nil {
				return e
			}
		}
	}
	if len(expected) == 0 {
		return errors.New("missing runtime manifest")
	}
	if e := filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		rel, e := filepath.Rel(root, path)
		if e != nil {
			return e
		}
		rel = filepath.ToSlash(rel)
		if rel != "runtime-files.json" && expected[rel] == "" {
			return errors.New("unexpected runtime file")
		}
		return nil
	}); e != nil {
		return e
	}
	for name, want := range expected {
		f, e := os.Open(filepath.Join(root, filepath.FromSlash(name)))
		if e != nil {
			return e
		}
		h := sha256.New()
		_, e = io.Copy(h, f)
		f.Close()
		if e != nil {
			return e
		}
		if hex.EncodeToString(h.Sum(nil)) != want {
			return errors.New("runtime checksum mismatch")
		}
	}
	return nil
}

func (r *Runtime) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.closed = true
	if r.cmd != nil {
		stop(r.cmd)
		select {
		case <-r.done:
		case <-time.After(5 * time.Second):
			return
		}
	}
	// Only this process's generated directory may be removed. Refuse links.
	if r.work != "" && filepath.Dir(r.work) == filepath.Clean(r.root) {
		if safeTree(r.work) == nil {
			_ = os.RemoveAll(r.work)
		}
	}
}

// RemoveTemporary discards only a standalone MCP process's generated temp root.
// App-owned versioned caches remain available across restarts.
func (r *Runtime) RemoveTemporary() {
	r.Close()
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.closed || filepath.Dir(r.root) != filepath.Clean(os.TempDir()) || !strings.HasPrefix(filepath.Base(r.root), "myai-search-") {
		return
	}
	if r.cmd != nil {
		select {
		case <-r.done:
		default:
			return
		}
	}
	if safeTree(r.root) == nil {
		_ = os.RemoveAll(r.root)
	}
}

func (c Connection) Search(ctx context.Context, query string) ([]byte, error) {
	u, e := url.Parse(c.URL)
	if e != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || len(c.Token) != 64 {
		return nil, errors.New("Invalid private search connection")
	}
	port, e := strconv.Atoi(u.Port())
	if e != nil || port < 1024 || port > 65535 {
		return nil, errors.New("Invalid private search port")
	}
	u.Path = "/search"
	u.RawQuery = url.Values{"q": {query}, "format": {"json"}, "categories": {"general"}, "safesearch": {"1"}, "language": {"en"}}.Encode()
	req, e := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if e != nil {
		return nil, e
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")
	tr := &http.Transport{Proxy: nil, ResponseHeaderTimeout: 15 * time.Second}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, e := client.Do(req)
	if e != nil {
		return nil, errors.New("SearXNG is unavailable. Try again shortly")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || !strings.Contains(resp.Header.Get("Content-Type"), "json") {
		return nil, errors.New("SearXNG could not complete this search. Try a different question")
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, (2<<20)+1))
	if e != nil || len(b) > 2<<20 {
		return nil, errors.New("SearXNG response exceeded the search limit")
	}
	return b, nil
}

func unpack(root string, b []byte) error {
	z, e := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if e != nil {
		return e
	}
	var total uint64
	seen := map[string]bool{}
	for _, f := range z.File {
		name := filepath.FromSlash(f.Name)
		clean := filepath.Clean(name)
		if name != clean || name == "." || filepath.IsAbs(name) || strings.Contains(name, ":") || strings.Contains(f.Name, "\\") || strings.HasPrefix(clean, "..") || f.Mode()&os.ModeSymlink != 0 || seen[strings.ToLower(name)] {
			return errors.New("invalid runtime member")
		}
		seen[strings.ToLower(name)] = true
		total += f.UncompressedSize64
		if total > 512<<20 || f.UncompressedSize64 > 64<<20 {
			return errors.New("runtime too large")
		}
		path := filepath.Join(root, name)
		if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
			return e
		}
		in, e := f.Open()
		if e != nil {
			return e
		}
		out, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			in.Close()
			return e
		}
		n, err := io.Copy(out, io.LimitReader(in, int64(f.UncompressedSize64)+1))
		ce := out.Close()
		in.Close()
		if err != nil || ce != nil || n != int64(f.UncompressedSize64) {
			return fmt.Errorf("runtime member failed verification")
		}
	}
	return nil
}

func safeTree(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.Type()&os.ModeSymlink != 0 || linked(path) {
			return errors.New("linked runtime path")
		}
		return nil
	})
}
