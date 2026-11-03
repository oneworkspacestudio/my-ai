package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkflowSkillScopeAndBoundedGuidance(t *testing.T) {
	for _, scope := range []toolScope{{}, {Project: "p"}, {Project: "p", Writable: true, WorkspaceSession: "s"}, {NoWeb: true}} {
		for _, skill := range workflowSkills(scope) {
			v, err := loadWorkflowSkill(scope, skill.ID)
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := json.Marshal(v)
			if len(raw) > 1600 {
				t.Fatal("Skill grew beyond small-model guidance budget", skill.ID)
			}
		}
	}
	if _, err := loadWorkflowSkill(toolScope{}, "deployment"); err == nil {
		t.Fatal("Deployment skill exposed outside project/app scope")
	}
	if _, err := loadWorkflowSkill(toolScope{NoWeb: true}, "research"); err == nil {
		t.Fatal("Research skill exposed when web disabled")
	}
	if _, err := loadWorkflowSkill(toolScope{}, "../../credentials"); err == nil {
		t.Fatal("Arbitrary skill path accepted")
	}
}

func TestSmallModelToolFootprint(t *testing.T) {
	scope := toolScope{Root: "private", UID: "user", Project: "project", Writable: true, WorkspaceSession: "session", LazyTools: true, NoWeb: true, LoadedSkills: map[string]bool{}}
	b, _ := json.Marshal(toolDefinitions("ollama", scope))
	t.Logf("Initial project tool bytes: %d; workflow guidance bytes: %d", len(b), len(skillInstructions(scope)))
	if len(b)+len(skillInstructions(scope)) > 5500 {
		t.Fatal("Initial tools/workflows grew beyond compact scope")
	}
	for _, spec := range disclosedTools(scope) {
		if spec.Name == "read_conversation" || spec.Name == "read_attachment" {
			t.Fatal("Optional reader schema loaded eagerly")
		}
	}
	loadWorkflowSkill(scope, "conversations")
	found := false
	for _, spec := range disclosedTools(scope) {
		if spec.Name == "read_conversation" {
			found = true
		}
	}
	if !found {
		t.Fatal("Loading a skill did not expose its reader tools")
	}
}

func TestSmallLocalProjectTurnFits8K(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(body)
		if !strings.Contains(string(data), "Keep the heading") {
			t.Error("User constraint lost")
		}
		w.Write([]byte("{\"message\":{\"content\":\"Ready to inspect the selected project.\"},\"done\":true}\n"))
	}))
	defer server.Close()
	a := &App{}
	s := Session{ID: strings.Repeat("c", 48), Provider: "ollama", Model: "fixture", WebDisabled: true, ToolInput: "Keep the heading", Messages: []Message{{ID: strings.Repeat("d", 48), Role: "user", Content: "Keep the heading"}}}
	_, err := a.chatWithTools(context.Background(), Provider{ID: "ollama", Base: server.URL, Context: &ContextBudget{Selected: 8192, Reply: 2048}}, &s, func(string, any) {}, toolScope{Root: t.TempDir(), UID: "user", Project: "selected", Writable: true, WorkspaceSession: "fixture"})
	if err != nil || calls != 1 {
		t.Fatal("Small local project prompt failed before inference", calls, err)
	}
}

