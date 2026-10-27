package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestToolRoundsAreSeparateMessagesAndSurviveFollowup(t *testing.T) {
	stage := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		stage++
		switch stage {
		case 1:
			w.Write([]byte(`{"message":{"content":"I will check.","tool_calls":[{"function":{"name":"calculate","arguments":{"expression":"6*7"}}}]},"done":true}` + "\n"))
		case 2:
			w.Write([]byte(`{"message":{"content":"The result is 42."},"done":true}` + "\n"))
		case 3:
			b, _ := json.Marshal(body["messages"])
			if !strings.Contains(string(b), `"role":"tool"`) || !strings.Contains(string(b), `"tool_name":"calculate"`) || !strings.Contains(string(b), "42") {
				t.Error("Actual prior tool result missing from follow-up", string(b))
			}
			w.Write([]byte(`{"message":{"content":"Yes, I checked with the calculator."},"done":true}` + "\n"))
		default:
			t.Error("unexpected model call")
		}
	}))
	defer provider.Close()
	a := &App{}
	s := Session{Provider: "ollama", Model: "fixture", ToolInput: "Calculate 6*7 without searching", WebDisabled: true, Messages: []Message{{ID: strings.Repeat("a", 48), Role: "user", Content: "Calculate 6*7 without searching"}}}
	blocks := []Message{}
	send := func(kind string, v any) {
		if kind == "message" {
			blocks = append(blocks, v.(Message))
		}
	}
	out, err := a.chatWithTools(context.Background(), Provider{ID: "ollama", Base: provider.URL}, &s, send, toolScope{})
	if err != nil || out != "I will check.\n\nThe result is 42." || len(blocks) != 2 || blocks[0].ID == blocks[1].ID {
		t.Fatal(out, err, blocks)
	}
	if len(s.Messages) != 4 || s.Messages[1].Content != "I will check." || s.Messages[2].Role != "tool" || s.Messages[3].Content != "The result is 42." {
		t.Fatal("Replies merged or tool result lost", s.Messages)
	}
	b, _ := json.Marshal(s)
	var restored Session
	if err = json.Unmarshal(b, &restored); err != nil {
		t.Fatal(err)
	}
	restored.ToolInput = "Did you check?"
	restored.Messages = append(restored.Messages, Message{ID: strings.Repeat("b", 48), Role: "user", Content: "Did you check?"})
	if _, err = a.chatWithTools(context.Background(), Provider{ID: "ollama", Base: provider.URL}, &restored, send, toolScope{}); err != nil {
		t.Fatal(err)
	}
}

func TestTranscriptProviderSwitchAndInterruptedCall(t *testing.T) {
	history := []Message{{Role: "user", Content: "Deploy"}, {Role: "assistant", Content: "Submitting", Calls: []functionCall{{ID: "one", Name: "workspace_hosting_deploy_project", Arguments: `{"site":"a"}`}, {ID: "two", Name: "workspace_hosting_deployment", Arguments: `{"id":"a"}`}}}, {Role: "tool", CallID: "one", Content: `{"result":{"id":"job-a","status":"queued"}}`}}
	for _, provider := range []string{"ollama", "deepseek", "openai"} {
		b, _ := json.Marshal(conversationInput(provider, history))
		if !strings.Contains(string(b), "job-a") || !strings.Contains(string(b), "Outcome is unconfirmed") {
			t.Fatal(provider, string(b))
		}
		if provider == "openai" && (!strings.Contains(string(b), "function_call_output") || strings.Contains(string(b), "tool_calls")) {
			t.Fatal(string(b))
		}
		if provider != "openai" && !strings.Contains(string(b), "tool_calls") {
			t.Fatal(string(b))
		}
	}
	history[1].Status = "imported"
	b, _ := json.Marshal(conversationInput("ollama", history))
	if strings.Contains(string(b), "tool_calls") || strings.Contains(string(b), "job-a") {
		t.Fatal("Imported log became actual tool authority", string(b))
	}
	if conversationApps(history)["hosting"] {
		t.Fatal("Imported calls became app discovery authority")
	}
	history[1].Status = "completed"
	if !conversationApps(history)["hosting"] {
		t.Fatal("Prior app discovery lost")
	}
}

