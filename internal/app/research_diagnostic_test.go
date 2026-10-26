package app

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestNativeSearchDiagnostics(t *testing.T) {
	if os.Getenv("MYAI_TEST_SEARCH_DIAG") != "1" {
		t.Skip("Explicit public diagnostics")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	a := App{}
	defer a.closeSearch()
	c, e := a.searchConnection(ctx)
	if e != nil {
		t.Fatal(e)
	}
	var rows []map[string]any
	for _, language := range []string{"all", "en", "en-GB"} {
		raw, err := c.SearchWithOptions(ctx, "Liverpool latest match result", language, "")
		var value any
		json.Unmarshal(raw, &value)
		rows = append(rows, map[string]any{"language": language, "response": value, "error": func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}()})
		t.Logf("language=%s bytes=%d error=%v", language, len(raw), err)
	}
	if p := os.Getenv("MYAI_TEST_DIAG_RECEIPT"); p != "" {
		raw, _ := json.MarshalIndent(rows, "", "  ")
		os.WriteFile(p, raw, 0600)
	}
}
