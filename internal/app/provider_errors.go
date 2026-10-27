package app

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Classify bounded provider responses, but never return their contents: upstream
// errors can contain credentials, private prompts, paths and HTML.
type providerHTTPError struct {
	Status           int
	Provider         string
	Reason           string
	ToolsUnsupported bool
}

func (e *providerHTTPError) Error() string {
	if e.Provider == "ollama" {
		switch e.Reason {
		case "format":
			return "The server's Ollama version cannot load this model format. Update Ollama on the server, then try again. Your downloaded model is kept"
		case "memory":
			return "Ollama could not fit this model in the server's available memory. Try a smaller model or a shorter conversation. Your downloaded model is kept"
		case "missing":
			return "This model is not installed on this server. Open Models to download it or choose an installed model"
		case "context":
			return "This conversation is too long for the selected local model. Start a new conversation or choose a model with a larger context window"
		case "tool-format":
			return "The local model produced a malformed tool call. No action from that failed response was executed. Try again or choose another model"
		}
		if e.ToolsUnsupported {
			return "This local model does not support automatic tools. Choose a model with tool support for project editing"
		}
		if e.Status == 503 || e.Status == 429 {
			return "Ollama is busy. Wait for the current request to finish, then try again"
		}
		return "Ollama could not load or run this model. The server owner may need to update Ollama or check available memory. Your downloaded model is kept"
	}
	if e.Status == 401 || e.Status == 403 {
		return "The AI provider rejected access. Check this provider's API key and model access in Settings"
	}
	if e.Status == 429 {
		return "The AI provider is busy or its usage limit was reached. Try again shortly or check your provider usage"
	}
	if e.Status >= 500 || e.Status == 0 {
		return "The AI provider could not complete this response. Try again shortly"
	}
	return fmt.Sprintf("The AI provider could not accept this request (HTTP %d). Check the selected model and try again", e.Status)
}

func providerFailure(provider string, status int, raw []byte) *providerHTTPError {
	e := &providerHTTPError{Provider: provider, Status: status}
	var envelope struct {
		Error json.RawMessage `json:"error"`
	}
	message := raw
	if json.Unmarshal(raw, &envelope) == nil && len(envelope.Error) > 0 {
		message = envelope.Error
	}
	var text string
	if json.Unmarshal(message, &text) != nil {
		var detail struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		}
		if json.Unmarshal(message, &detail) != nil {
			return e
		}
		text = detail.Message + " " + detail.Code
	}
	text = strings.ToLower(text)
	has := func(words ...string) bool {
		for _, word := range words {
			if strings.Contains(text, word) {
				return true
			}
		}
		return false
	}
	e.ToolsUnsupported = has("does not support tools", "not support tool", "tools is not supported", "tool use is not supported")
	if provider == "ollama" {
		switch {
		case has("tool call parsing failed", "error parsing tool call", "failed to parse tool call", "xml syntax error"):
			e.Reason = "tool-format"
		case has("unknown tensor type", "unsupported tensor type", "unknown model architecture", "unsupported model architecture", "unsupported model format", "requires a newer version", "invalid ggml type", "unknown ggml type", "unsupported ggml type", "unsupported model type"):
			e.Reason = "format"
		case has("out of memory", "out-of-memory", "insufficient memory", "requires more system memory", "unable to allocate", "failed to allocate", "memory layout cannot be allocated"):
			e.Reason = "memory"
		case has("model") && has("not found", "does not exist"):
			e.Reason = "missing"
		case has("context_length_exceeded", "context length exceeded", "no user query found", "input length exceeds", "prompt is too long"):
			e.Reason = "context"
		}
	}
	return e
}

var ollamaVersionRE = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][a-zA-Z0-9.-]+)?$`)

func validOllamaVersion(s string) bool { return len(s) <= 48 && ollamaVersionRE.MatchString(s) }
