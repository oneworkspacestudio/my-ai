package app

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
)

type hostMemory struct {
	RAMTotal     uint64 `json:"ramTotalBytes,omitempty"`
	RAMAvailable uint64 `json:"ramAvailableBytes,omitempty"`
	GPUFree      uint64 `json:"gpuFreeBytes,omitempty"`
	GPUTotal     uint64 `json:"gpuTotalBytes,omitempty"`
}
type modelContextFacts struct {
	Limit                          int
	Weights, LoadedRAM, LoadedVRAM uint64
	KVPerToken                     uint64
	LoadedContext                  int
}
type ContextBudget struct {
	ModelMax      int        `json:"modelMaxTokens,omitempty"`
	AutomaticChat int        `json:"automaticChatTokens"`
	AutomaticCode int        `json:"automaticCodeTokens"`
	Chat          int        `json:"chatTokens"`
	Code          int        `json:"codeTokens"`
	Selected      int        `json:"selectedTokens"`
	Reply         int        `json:"replyTokens"`
	Mode          string     `json:"mode"`
	Memory        hostMemory `json:"memory"`
	Basis         string     `json:"basis"`
	Notice        string     `json:"notice"`
}

func contextChoice(f modelContextFacts, mem hostMemory, settings *LocalModelSettings, mode string) ContextBudget {
	limit := f.Limit
	if limit < 512 {
		limit = 8192
	}
	limit = min(limit, 1048576)
	// Conservative fp16 KV estimate. Unknown/hybrid metadata uses 256 KiB/token,
	// rather than claiming a precisely known maximum for unsupported architectures.
	kv := f.KVPerToken
	if kv == 0 {
		kv = 256 * 1024
	}
	reserve := uint64(2 << 30)
	weights := f.Weights
	if weights == 0 {
		weights = 4 << 30
	}
	capacity := 8192
	basis := "Conservative fallback: model or memory information is incomplete"
	if mem.RAMAvailable > 0 {
		loaded := f.LoadedRAM
		if mem.RAMTotal > 0 {
			loaded = min(loaded, mem.RAMTotal)
		}
		available := mem.RAMAvailable + loaded
		if mem.RAMTotal > 0 {
			available = min(available, mem.RAMTotal)
		}
		if available > reserve+weights {
			capacity = int((available - reserve - weights) / kv / 2)
		} else {
			capacity = 512
		}
		basis = "Estimate from available system RAM, model weights and fp16 KV-cache allowance"
	}
	loadedVRAM := min(f.LoadedVRAM, mem.GPUTotal)
	if mem.GPUTotal > 0 && min(mem.GPUTotal, mem.GPUFree+loadedVRAM) > reserve+weights {
		capacity = int((min(mem.GPUTotal, mem.GPUFree+loadedVRAM) - reserve - weights) / kv / 2)
		basis = "Estimate from available NVIDIA VRAM, model weights and fp16 KV-cache allowance"
	}
	capacity = max(512, min(capacity, limit))
	if capacity >= 1024 {
		capacity = capacity / 1024 * 1024
	}
	b := ContextBudget{ModelMax: f.Limit, AutomaticChat: min(32768, capacity), AutomaticCode: min(65536, capacity), Mode: mode, Memory: mem, Basis: basis}
	b.Chat, b.Code = b.AutomaticChat, b.AutomaticCode
	if settings != nil {
		if settings.ContextTokens != nil {
			b.Chat = min(*settings.ContextTokens, limit)
			b.Code = b.Chat
		}
		if settings.CodeContextTokens != nil {
			b.Code = min(*settings.CodeContextTokens, limit)
		}
	}
	b.Selected = b.Chat
	if mode == "code" {
		b.Selected = b.Code
	}
	b.Reply = min(4096, max(128, b.Selected/4))
	if mode == "code" {
		b.Reply = min(8192, max(128, b.Selected/4))
	}
	if settings != nil && settings.MaxOutputTokens != nil {
		b.Reply = min(*settings.MaxOutputTokens, max(16, b.Selected/2))
	}
	b.Notice = "Automatic values are estimates, not a memory guarantee. Other models and concurrent requests also use memory."
	if mem.GPUTotal == 0 {
		b.Notice += " GPU memory is unavailable to this probe; system RAM estimates may involve CPU offloading."
	}
	if f.Limit == 0 {
		b.Notice += " Native model limit is unavailable; effective context is limited to 8K until metadata is available."
	}
	if b.Selected > capacity {
		b.Notice += " Your manual override exceeds the current memory estimate and may be slower or fail to load."
	}
	if settings != nil && ((settings.ContextTokens != nil && *settings.ContextTokens > limit) || (settings.CodeContextTokens != nil && *settings.CodeContextTokens > limit)) {
		b.Notice += " The saved request is capped to the model's reported context limit."
	}
	return b
}