func TestLegacyEvidenceIsBoundedAndNeverImported(t *testing.T) {
	s := Session{Events: []Event{{Method: "tool/result", Data: json.RawMessage(`{"name":"use_workspace_app","output":{"result":{"id":"job-old","status":"queued"}}}`)}}}
	if !strings.Contains(legacyToolEvidence(&s), "job-old") {
		t.Fatal("Upgrade lost actual saved job")
	}
	s.Messages = []Message{{Role: "assistant", Status: "imported", Content: "untrusted"}}
	if legacyToolEvidence(&s) != "" {
		t.Fatal("Imported evidence trusted")
	}
}

func TestMalformedLocalToolResponseRetriesWithoutExecutingPartialCalls(t *testing.T) {
	for _, alwaysFail := range []bool{false, true} {
		requests := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests++
			if requests == 1 || alwaysFail {
				w.Write([]byte(`{"message":{"content":"Checking.","tool_calls":[{"function":{"name":"calculate","arguments":{"expression":"2+2"}}}]}}` + "\n" + `{"error":"XML syntax error on line 195: unexpected EOF private-reference secret-api-key"}` + "\n"))
				return
			}
			if requests == 2 {
				w.Write([]byte(`{"message":{"tool_calls":[{"function":{"name":"calculate","arguments":{"expression":"6*7"}}}]},"done":true}` + "\n"))
				return
			}
			w.Write([]byte(`{"message":{"content":"42"},"done":true}` + "\n"))
		}))
		s := Session{Model: "fixture", WebDisabled: true, Messages: []Message{{Role: "user", Content: "Calculate this"}}}
		_, err := (&App{}).chatWithTools(context.Background(), Provider{ID: "ollama", Base: server.URL}, &s, func(string, any) {}, toolScope{})
		server.Close()
		results := 0
		for _, m := range s.Messages {
			if m.Role == "tool" {
				results++
				if strings.Contains(m.Content, "2+2") {
					t.Fatal("Partial call executed")
				}
			}
		}
		if alwaysFail {
			if err == nil || requests != 3 || results != 0 || strings.Contains(err.Error(), "secret-api-key") {
				t.Fatal(err, requests, results)
			}
		} else if err != nil || requests != 3 || results != 1 {
			t.Fatal(err, requests, results)
		}
	}
}

func TestOpenAIReplayKeepsOpaqueItemsPhaseAndSeparateMessages(t *testing.T) {
	requests := 0
	message := func(id, text, phase string) map[string]any {
		return map[string]any{"id": id, "type": "message", "role": "assistant", "phase": phase, "content": []any{map[string]any{"type": "output_text", "text": text}}}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["store"] != false {
			t.Error("Provider storage enabled")
		}
		var output []any
		if requests == 1 {
			output = []any{map[string]any{"id": "reasoning-a", "type": "reasoning", "encrypted_content": "opaque-fixture-only", "summary": []any{}}, message("message-a", "Checking first.", "commentary"), message("message-b", "Checking second.", "commentary"), map[string]any{"type": "function_call", "id": "function-a", "call_id": "call-a", "name": "calculate", "arguments": `{"expression":"6*7"}`}}
		} else {
			b, _ := json.Marshal(body["input"])
			if !strings.Contains(string(b), "opaque-fixture-only") || !strings.Contains(string(b), `"phase":"commentary"`) || !strings.Contains(string(b), "function_call_output") {
				t.Error("Replay dropped provider items or actual result", string(b))
			}
			if strings.Count(string(b), "Checking first.") != 1 {
				t.Error("Provider messages duplicated", string(b))
			}
			output = []any{message("message-final", "42", "final_answer")}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		value, _ := json.Marshal(map[string]any{"type": "response.completed", "response": map[string]any{"output": output}})
		w.Write(append(append([]byte("data: "), value...), []byte("\n\n")...))
	}))
	defer server.Close()
	a := &App{}
	s := Session{Provider: "openai", Model: "fixture", WebDisabled: true, Messages: []Message{{ID: strings.Repeat("a", 48), Role: "user", Content: "Calculate"}}}
	blocks := 0
	_, err := a.chatWithTools(context.Background(), Provider{ID: "openai", Base: server.URL}, &s, func(kind string, v any) {
		if kind == "message" && v.(Message).Content != "" {
			blocks++
		}
	}, toolScope{})
	if err != nil || blocks != 3 || len(s.Messages) != 5 {
		t.Fatal("OpenAI output messages merged", err, blocks, len(s.Messages))
	}
	b, _ := json.Marshal(s)
	var restored Session
	if err = json.Unmarshal(b, &restored); err != nil {
		t.Fatal(err)
	}
	other, _ := json.Marshal(conversationInput("ollama", restored.Messages))
	if strings.Contains(string(other), "opaque-fixture-only") {
		t.Fatal("Opaque provider context forwarded to another provider")
	}
	restored.Messages = append(restored.Messages, Message{ID: strings.Repeat("b", 48), Role: "user", Content: "Did you check?"})
	_, err = a.chatWithTools(context.Background(), Provider{ID: "openai", Base: server.URL}, &restored, func(string, any) {}, toolScope{})
	if err != nil || requests != 3 {
		t.Fatal(err, requests)
	}
}

