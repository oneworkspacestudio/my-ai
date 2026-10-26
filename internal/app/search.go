package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
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
	return a.searchWebOptions(ctx, query, "all", "")
}

type searchResponse struct {
	Provider  string      `json:"provider"`
	Query     string      `json:"query"`
	Retrieved string      `json:"retrieved"`
	Sources   []webResult `json:"sources"`
	Warnings  []string    `json:"warnings,omitempty"`
	Untrusted bool        `json:"untrusted_reference"`
}

func (a *App) searchWebOptions(ctx context.Context, query, language, timeRange string) (searchResponse, error) {
	query = strings.TrimSpace(query)
	if !validSearchQuery(query) {
		return searchResponse{}, errors.New("Enter a search question of up to500 characters")
	}
	c, e := a.searchConnection(ctx)
	if e != nil {
		return searchResponse{}, e
	}
	b, e := c.SearchWithOptions(ctx, query, language, timeRange)
	if e != nil {
		return searchResponse{}, e
	}
	result, e := searxResults(query, b)
	if e != nil {
		return result, e
	}
	// Retrieval is part of the shared tool, including MCP and models without
	// function calls. A blocked page remains explicitly a snippet-only source.
	var wg sync.WaitGroup
	for i := 0; i < len(result.Sources) && i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			readCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
			defer cancel()
			doc, err := a.readPublicPage(readCtx, result.Sources[i].URL, query)
			if err != nil {
				result.Sources[i].ReadError = err.Error()
				return
			}
			result.Sources[i].Text = doc.Text
			result.Sources[i].Truncated = doc.Truncated
			result.Sources[i].ReadURL = doc.URL
			result.Sources[i].Links = doc.Links
			if result.Sources[i].Published == "" {
				result.Sources[i].Published = doc.Published
			}
		}(i)
	}
	wg.Wait()
	return result, nil
}

func searxResults(query string, b []byte) (searchResponse, error) {
	var response struct {
		Results []struct {
			Title, URL, Content string
			PublishedDate       json.RawMessage `json:"publishedDate"`
			Engines             []string        `json:"engines"`
		} `json:"results"`
		Unresponsive []json.RawMessage `json:"unresponsive_engines"`
	}
	if json.Unmarshal(b, &response) != nil {
		return searchResponse{}, errors.New("SearXNG returned an unreadable response. Try again shortly")
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
		var published string
		_ = json.Unmarshal(v.PublishedDate, &published)
		out = append(out, webResult{Title: title, URL: address, Snippet: pageText(v.Content, 1200), Published: first(published, 80), Engines: v.Engines})
		if len(out) == 10 {
			break
		}
	}
	if len(out) == 0 {
		return searchResponse{}, errors.New("SearXNG found no usable sources; this does not establish that the requested event or fact does not exist")
	}
	r := searchResponse{Provider: "SearXNG", Query: query, Retrieved: now(), Sources: out, Untrusted: true}
	if len(response.Unresponsive) > 0 {
		r.Warnings = []string{"Some search engines did not respond; these results may be incomplete."}
	}
	return r, nil
}

func inheritedSearchConnection() (searchruntime.Connection, bool) {
	u, t := os.Getenv("MYAI_SEARCH_URL"), os.Getenv("MYAI_SEARCH_TOKEN")
	if u == "" && t == "" {
		return searchruntime.Connection{}, false
	}
	return searchruntime.Connection{URL: u, Token: t}, true
}