func metadataNumber(info map[string]any, key string) uint64 {
	n, _ := info[key].(float64)
	if n <= 0 || math.IsNaN(n) || math.IsInf(n, 0) || n > float64(1<<60) {
		return 0
	}
	return uint64(n)
}
func (a *App) contextFacts(ctx context.Context, p Provider, model string) (modelContextFacts, hostMemory) {
	if a.contextProbe != nil {
		return a.contextProbe(ctx, p, model)
	}
	mem := readHostMemory(ctx)
	f := modelContextFacts{}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	read := func(method, path string, body any, out any) bool {
		r, e := request(ctx, p, method, path, body)
		if e != nil {
			return false
		}
		defer r.Body.Close()
		return json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(out) == nil
	}
	var show struct {
		Info map[string]any `json:"model_info"`
	}
	if read("POST", "/api/show", map[string]any{"model": model, "verbose": false}, &show) {
		arch, _ := show.Info["general.architecture"].(string)
		prefix := arch + "."
		f.Limit = int(metadataNumber(show.Info, prefix+"context_length"))
		layers := metadataNumber(show.Info, prefix+"block_count")
		heads := metadataNumber(show.Info, prefix+"attention.head_count")
		kv := metadataNumber(show.Info, prefix+"attention.head_count_kv")
		key := metadataNumber(show.Info, prefix+"attention.key_length")
		value := metadataNumber(show.Info, prefix+"attention.value_length")
		if key == 0 && heads > 0 {
			key = metadataNumber(show.Info, prefix+"embedding_length") / heads
		}
		if value == 0 {
			value = key
		}
		if layers > 0 && layers <= 512 && kv > 0 && kv <= 256 && key > 0 && key <= 8192 && value <= 8192 {
			f.KVPerToken = 2 * layers * kv * (key + value)
		}
	}
	var tags struct {
		Models []struct {
			Name string `json:"name"`
			Size uint64 `json:"size"`
		}
	}
	if read("GET", "/api/tags", nil, &tags) {
		for _, m := range tags.Models {
			if m.Name == model {
				f.Weights = m.Size
				break
			}
		}
	}
	var ps struct {
		Models []struct {
			Name    string `json:"name"`
			Size    uint64 `json:"size"`
			VRAM    uint64 `json:"size_vram"`
			Context int    `json:"context_length"`
		}
	}
	if read("GET", "/api/ps", nil, &ps) {
		for _, m := range ps.Models {
			if m.Name == model {
				f.LoadedVRAM = m.VRAM
				if m.Size > m.VRAM {
					f.LoadedRAM = m.Size - m.VRAM
				}
				f.LoadedContext = m.Context
				break
			}
		}
	}
	return f, mem
}
func (a *App) resolveContext(ctx context.Context, p Provider, model, mode string) ContextBudget {
	facts, mem := a.contextFacts(ctx, p, model)
	b := contextChoice(facts, mem, p.Generation, mode)
	if p.ID == "ollama" && os.Getenv("MYAI_SERVICE_URL") != "" {
		b.AutomaticChat = min(b.AutomaticChat, 32768)
		b.AutomaticCode = min(b.AutomaticCode, 32768)
		b.Chat = min(b.Chat, 32768)
		b.Code = min(b.Code, 32768)
		b.Selected = min(b.Selected, 32768)
		b.Reply = min(b.Reply, max(128, b.Selected/2))
		b.Notice += " AI Service limits shared requests to32768 context tokens."
	}
	return b
}
func (a *App) localModelContext(w http.ResponseWriter, r *http.Request) {
	model := r.URL.Query().Get("model")
	if !modelRE.MatchString(model) {
		apphost.Problem(w, 400, "Choose a valid local model")
		return
	}
	uid := apphost.Current(r).User.ID
	a.mu.Lock()
	saved, err := a.readLocalSettings(uid)
	a.mu.Unlock()
	if err != nil {
		apphost.Problem(w, 503, "Could not read model settings")
		return
	}
	s := saved.Models[model]
	p := Provider{ID: "ollama", Base: defaults()[2].Base, Generation: &s, User: uid}
	mode := r.URL.Query().Get("mode")
	if mode != "code" {
		mode = "chat"
	}
	apphost.JSON(w, 200, a.resolveContext(r.Context(), p, model, mode))
}

// Upper-bound UTF-8 byte accounting is intentionally conservative across local
// tokenizers. It never silently crops the current instruction or a tool schema.
func validateContextPayload(body map[string]any, budget *ContextBudget) error {
	if budget == nil {
		return nil
	}
	options, _ := body["options"].(map[string]any)
	ctx, _ := options["num_ctx"].(int)
	reply, _ := options["num_predict"].(int)
	data, _ := json.Marshal(map[string]any{"messages": body["messages"], "tools": body["tools"]})
	if len(data)+max(reply, 128)+256 > ctx {
		return &contextOverflow{}
	}
	return nil
}

type contextOverflow struct{}

func (*contextOverflow) Error() string {
	return "This request exceeds the effective context budget. Keep files attached, read smaller sections, start a new conversation, or increase this model's context in Model settings. The request was not sent with silently truncated instructions."
}

func applyContextBudget(body map[string]any, b *ContextBudget, planner bool) {
	if b == nil {
		return
	}
	o, _ := body["options"].(map[string]any)
	if o == nil {
		o = map[string]any{}
	}
	ctx, reply := b.Selected, b.Reply
	if planner {
		ctx = min(ctx, 16384)
		reply = min(reply, 800)
	}
	o["num_ctx"], o["num_predict"] = ctx, reply
	body["options"] = o
}
