package app

import (
	"encoding/json"
	"errors"
)

// Only model-facing copies shrink. The canonical receipt and Activity record
// remain complete. Read operations can retrieve bounded sections by exact ID.
func modelToolResult(raw, session, message, name string) string {
	// Bounded file pages and schemas must remain exact: abbreviating those would
	// make edits impossible or tempt the model to invent missing arguments.
	switch name {
	case "describe_workspace_app", "load_skill", "read_tool_result", "read_project_file", "read_attachment", "search_attachment", "read_conversation":
		return raw
	}
	if len(raw) <= 4000 || !idRE.MatchString(session) || !idRE.MatchString(message) {
		return raw
	}
	var value any
	if json.Unmarshal([]byte(raw), &value) != nil {
		return raw
	}
	out, _ := json.Marshal(map[string]any{"notice": "Abbreviated tool result. Do not infer success or absence from omitted fields. Use read_tool_result for exact evidence before depending on omitted details.", "session": session, "message": message, "totalCharacters": len([]rune(raw)), "summary": summarizeToolValue(value, 0), "untrusted_reference": true})
	if len(out) > 4000 {
		out, _ = json.Marshal(map[string]any{"notice": "Large tool result stored. Read exact evidence with read_tool_result; no outcome is implied by this reference.", "session": session, "message": message, "totalCharacters": len([]rune(raw)), "untrusted_reference": true})
	}
	return string(out)
}
func summarizeToolValue(v any, depth int) any {
	switch x := v.(type) {
	case string:
		if len([]rune(x)) > 300 {
			return map[string]any{"excerpt": first(x, 250), "omitted": true}
		}
		return x
	case []any:
		if depth >= 4 {
			return map[string]any{"items": len(x), "omitted": true}
		}
		out := []any{}
		for i, item := range x {
			if i == 3 {
				out = append(out, map[string]any{"omittedItems": len(x) - i})
				break
			}
			out = append(out, summarizeToolValue(item, depth+1))
		}
		return out
	case map[string]any:
		if depth >= 4 {
			return map[string]any{"fields": len(x), "omitted": true}
		}
		out := map[string]any{}
		for k, item := range x {
			out[k] = summarizeToolValue(item, depth+1)
		}
		return out
	default:
		return v
	}
}
func (a *App) readToolResult(scope toolScope, args map[string]string) (any, error) {
	if scope.Root == "" || !idRE.MatchString(args["session"]) || !idRE.MatchString(args["message"]) {
		return nil, errors.New("Choose a tool result in your private conversation")
	}
	a.mu.Lock()
	s, err := a.read(scope.Root, args["session"])
	a.mu.Unlock()
	if err != nil {
		return nil, errors.New("Tool result not found in your private scope")
	}
	for _, m := range s.Messages {
		if m.ID == args["message"] && m.Role == "tool" && m.Status != "imported" {
			chars := []rune(m.Content)
			lo, hi, err := attachmentRange(args, len(chars))
			if err != nil {
				return nil, err
			}
			return map[string]any{"session": s.ID, "message": m.ID, "tool": m.ToolName, "content": string(chars[lo:hi]), "offset": lo, "nextOffset": hi, "totalCharacters": len(chars), "hasMore": hi < len(chars), "untrusted_reference": true}, nil
		}
	}
	return nil, errors.New("Tool result not found in your private scope")
}
