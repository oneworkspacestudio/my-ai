package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/oneworkspacestudio/my-ai/internal/searchruntime"
)

func validSearchQuery(query string) bool {
	return strings.TrimSpace(query) != "" && utf8.ValidString(query) && len(query) <= 500 && !strings.ContainsRune(query, 0)
}

func (a *App) searchConnection(ctx context.Context) (searchruntime.Connection, error) {
	if a.searchConnectionOverride != nil {
		return a.searchConnectionOverride(ctx)
	}
	a.searchMu.Lock()
	if a.search == nil {
		var root string
		if a.H != nil {
			root = filepath.Join(a.H.Config.Data, "search-runtime")
		} else {
			var e error
			root, e = os.MkdirTemp("", "myai-search-")
			if e != nil {
				a.searchMu.Unlock()
				return searchruntime.Connection{}, errors.New("My AI could not prepare private web search")
			}
			a.searchTemp = root
		}
		a.search = searchruntime.New(root)
	}
	r := a.search
	a.searchMu.Unlock()
	return r.Connection(ctx)
}

func (a *App) closeSearch() {
	a.searchMu.Lock()
	defer a.searchMu.Unlock()
	if a.search != nil {
		a.search.Close()
	}
	if a.searchTemp != "" {
		a.search.RemoveTemporary()
	}
}

func (a *App) searchWeb(ctx context.Context, query string) (any, error) {
	query = strings.TrimSpace(query)
	if !validSearchQuery(query) {
		return nil, errors.New("Enter a search question of up to500 characters")
	}
	c, e := a.searchConnection(ctx)
	if e != nil {
		return nil, e
	}
	b, e := c.Search(ctx, query)
	if e != nil {
		return nil, e
	}
	return searxResults(query, b)
}

func searxResults(query string, b []byte) (any, error) {
	var response struct {
		Results []struct{ Title, URL, Content string } `json:"results"`
	}
	if json.Unmarshal(b, &response) != nil {
		return nil, errors.New("SearXNG returned an unreadable response. Try again shortly")
	}
	out := []webResult{}
	seen := map[string]bool{}
	for _, v := range response.Results {
		u, e := validateWebURL(v.URL)
		if e != nil {
			continue
		}
		u.Fragment = ""
		address := u.String()
		if seen[address] {
			continue
		}
		seen[address] = true
		title := pageText(v.Title, 300)
		if title == "" {
			title = u.Hostname()
		}
		out = append(out, webResult{Title: title, URL: address, Snippet: pageText(v.Content, 1200)})
		if len(out) == 6 {
			break
		}
	}
	if len(out) == 0 {
		return nil, errors.New("SearXNG found no usable sources. Try a different question; search engines may be temporarily unavailable")
	}
	return map[string]any{"provider": "SearXNG", "query": query, "retrieved": now(), "sources": out, "untrusted_reference": true}, nil
}

func inheritedSearchConnection() (searchruntime.Connection, bool) {
	u, t := os.Getenv("MYAI_SEARCH_URL"), os.Getenv("MYAI_SEARCH_TOKEN")
	if u == "" && t == "" {
		return searchruntime.Connection{}, false
	}
	return searchruntime.Connection{URL: u, Token: t}, true
}
