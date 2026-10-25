package searchruntime

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRuntimeArchiveRejectsEscapesAndDuplicatePaths(t *testing.T) {
	for _, names := range [][]string{{"../escape"}, {"foo/../../escape"}, {"C:/escape"}, {"file", "FILE"}, {"bad\\path"}} {
		var b bytes.Buffer
		z := zip.NewWriter(&b)
		for _, name := range names {
			w, _ := z.Create(name)
			w.Write([]byte("fixture"))
		}
		z.Close()
		if unpack(t.TempDir(), b.Bytes()) == nil {
			t.Fatal("Unsafe archive accepted", names)
		}
	}
}
func TestConnectionRejectsUntrustedDestinations(t *testing.T) {
	for _, u := range []string{"http://localhost:1234", "http://10.0.0.1:1234", "https://127.0.0.1:1234", "http://127.0.0.1:1234/other", "http://user:secret@127.0.0.1:1234", "http://127.0.0.1:80"} {
		if _, e := (Connection{URL: u, Token: strings.Repeat("a", 64)}).Search(context.Background(), "query"); e == nil {
			t.Fatal("Unsafe search destination", u)
		}
	}
}

func TestIncludedSearXNGNativeLiveSearch(t *testing.T) {
	if os.Getenv("MYAI_TEST_SEARCH") != "1" {
		t.Skip("Explicit real included-runtime test")
	}
	r := New(t.TempDir())
	defer r.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	c, e := r.Connection(ctx)
	if e != nil {
		t.Fatal(e)
	}
	version, e := exec.Command(filepath.Join(r.cmd.Dir, "python.exe"), "--version").Output()
	if e != nil || strings.TrimSpace(string(version)) != "Python 3.13.18" {
		t.Fatal("Incorrect included interpreter", e, string(version))
	}
	resp, e := http.Get(c.URL + "/healthz")
	if e != nil {
		t.Fatal(e)
	}
	resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatal("Anonymous private search access")
	}
	raw, e := c.Search(ctx, "SearXNG documentation")
	if e != nil {
		t.Fatal(e)
	}
	var results struct {
		Results []struct {
			URL string `json:"url"`
		}
	}
	if json.Unmarshal(raw, &results) != nil || len(results.Results) == 0 {
		t.Fatal("No actual live search results")
	}
	found := false
	for _, s := range results.Results {
		if strings.Contains(s.URL, "docs.searxng.org") {
			found = true
		}
	}
	if !found {
		t.Fatal("Relevant source absent")
	}
	t.Log("Included patched Python3.13.18 + native SearXNG returned real documentation results; anonymous access denied")
}
