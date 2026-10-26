package app

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

type pageLink struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}
type webDocument struct {
	URL       string     `json:"url"`
	Title     string     `json:"title,omitempty"`
	Text      string     `json:"text"`
	Published string     `json:"published,omitempty"`
	Links     []pageLink `json:"links,omitempty"`
	Retrieved string     `json:"retrieved"`
	Untrusted bool       `json:"untrusted_reference"`
	Truncated bool       `json:"truncated,omitempty"`
}

func (a *App) readPublicPage(ctx context.Context, address, query string) (webDocument, error) {
	get := webGet
	if a.webReadOverride != nil {
		get = a.webReadOverride
	}
	b, kind, finalURL, err := get(ctx, address)
	if err != nil {
		return webDocument{}, err
	}
	d := extractWebDocument(string(b), kind, finalURL, query)
	return d, nil
}

func extractWebDocument(raw, kind, address, query string) webDocument {
	d := webDocument{URL: address, Retrieved: now(), Untrusted: true, Truncated: strings.Contains(kind, "myai-truncated=1")}
	if !strings.Contains(kind, "html") {
		d.Text = relevantExcerpt(raw, query, 6500)
		return d
	}
	root, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return d
	}
	var main *html.Node
	var structured []string
	var scan func(*html.Node)
	scan = func(n *html.Node) {
		if n.Type == html.ElementNode {
			attrs := map[string]string{}
			for _, a := range n.Attr {
				attrs[a.Key] = a.Val
			}
			if main == nil && (n.Data == "main" || n.Data == "article" || attrs["role"] == "main") || n.Data == "main" && main != nil && main.Data != "main" {
				main = n
			}
			if n.Data == "title" {
				d.Title = pageText(nodeInnerText(n), 300)
			}
			if n.Data == "meta" && (attrs["property"] == "article:published_time" || attrs["name"] == "datePublished") {
				d.Published = first(attrs["content"], 80)
			}
			if n.Data == "script" && attrs["type"] == "application/ld+json" {
				v := nodeInnerText(n)
				var value any
				if len(v) <= 40000 && json.Unmarshal([]byte(v), &value) == nil {
					structured = append(structured, v)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			scan(c)
		}
	}
	scan(root)
	if main == nil {
		main = root
	}
	var text strings.Builder
	seen := map[string]bool{}
	base, _ := validateWebURL(address)
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if text.Len() > 200000 {
			return
		}
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "noscript", "svg", "nav", "footer", "form":
				return
			}
			if n.Data == "header" && main == root {
				return
			}
			for _, attr := range n.Attr {
				if (n.Data == "img" && attr.Key == "alt") || attr.Key == "aria-label" {
					text.WriteString(" " + first(attr.Val, 300) + " ")
				}
				if n.Data == "time" && attr.Key == "datetime" {
					text.WriteString("\n" + attr.Val + " ")
				}
				if n.Data == "a" && attr.Key == "href" && base != nil && len(d.Links) < 24 {
					link, e := base.Parse(attr.Val)
					if e != nil {
						continue
					}
					link.Fragment = ""
					if _, e = validateWebURL(link.String()); e == nil && !seen[link.String()] {
						seen[link.String()] = true
						d.Links = append(d.Links, pageLink{Title: first(strings.Join(strings.Fields(nodeInnerText(n)), " "), 160), URL: link.String()})
					}
				}
			}
			switch n.Data {
			case "p", "div", "li", "tr", "h1", "h2", "h3", "section", "br":
				text.WriteByte('\n')
			}
		}
		if n.Type == html.TextNode {
			text.WriteString(n.Data)
			text.WriteByte(' ')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == html.ElementNode && (n.Data == "p" || n.Data == "div" || n.Data == "li" || n.Data == "tr") {
			text.WriteByte('\n')
		}
	}
	walk(main)
	body := d.Title + "\n" + text.String()
	if len(structured) > 0 {
		body += "\nStructured page data (untrusted):\n" + strings.Join(structured, "\n")
	}
	d.Text = relevantExcerpt(body, query, 6500)
	return d
}

func nodeInnerText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(v *html.Node) {
		if v.Type == html.TextNode {
			b.WriteString(v.Data)
		}
		for c := v.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

// Keep relevant passages from throughout a page, rather than just its first
// navigation-heavy characters. Original passage order and dates are retained.
func relevantExcerpt(text, query string, limit int) string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.Join(strings.Fields(line), " ")
		for len([]rune(line)) > 900 {
			r := []rune(line)
			lines = append(lines, string(r[:900]))
			line = string(r[900:])
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	tokens := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
	stop := " the what when where which with from this that was were is are does please search latest official reference date result "
	type rank struct{ i, score int }
	ranks := []rank{}
	for i, line := range lines {
		score := 0
		lower := strings.ToLower(line)
		for _, w := range tokens {
			if len([]rune(w)) > 1 && !strings.Contains(stop, " "+w+" ") && strings.Contains(lower, w) {
				score += 2
			}
		}
		ranks = append(ranks, rank{i, score})
	}
	sort.SliceStable(ranks, func(i, j int) bool { return ranks[i].score > ranks[j].score })
	keep := map[int]bool{0: true}
	size := len([]rune(lines[0]))
	for _, r := range ranks {
		if r.score == 0 {
			continue
		}
		for _, i := range []int{r.i - 1, r.i, r.i + 1} {
			if i >= 0 && i < len(lines) && !keep[i] && size+len([]rune(lines[i])) < limit-100 {
				keep[i] = true
				size += len([]rune(lines[i]))
			}
		}
	}
	if size < limit/2 {
		for i, line := range lines {
			if !keep[i] && size+len([]rune(line)) < limit-100 {
				keep[i] = true
				size += len([]rune(line))
			}
		}
	}
	var b strings.Builder
	previous := -1
	for i, line := range lines {
		if keep[i] {
			if previous >= 0 && i > previous+1 {
				b.WriteString("\n[…]\n")
			}
			b.WriteString(line)
			b.WriteByte('\n')
			previous = i
		}
	}
	return first(b.String(), limit)
}