func TestAutomaticReplyLeavesRoomWithoutChangingManualLimits(t *testing.T) {
	body := map[string]any{"messages": []any{map[string]string{"role": "user", "content": strings.Repeat("x", 6000)}}, "options": map[string]any{"num_ctx": 8192, "num_predict": 2048}}
	if fitAutomaticReply(body, &LocalModelSettings{MaxOutputTokens: tuneInt(2048)}) {
		t.Fatal("Manual reply override changed")
	}
	if !fitAutomaticReply(body, nil) || validateContextPayload(body, &ContextBudget{}) != nil {
		t.Fatal("Automatic output space did not fit")
	}
	body["messages"] = []any{map[string]string{"role": "user", "content": strings.Repeat("x", 9000)}}
	if fitAutomaticReply(body, nil) || validateContextPayload(body, &ContextBudget{}) == nil {
		t.Fatal("Oversized user instruction silently accepted")
	}
}
func TestLargeToolReceiptIsRetrievableAndAccountScoped(t *testing.T) {
	a := &App{}
	root := t.TempDir()
	sid := strings.Repeat("a", 48)
	mid := strings.Repeat("b", 48)
	raw := `{"id":"job-one","state":"failed","error":"Build failed","log":"` + strings.Repeat("α漢字", 3000) + `"}`
	s := Session{ID: sid, Messages: []Message{{ID: mid, Role: "tool", Content: raw, ToolName: "workspace_hosting_deployment"}}}
	if err := a.write(root, &s); err != nil {
		t.Fatal(err)
	}
	packed := modelToolResult(raw, sid, mid, "workspace_hosting_deployment")
	if len(packed) >= len(raw) || !strings.Contains(packed, "failed") || !strings.Contains(packed, mid) {
		t.Fatal("Receipt summary lost status/reference", packed)
	}
	args := map[string]string{"session": sid, "message": mid, "offset": "2000", "limit": "1000"}
	got, err := a.readToolResult(toolScope{Root: root}, args)
	if err != nil {
		t.Fatal(err)
	}
	if got.(map[string]any)["content"] != string([]rune(raw)[2000:3000]) {
		t.Fatal("Unicode page corrupted")
	}
	if _, err = a.readToolResult(toolScope{Root: t.TempDir()}, args); err == nil {
		t.Fatal("Cross-account receipt leaked")
	}
	if _, err = a.readToolResult(toolScope{}, args); err == nil {
		t.Fatal("Unscoped receipt leaked")
	}
	s.Messages[0].Status = "imported"
	a.write(root, &s)
	if _, err = a.readToolResult(toolScope{Root: root}, args); err == nil {
		t.Fatal("Imported text promoted to tool evidence")
	}
	if raw != s.Messages[0].Content {
		t.Fatal("Stored receipt changed")
	}
	private := false
	s.Messages[0].Status = ""
	a.write(root, &s)
	a.runTool(context.Background(), toolScope{Root: root, PrivateAppRead: &private}, &s, func(string, any) {}, "read_tool_result", args)
	if !private {
		t.Fatal("Private receipt did not close web boundary")
	}
}
func TestContextPackingKeepsUserConstraintsAndPendingCalls(t *testing.T) {
	call := func(id, content string) any {
		return map[string]any{"role": "assistant", "content": "I will write the file.", "tool_calls": []any{map[string]any{"id": id, "function": map[string]any{"name": "write_project_file", "arguments": map[string]any{"content": content}}}}}
	}
	system := map[string]string{"role": "system", "content": "Respect scope"}
	user := map[string]string{"role": "user", "content": "Make it accessible. Keep my heading exactly."}
	pending := call("pending", "not run")
	msgs := []any{system, user, call("done", strings.Repeat("x", 12000)), map[string]any{"role": "tool", "tool_call_id": "done", "tool_name": "write_project_file", "content": `{"saved":true,"path":"index.html"}`}, pending}
	body := map[string]any{"messages": msgs, "tools": []string{"unchanged schema"}, "options": map[string]any{"num_ctx": 4096, "num_predict": 512}}
	if packLocalContext(body, toolScope{}) != 0 {
		t.Fatal("Packed without retrieval scope")
	}
	if packLocalContext(body, toolScope{Root: "private"}) != 1 {
		t.Fatal("Did not pack completed exchange")
	}
	out := body["messages"].([]any)

	raw, _ := json.Marshal(out)
	text := string(raw)
	for _, want := range []string{"Respect scope", "Keep my heading exactly", "pending", "not run", "saved", "index.html"} {
		if !strings.Contains(text, want) {
			t.Fatal("Lost protected context", want)
		}
	}
	if strings.Contains(text, strings.Repeat("x", 100)) {
		t.Fatal("Repeated source remains in context")
	}
	if err := validateContextPayload(body, &ContextBudget{}); err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 5 {
		t.Fatal("Original replay changed")
	}
}
func TestBranchPreservesReceiptsAttachmentsAndOriginal(t *testing.T) {
	s := Session{ID: strings.Repeat("a", 48), Title: "Example", Thread: "old-thread", Revision: 8, WorkspaceContext: true, Messages: []Message{{ID: "user-one", Role: "user", Content: "Build", Attachments: []AttachmentRef{{ID: "file-one"}}}, {ID: "assistant-one", Role: "assistant", Content: "Working", Calls: []functionCall{{ID: "call-one"}}, ResponseItems: []json.RawMessage{json.RawMessage(`{"secret":"opaque"}`)}}, {ID: "receipt", Role: "tool", CallID: "call-one", Content: `{"saved":true}`}, {ID: "user-two", Role: "user", Content: "Next"}}}
	branch, err := branchSession(s, "assistant-one", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(branch.Messages) != 3 || branch.Messages[2].Role != "tool" || len(branch.Messages[0].Attachments) != 1 || branch.Thread != "" || !branch.WorkspaceContext || branch.ID == s.ID {
		t.Fatal("Branch lost exact history/private boundary")
	}
	if len(branch.Messages[1].ResponseItems) != 0 || len(s.Messages[1].ResponseItems) != 1 || len(s.Messages) != 4 {
		t.Fatal("Opaque future state retained or original mutated")
	}
	edit, err := branchSession(s, "user-two", true)
	if err != nil || len(edit.Messages) != 3 {
		t.Fatal("Edit retained replaced prompt")
	}
	if _, err := branchSession(s, "receipt", false); err == nil {
		t.Fatal("Cannot branch inside a tool pair")
	}
}
