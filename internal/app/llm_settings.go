package app

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/oneworkspacestudio/one-workspace/pkg/apphost"
	"io"
	"math"
	"net/http"
	"os"
)

type LocalModelSettings struct {
	ContextTokens     *int     `json:"contextTokens,omitempty"`
	CodeContextTokens *int     `json:"codeContextTokens,omitempty"`
	MaxOutputTokens   *int     `json:"maxOutputTokens,omitempty"`
	Temperature       *float64 `json:"temperature,omitempty"`
	TopP              *float64 `json:"topP,omitempty"`
	TopK              *int     `json:"topK,omitempty"`
	RepeatPenalty     *float64 `json:"repeatPenalty,omitempty"`
	Seed              *int     `json:"seed,omitempty"`
	BatchSize         *int     `json:"batchSize,omitempty"`
	KeepAlive         string   `json:"keepAlive,omitempty"`
	ChatMode          string   `json:"chatMode,omitempty"`
}
type localSettingsFile struct {
	Schema int                           `json:"schema"`
	Models map[string]LocalModelSettings `json:"models"`
}

func (s LocalModelSettings) validate() error {
	for _, v := range []struct {
		name   string
		p      *int
		lo, hi int
	}{
		{"Context size", s.ContextTokens, 512, 1048576}, {"Code context size", s.CodeContextTokens, 512, 1048576}, {"Reply limit", s.MaxOutputTokens, 16, 32768}, {"Top K", s.TopK, 0, 200}, {"Seed", s.Seed, 0, 2147483647}, {"Prompt batch size", s.BatchSize, 1, 2048},
	} {
		if v.p != nil && (*v.p < v.lo || *v.p > v.hi) {
			return fmt.Errorf("%s must be between %d and %d", v.name, v.lo, v.hi)
		}
	}
	for _, v := range []struct {
		name string
		p    *float64
		hi   float64
	}{
		{"Temperature", s.Temperature, 2}, {"Top P", s.TopP, 1}, {"Repetition penalty", s.RepeatPenalty, 2},
	} {
		if v.p != nil && (math.IsNaN(*v.p) || math.IsInf(*v.p, 0) || *v.p < 0 || *v.p > v.hi) {
			return fmt.Errorf("%s must be between 0 and %g", v.name, v.hi)
		}
	}
	if s.ContextTokens != nil && s.MaxOutputTokens != nil && *s.MaxOutputTokens >= *s.ContextTokens {
		return errors.New("Reply limit must be smaller than context size to leave room for your prompt")
	}
	if s.ChatMode != "" && s.ChatMode != "auto" && s.ChatMode != "plain" {
		return errors.New("Choose Automatic tools or Plain chat")
	}
	if s.CodeContextTokens != nil && s.MaxOutputTokens != nil && *s.MaxOutputTokens >= *s.CodeContextTokens {
		return errors.New("Reply limit must be smaller than Code context size")
	}
	switch s.KeepAlive {
	case "", "0", "2m", "5m", "15m":
	default:
		return errors.New("Choose a supported model memory duration")
	}
	return nil
}
func (a *App) localSettingsPath(uid string) (string, error) {
	if uid == "" {
		return "", errors.New("account required")
	}
	return safePath(a.H.Config.Data, "model-settings", fmt.Sprintf("%x.json", sha256.Sum256([]byte(uid))))
}
func (a *App) readLocalSettings(uid string) (localSettingsFile, error) {
	result := localSettingsFile{Schema: 1, Models: map[string]LocalModelSettings{}}
	path, err := a.localSettingsPath(uid)
	if err != nil {
		return result, err
	}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	defer file.Close()
	b, err := io.ReadAll(io.LimitReader(file, 256*1024+1))
	if err != nil {
		return result, err
	}
	if len(b) > 256*1024 {
		return result, errors.New("model settings are too large")
	}
	if err = json.Unmarshal(b, &result); err != nil {
		return result, err
	}
	if result.Schema != 1 || len(result.Models) > 128 {
		return result, errors.New("unsupported model settings")
	}
	if result.Models == nil {
		result.Models = map[string]LocalModelSettings{}
	}
	for model, settings := range result.Models {
		if !modelRE.MatchString(model) {
			return result, errors.New("invalid model")
		}
		if err = settings.validate(); err != nil {
			return result, err
		}
	}
	return result, nil
}
func (a *App) saveLocalSettings(uid, model string, s LocalModelSettings) error {
	if !modelRE.MatchString(model) {
		return errors.New("invalid model")
	}
	if err := s.validate(); err != nil {
		return err
	}
	saved, err := a.readLocalSettings(uid)
	if err != nil {
		return err
	}
	if s.ChatMode == "auto" {
		s.ChatMode = ""
	}
	if s == (LocalModelSettings{}) {
		delete(saved.Models, model)
	} else {
		saved.Models[model] = s
	}
	if len(saved.Models) > 128 {
		return errors.New("model profile limit reached")
	}
	path, err := a.localSettingsPath(uid)
	if err != nil {
		return err
	}
	return atomic(path, saved)
}
func (a *App) localModelSettings(w http.ResponseWriter, r *http.Request) {
	uid := apphost.Current(r).User.ID
	model := r.URL.Query().Get("model")
	var settings LocalModelSettings
	if r.Method == "POST" {
		var input struct {
			Model    string             `json:"model"`
			Settings LocalModelSettings `json:"settings"`
		}
		if !apphost.Decode(w, r, &input) {
			return
		}
		model, settings = input.Model, input.Settings
		if err := settings.validate(); err != nil {
			apphost.Problem(w, 400, err.Error())
			return
		}
	}
	if !modelRE.MatchString(model) {
		apphost.Problem(w, 400, "Choose a valid local model")
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if r.Method == "POST" {
		if err := a.saveLocalSettings(uid, model, settings); err != nil {
			apphost.Problem(w, 503, "Could not save model settings. Check private storage or the 128-model profile limit.")
			return
		}
		a.H.Audit(uid, "model-settings-save", model)
	}
	saved, err := a.readLocalSettings(uid)
	if err != nil {
		apphost.Problem(w, 503, "Could not read your saved model settings. Check private storage.")
		return
	}
	apphost.JSON(w, 200, map[string]any{"model": model, "settings": saved.Models[model]})
}
func applyLocalSettings(body map[string]any, s *LocalModelSettings, planner bool) {
	if s == nil {
		return
	}
	options, _ := body["options"].(map[string]any)
	if options == nil {
		options = map[string]any{}
	}
	if s.ContextTokens != nil {
		options["num_ctx"] = *s.ContextTokens
	}
	limit := 0
	if s.MaxOutputTokens != nil {
		limit = *s.MaxOutputTokens
	} else if s.ContextTokens != nil {
		limit = min(1024, *s.ContextTokens/4)
	}
	if limit > 0 {
		if old, ok := options["num_predict"].(int); planner && ok && old > 0 {
			limit = min(limit, old)
		}
		options["num_predict"] = limit
	}
	if s.BatchSize != nil {
		options["num_batch"] = *s.BatchSize
	}
	if !planner {
		if s.Temperature != nil {
			options["temperature"] = *s.Temperature
		}
		if s.TopP != nil {
			options["top_p"] = *s.TopP
		}
		if s.TopK != nil {
			options["top_k"] = *s.TopK
		}
		if s.RepeatPenalty != nil {
			options["repeat_penalty"] = *s.RepeatPenalty
		}
		if s.Seed != nil {
			options["seed"] = *s.Seed
		}
	}
	if len(options) > 0 {
		body["options"] = options
	}
	if s.KeepAlive == "0" {
		body["keep_alive"] = 0
	} else if s.KeepAlive != "" {
		body["keep_alive"] = s.KeepAlive
	}
}
