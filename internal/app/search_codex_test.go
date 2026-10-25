package app

import (
	"context"
	"encoding/json"
	"github.com/oneworkspacestudio/my-ai/internal/searchruntime"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "mcp-serve" {
		if RunMCP(os.Stdin, os.Stdout) != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestOfficialCodexDiscoversIncludedSearchMCP(t *testing.T) {
	if os.Getenv("MYAI_TEST_CODEX_MCP") != "1" {
		t.Skip("Explicit isolated official Codex discovery check")
	}
	own, _ := os.Executable()
	if strings.HasSuffix(strings.ToLower(own), ".test.exe") {
		t.Fatal("Build this test as NativeSearchCheck.exe so production MCP configuration is exercised")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	root := t.TempDir()
	r := searchruntime.New(filepath.Join(root, "search"))
	defer r.Close()
	c, e := r.Connection(ctx)
	if e != nil {
		t.Fatal(e)
	}
	g, e := startAgentWithSearch(filepath.Join(root, "codex"), root, "", c)
	if e != nil {
		t.Fatal(e)
	}
	defer g.Close()
	if e = g.Initialize(ctx); e != nil {
		t.Fatal(e)
	}
	b, e := g.Call(ctx, "mcpServerStatus/list", map[string]any{"limit": 20})
	if e != nil {
		t.Fatal(e)
	}
	var status struct {
		Data []struct {
			Name  string                     `json:"name"`
			Tools map[string]json.RawMessage `json:"tools"`
		} `json:"data"`
	}
	if e = json.Unmarshal(b, &status); e != nil {
		t.Fatal(e)
	}
	found := false
	for _, s := range status.Data {
		if s.Name == "myai" {
			_, found = s.Tools["web_search"]
			if len(s.Tools) != 4 {
				t.Fatal("Unexpected public MCP tools", string(b))
			}
		}
	}
	if !found {
		t.Fatal("Official Codex did not discover private SearXNG tool", string(b))
	}
	t.Log("Actual official Codex app-server discovered all four public My AI MCP tools in an isolated home, without a provider key or global login")
}