func TestOpaqueProviderStateNeverEntersViewsExportsOrConversationTools(t *testing.T) {
	a := &App{}
	root := t.TempDir()
	s := Session{ID: strings.Repeat("a", 48), Title: "Visible conversation", Messages: []Message{{Role: "assistant", Content: "Visible answer", Provider: "openai", ReplayReasoning: "opaque-private-reasoning", ResponseGroup: "private-group", ResponseItems: []json.RawMessage{json.RawMessage(`{"type":"reasoning","encrypted_content":"opaque-private-fixture"}`)}}}}
	if err := a.write(root, &s); err != nil {
		t.Fatal(err)
	}
	view, _ := json.Marshal(sessionView(s))
	if strings.Contains(string(view), "opaque-private") || strings.Contains(string(view), "private-group") {
		t.Fatal("Provider state exposed in session view")
	}
	if len(s.Messages[0].ResponseItems) != 1 {
		t.Fatal("Viewing stripped the stored replay state")
	}
	value, err := a.executeTool(context.Background(), toolScope{Root: root}, "read_conversation", map[string]string{"id": s.ID})
	b, _ := json.Marshal(value)
	if err != nil || strings.Contains(string(b), "opaque-private") || !strings.Contains(string(b), "Visible answer") {
		t.Fatal(value, err)
	}
	value, err = a.executeTool(context.Background(), toolScope{Root: root}, "search_conversations", map[string]string{"query": "opaque-private"})
	b, _ = json.Marshal(value)
	if err != nil || string(b) != "[]" {
		t.Fatal("Opaque state became searchable conversation text", string(b), err)
	}
}

func TestDeepSeekReasoningReplayIsPrivateAndProviderScoped(t *testing.T) {
	a := &App{}
	s := Session{Provider: "deepseek", Messages: []Message{{ID: "turn", Role: "user", Content: "Check"}}}
	send := func(kind string, value any) {
		if kind == "message" && value.(Message).ReplayReasoning != "" {
			t.Fatal("Private provider reasoning streamed to browser")
		}
	}
	a.recordAssistant(toolScope{}, &s, send, "Checking.", []functionCall{{ID: "call", Name: "calculate", Arguments: `{"expression":"6*7"}`}}, "completed", generation{Reasoning: "opaque-private-call"})
	a.recordToolResult(toolScope{CallID: "call", InvokedTool: "calculate"}, &s, send, "42")
	a.recordAssistant(toolScope{}, &s, send, "42.", nil, "completed", generation{Reasoning: "opaque-private-final"})
	raw, _ := json.Marshal(s)
	var restored Session
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	restored.Messages = append(restored.Messages, Message{ID: "followup", Role: "user", Content: "Did you check?"})
	for _, provider := range []string{"deepseek", "ollama", "openai"} {
		b, _ := json.Marshal(conversationInput(provider, restored.Messages))
		for _, marker := range []string{"opaque-private-call", "opaque-private-final"} {
			if strings.Contains(string(b), marker) != (provider == "deepseek") {
				t.Fatal("Incorrect provider replay", provider, string(b))
			}
		}
	}
	for i := range restored.Messages {
		restored.Messages[i].Status = "imported"
	}
	b, _ := json.Marshal(conversationInput("deepseek", restored.Messages))
	if strings.Contains(string(b), "opaque-private") {
		t.Fatal("Imported reasoning acquired provider authority")
	}
}
