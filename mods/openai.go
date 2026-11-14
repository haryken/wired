package mods

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/os-vector/wired/vars"
)

// OpenAI backend for AI giao tiếp. vic-cloud reads openai.json (the API key
// stays here, never in xiaozhi.json); the web UI only ever sees a masked key.
var (
	openAIConfigPath = "/data/data/com.anki.victor/persistent/xiaozhi/openai.json"
	openAIMemoryPath = "/data/data/com.anki.victor/persistent/xiaozhi/openai_memory.json"
)

type OpenAICfg struct {
	APIKey         string `json:"api_key"`
	Mode           string `json:"mode"`
	BaseURL        string `json:"base_url"`
	RealtimeModel  string `json:"realtime_model"`
	STTModel       string `json:"stt_model"`
	STTBaseURL     string `json:"stt_base_url"`
	STTAPIKey      string `json:"stt_api_key"`
	LLMModel       string `json:"llm_model"`
	LLMBaseURL     string `json:"llm_base_url"`
	LLMAPIKey      string `json:"llm_api_key"`
	TTSModel       string `json:"tts_model"`
	TTSEngine      string `json:"tts_engine"`
	EdgeVoice      string `json:"edge_voice"`
	GeminiAPIKey   string `json:"gemini_api_key"`
	GroqAPIKey     string `json:"groq_api_key"`
	GeminiTTSModel string `json:"gemini_tts_model"`
	GeminiVoice    string `json:"gemini_voice"`
	// Gemini Live backend (xiaozhi.json backend "gemini").
	GeminiLiveModel string `json:"gemini_live_model"`
	GeminiLiveVoice string `json:"gemini_live_voice"`
	// Grok Voice backend (xiaozhi.json backend "grok", xAI key).
	XAIAPIKey    string `json:"xai_api_key"`
	GrokModel    string `json:"grok_model"`
	GrokVoice    string `json:"grok_voice"`
	// Qwen Live backend (xiaozhi.json backend "qwen", DashScope key).
	DashScopeAPIKey string `json:"dashscope_api_key"`
	QwenModel       string `json:"qwen_model"`
	QwenVoice       string `json:"qwen_voice"`
	QwenRegion      string `json:"qwen_region"` // "intl" | "cn"
	Voice           string `json:"voice"`
	Language        string `json:"language"`
	Instructions    string `json:"instructions"`
	MemoryTurns     int    `json:"memory_turns"`
	VisionModel     string `json:"vision_model"`
}

const (
	openAIModeRealtime   = "realtime"
	openAIModePipeline   = "pipeline"
	openAIMaxInstruction = 2000

	ttsEngineOpenAI = "openai"
	ttsEngineGoogle = "google"
	ttsEngineEdge   = "edge"
	ttsEngineGemini = "gemini"

	geminiAPIHost = "generativelanguage.googleapis.com"
	groqAPIHost   = "api.groq.com"
)

var geminiModelsBase = "https://" + geminiAPIHost + "/v1beta"

// voiceIDRe / modelIDRe keep user values safe inside SSML and URL paths.
// Underscore/dot allowed for Qwen voices (longanqian_v3.1).
var (
	voiceIDRe = regexp.MustCompile(`^[A-Za-z0-9._-]{1,80}$`)
	modelIDRe = regexp.MustCompile(`^[A-Za-z0-9._/:-]{1,120}$`)
)

func defaultOpenAICfg() OpenAICfg {
	return OpenAICfg{
		Mode:            openAIModeRealtime,
		BaseURL:         "https://api.openai.com/v1",
		RealtimeModel:   "gpt-realtime-mini",
		STTModel:        "gpt-4o-mini-transcribe",
		LLMModel:        "gpt-4o-mini",
		TTSModel:        "gpt-4o-mini-tts",
		TTSEngine:       ttsEngineOpenAI,
		EdgeVoice:       "en-US-EmmaMultilingualNeural",
		GeminiTTSModel:  "gemini-2.5-flash-preview-tts",
		GeminiVoice:     "Kore",
		GeminiLiveModel: "gemini-3.8-live",
		GeminiLiveVoice: "Kore",
		GrokModel:       "grok-voice-latest",
		GrokVoice:       "eve",
		QwenModel:       "qwen-audio-3.1-realtime-flash",
		QwenVoice:       "longanqian_v3.1",
		QwenRegion:      "intl",
		Voice:           "coral",
		Language:        "vi",
		MemoryTurns:     10,
		VisionModel:     "gpt-4o-mini",
	}
}

func urlHost(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Host)
}

// Key routing mirrors vic-cloud: Groq and Gemini have one key each, used by
// whichever step points at them; a custom server only gets its own step key
// (shared when STT and LLM use the same custom host).
func (c OpenAICfg) sttBase() string {
	if c.STTBaseURL != "" {
		return c.STTBaseURL
	}
	return c.BaseURL
}

func (c OpenAICfg) llmBase() string {
	if c.LLMBaseURL != "" {
		return c.LLMBaseURL
	}
	return c.BaseURL
}

func (c OpenAICfg) stageKey(base, own, otherBase, otherOwn string) string {
	switch {
	case base == "":
		return c.APIKey
	case urlHost(base) == groqAPIHost:
		return c.GroqAPIKey
	case urlHost(base) == geminiAPIHost:
		return c.GeminiAPIKey
	case own == "" && otherBase != "" && urlHost(base) == urlHost(otherBase):
		return otherOwn
	}
	return own
}

func (c OpenAICfg) sttKey() string {
	return c.stageKey(c.STTBaseURL, c.STTAPIKey, c.LLMBaseURL, c.LLMAPIKey)
}

func (c OpenAICfg) llmKey() string {
	return c.stageKey(c.LLMBaseURL, c.LLMAPIKey, c.STTBaseURL, c.STTAPIKey)
}

func (c OpenAICfg) geminiKey() string { return c.GeminiAPIKey }

// normalizeProviderKeys moves Groq / Gemini keys that older versions stored
// per step into the provider's own field, leaving the step fields for custom
// servers only.
func normalizeProviderKeys(c *OpenAICfg) {
	for _, s := range []struct {
		base string
		key  *string
	}{{c.STTBaseURL, &c.STTAPIKey}, {c.LLMBaseURL, &c.LLMAPIKey}} {
		var dst *string
		switch urlHost(s.base) {
		case groqAPIHost:
			dst = &c.GroqAPIKey
		case geminiAPIHost:
			dst = &c.GeminiAPIKey
		default:
			continue
		}
		if *dst == "" {
			*dst = *s.key
		}
		*s.key = ""
	}
}

func isOfficialOpenAI(base string) bool {
	return strings.Contains(base, "api.openai.com")
}

func loadOpenAICfg() OpenAICfg {
	cfg := defaultOpenAICfg()
	if data, err := os.ReadFile(openAIConfigPath); err == nil {
		_ = json.Unmarshal(data, &cfg)
		if isLegacyKeyLayout(data) {
			normalizeProviderKeys(&cfg)
		}
	}
	return cfg
}

// isLegacyKeyLayout: files written before per-provider keys have no
// groq_api_key field (current saves always write it, even empty).
func isLegacyKeyLayout(data []byte) bool {
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return false
	}
	_, ok := raw["groq_api_key"]
	return !ok
}

func saveOpenAICfg(cfg OpenAICfg) error {
	dir := filepath.Dir(openAIConfigPath)
	if err := vars.MkdirAll(dir); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := openAIConfigPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0640); err != nil {
		return err
	}
	// vic-cloud (user cloud, group anki) must read it; nobody else.
	_, gid := vars.AnkiIDs()
	_ = os.Chown(tmp, 0, gid)
	_ = os.Chmod(tmp, 0640)
	return os.Rename(tmp, openAIConfigPath)
}

func maskKey(k string) string {
	k = strings.TrimSpace(k)
	if k == "" {
		return ""
	}
	if len(k) <= 8 {
		return "…"
	}
	return "…" + k[len(k)-4:]
}

// openAIView is what the UI gets: everything except the keys themselves.
func openAIView(cfg OpenAICfg) map[string]interface{} {
	xz, _ := loadXiaozhiCfg()
	return map[string]interface{}{
		"mode":                cfg.Mode,
		"base_url":            cfg.BaseURL,
		"realtime_model":      cfg.RealtimeModel,
		"stt_model":           cfg.STTModel,
		"stt_base_url":        cfg.STTBaseURL,
		"llm_model":           cfg.LLMModel,
		"llm_base_url":        cfg.LLMBaseURL,
		"tts_model":           cfg.TTSModel,
		"tts_engine":          cfg.TTSEngine,
		"edge_voice":          cfg.EdgeVoice,
		"gemini_tts_model":    cfg.GeminiTTSModel,
		"gemini_voice":        cfg.GeminiVoice,
		"gemini_live_model":   cfg.GeminiLiveModel,
		"gemini_live_voice":   cfg.GeminiLiveVoice,
		"has_gemini_live_key": cfg.geminiKey() != "",
		"grok_model":          cfg.GrokModel,
		"grok_voice":          cfg.GrokVoice,
		"has_xai_key":         strings.TrimSpace(cfg.XAIAPIKey) != "",
		"xai_key_hint":        maskKey(cfg.XAIAPIKey),
		"qwen_model":          cfg.QwenModel,
		"qwen_voice":          cfg.QwenVoice,
		"qwen_region":         cfg.QwenRegion,
		"has_dashscope_key":   strings.TrimSpace(cfg.DashScopeAPIKey) != "",
		"dashscope_key_hint":  maskKey(cfg.DashScopeAPIKey),
		"backend":             xz.Backend,
		"voice":               cfg.Voice,
		"language":            cfg.Language,
		"instructions":        cfg.Instructions,
		"memory_turns":        cfg.MemoryTurns,
		"vision_model":        cfg.VisionModel,
		"has_key":             strings.TrimSpace(cfg.APIKey) != "",
		"key_hint":            maskKey(cfg.APIKey),
		"has_stt_key":         strings.TrimSpace(cfg.STTAPIKey) != "",
		"stt_key_hint":        maskKey(cfg.STTAPIKey),
		"has_llm_key":         strings.TrimSpace(cfg.LLMAPIKey) != "",
		"llm_key_hint":        maskKey(cfg.LLMAPIKey),
		"has_gemini_key":      strings.TrimSpace(cfg.GeminiAPIKey) != "",
		"gemini_key_hint":     maskKey(cfg.GeminiAPIKey),
		"has_groq_key":        strings.TrimSpace(cfg.GroqAPIKey) != "",
		"groq_key_hint":       maskKey(cfg.GroqAPIKey),
		"needs_openai_key":    openAINeedsKey(cfg),
		"missing_keys":        openAIMissingKeys(cfg),
		"active":              xz.Enabled && isDirectAPIBackend(xz.Backend),
	}
}

// isLocalHost: LAN servers (Ollama, LM Studio, …) usually need no key.
func isLocalHost(host string) bool {
	h := host
	if hh, _, err := net.SplitHostPort(host); err == nil {
		h = hh
	}
	if h == "localhost" || strings.HasSuffix(h, ".local") {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate())
}

// openAIMissingKeys lists pipeline stages pointing at a cloud server with no
// key; the robot would get 401/400 there and stay silent.
func openAIMissingKeys(c OpenAICfg) []string {
	missing := []string{}
	if c.Mode != openAIModePipeline {
		return missing
	}
	if c.STTBaseURL != "" && c.sttKey() == "" && !isLocalHost(urlHost(c.STTBaseURL)) {
		missing = append(missing, "stt")
	}
	if c.LLMBaseURL != "" && c.llmKey() == "" && !isLocalHost(urlHost(c.LLMBaseURL)) {
		missing = append(missing, "llm")
	}
	if c.TTSEngine == ttsEngineGemini && c.geminiKey() == "" {
		missing = append(missing, "tts")
	}
	return missing
}

// openAINeedsKey: some stage would call api.openai.com without a key.
func openAINeedsKey(c OpenAICfg) bool {
	if c.Mode != openAIModePipeline {
		return strings.TrimSpace(c.APIKey) == "" && isOfficialOpenAI(c.BaseURL)
	}
	if isOfficialOpenAI(c.sttBase()) && c.sttKey() == "" {
		return true
	}
	return isOfficialOpenAI(c.llmBase()) && c.llmKey() == ""
}

func validBaseURL(u string) bool {
	return strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://")
}

// applyOpenAIForm updates only the fields present in the request. Empty key
// fields keep the stored key; clear_key / clear_llm_key remove it.
func applyOpenAIForm(cfg *OpenAICfg, r *http.Request) error {
	_ = r.ParseForm()
	has := func(k string) bool { _, ok := r.Form[k]; return ok }
	str := func(k string) string { return strings.TrimSpace(r.Form.Get(k)) }

	if v := str("api_key"); v != "" {
		cfg.APIKey = v
	}
	if str("clear_key") == "1" {
		cfg.APIKey = ""
	}
	for _, k := range []struct {
		field, clear string
		dst          *string
	}{
		{"gemini_api_key", "clear_gemini_key", &cfg.GeminiAPIKey},
		{"groq_api_key", "clear_groq_key", &cfg.GroqAPIKey},
		{"xai_api_key", "clear_xai_key", &cfg.XAIAPIKey},
		{"dashscope_api_key", "clear_dashscope_key", &cfg.DashScopeAPIKey},
	} {
		if v := str(k.field); v != "" {
			*k.dst = v
		}
		if str(k.clear) == "1" {
			*k.dst = ""
		}
	}
	if str("clear_stt_key") == "1" {
		cfg.STTAPIKey = ""
	}
	if str("clear_llm_key") == "1" {
		cfg.LLMAPIKey = ""
	}
	if has("mode") {
		m := str("mode")
		if m != openAIModeRealtime && m != openAIModePipeline {
			return fmt.Errorf("mode must be realtime or pipeline")
		}
		cfg.Mode = m
	}
	if has("base_url") {
		u := strings.TrimRight(str("base_url"), "/")
		if u == "" {
			u = defaultOpenAICfg().BaseURL
		}
		if !validBaseURL(u) {
			return fmt.Errorf("base_url must start with http:// or https://")
		}
		cfg.BaseURL = u
	}
	for _, f := range []struct {
		key string
		dst *string
	}{
		{"llm_base_url", &cfg.LLMBaseURL},
		{"stt_base_url", &cfg.STTBaseURL},
	} {
		if !has(f.key) {
			continue
		}
		u := strings.TrimRight(str(f.key), "/")
		if u != "" && !validBaseURL(u) {
			return fmt.Errorf("%s must start with http:// or https://", f.key)
		}
		*f.dst = u
	}
	if has("tts_engine") {
		e := str("tts_engine")
		switch e {
		case ttsEngineOpenAI, ttsEngineGoogle, ttsEngineEdge, ttsEngineGemini:
		default:
			return fmt.Errorf("tts_engine must be openai, google, edge or gemini")
		}
		cfg.TTSEngine = e
	}
	if has("edge_voice") && str("edge_voice") != "" && !voiceIDRe.MatchString(str("edge_voice")) {
		return fmt.Errorf("edge_voice is not a valid voice name")
	}
	if has("gemini_voice") && str("gemini_voice") != "" && !voiceIDRe.MatchString(str("gemini_voice")) {
		return fmt.Errorf("gemini_voice is not a valid voice name")
	}
	for _, k := range []string{"gemini_tts_model", "gemini_live_model", "grok_model", "qwen_model"} {
		if has(k) && str(k) != "" && !modelIDRe.MatchString(str(k)) {
			return fmt.Errorf("%s is not a valid model name", k)
		}
	}
	for _, k := range []string{"gemini_live_voice", "grok_voice", "qwen_voice"} {
		if has(k) && str(k) != "" && !voiceIDRe.MatchString(str(k)) {
			return fmt.Errorf("%s is not a valid voice name", k)
		}
	}
	if has("qwen_region") {
		switch str("qwen_region") {
		case "", "intl", "cn":
		default:
			return fmt.Errorf("qwen_region must be intl or cn")
		}
	}
	if has("memory_turns") {
		n, err := strconv.Atoi(str("memory_turns"))
		if err != nil || n < 0 || n > 50 {
			return fmt.Errorf("memory_turns must be 0-50")
		}
		cfg.MemoryTurns = n
	}
	if has("instructions") {
		ins := strings.TrimSpace(r.Form.Get("instructions"))
		if len([]rune(ins)) > openAIMaxInstruction {
			return fmt.Errorf("instructions too long (max %d)", openAIMaxInstruction)
		}
		cfg.Instructions = ins
	}
	d := defaultOpenAICfg()
	for _, f := range []struct {
		key string
		dst *string
		def string
	}{
		{"realtime_model", &cfg.RealtimeModel, d.RealtimeModel},
		{"stt_model", &cfg.STTModel, d.STTModel},
		{"llm_model", &cfg.LLMModel, d.LLMModel},
		{"tts_model", &cfg.TTSModel, d.TTSModel},
		{"edge_voice", &cfg.EdgeVoice, d.EdgeVoice},
		{"gemini_tts_model", &cfg.GeminiTTSModel, d.GeminiTTSModel},
		{"gemini_voice", &cfg.GeminiVoice, d.GeminiVoice},
		{"gemini_live_model", &cfg.GeminiLiveModel, d.GeminiLiveModel},
		{"gemini_live_voice", &cfg.GeminiLiveVoice, d.GeminiLiveVoice},
		{"grok_model", &cfg.GrokModel, d.GrokModel},
		{"grok_voice", &cfg.GrokVoice, d.GrokVoice},
		{"qwen_model", &cfg.QwenModel, d.QwenModel},
		{"qwen_voice", &cfg.QwenVoice, d.QwenVoice},
		{"qwen_region", &cfg.QwenRegion, d.QwenRegion},
		{"voice", &cfg.Voice, d.Voice},
		{"language", &cfg.Language, d.Language},
		{"vision_model", &cfg.VisionModel, d.VisionModel},
	} {
		if has(f.key) {
			v := str(f.key)
			if v == "" {
				v = f.def
			}
			*f.dst = v
		}
	}
	cfg.Language = strings.ToLower(cfg.Language)
	if cfg.QwenRegion != "cn" {
		cfg.QwenRegion = "intl"
	}
	// A step key typed while that step points at Groq / Gemini (older pages
	// do this) belongs to the provider, not to the custom-server slot.
	for _, s := range []struct {
		field, base string
		own         *string
	}{
		{"stt_api_key", cfg.STTBaseURL, &cfg.STTAPIKey}, {"llm_api_key", cfg.LLMBaseURL, &cfg.LLMAPIKey},
	} {
		v := str(s.field)
		if v == "" {
			continue
		}
		switch urlHost(s.base) {
		case groqAPIHost:
			cfg.GroqAPIKey = v
		case geminiAPIHost:
			cfg.GeminiAPIKey = v
		default:
			*s.own = v
		}
	}
	return nil
}

var openAITestClient = &http.Client{Timeout: 12 * time.Second}

// openAICheck is the result for one stage (realtime / stt / llm / tts).
type openAICheck struct {
	Stage  string `json:"stage"`
	Host   string `json:"host,omitempty"`
	OK     bool   `json:"ok"`
	Models int    `json:"models"`
	Model  string `json:"model,omitempty"`
	Found  bool   `json:"model_found"`
	Free   bool   `json:"free,omitempty"`
	Error  string `json:"error,omitempty"`
}

type openAITestResult struct {
	Status string        `json:"status"`
	OK     bool          `json:"ok"`
	Checks []openAICheck `json:"checks"`
}

// listModels calls GET {base}/models (OpenAI and most compatible servers).
func listModels(base, key string) ([]string, error) {
	return fetchModelIDs(base+"/models", func(h http.Header) {
		if key != "" {
			h.Set("Authorization", "Bearer "+key)
		}
	})
}

// listGeminiModels uses the native Gemini API, which lists the TTS models.
func listGeminiModels(key string) ([]string, error) {
	return fetchModelIDs(geminiModelsBase+"/models?pageSize=1000", func(h http.Header) {
		h.Set("x-goog-api-key", key)
	})
}

// Built-in Live choices when the key is missing or Google's list is empty.
var geminiLiveModelDefaults = []string{
	"gemini-3.8-live",
	"gemini-2.5-flash-native-audio-latest",
}

func isGeminiLiveModelID(id string) bool {
	s := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(id), "models/"))
	if s == "" {
		return false
	}
	return strings.Contains(s, "live") || strings.Contains(s, "native-audio")
}

func geminiModelSupportsLive(methods []string) bool {
	for _, m := range methods {
		if strings.EqualFold(m, "bidiGenerateContent") {
			return true
		}
	}
	return false
}

// listGeminiLiveModels returns Live / native-audio model IDs for the dropdown.
// Prefers models advertising bidiGenerateContent; falls back to name matching.
func listGeminiLiveModels(key string) ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, geminiModelsBase+"/models?pageSize=1000", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-goog-api-key", key)
	resp, err := openAITestClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &e) == nil && e.Error.Message != "" {
			return nil, fmt.Errorf("%s: %s", resp.Status, e.Error.Message)
		}
		return nil, fmt.Errorf("%s", resp.Status)
	}
	var out struct {
		Models []struct {
			Name    string   `json:"name"`
			Methods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("bad /models response")
	}
	byMethod := make([]string, 0)
	byName := make([]string, 0)
	seen := map[string]bool{}
	add := func(dst *[]string, id string) {
		id = strings.TrimPrefix(strings.TrimSpace(id), "models/")
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		*dst = append(*dst, id)
	}
	for _, m := range out.Models {
		id := strings.TrimPrefix(m.Name, "models/")
		if geminiModelSupportsLive(m.Methods) {
			add(&byMethod, id)
			continue
		}
		if isGeminiLiveModelID(id) {
			add(&byName, id)
		}
	}
	if len(byMethod) > 0 {
		return byMethod, nil
	}
	if len(byName) > 0 {
		return byName, nil
	}
	return nil, fmt.Errorf("no Live models listed for this key")
}

func mergeGeminiLiveModels(api []string, current string) []string {
	return mergeModelLists(api, geminiLiveModelDefaults, current)
}

func mergeModelLists(api, defaults []string, current string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(api)+len(defaults)+1)
	add := func(id string) {
		id = strings.TrimPrefix(strings.TrimSpace(id), "models/")
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		out = append(out, id)
	}
	for _, id := range api {
		add(id)
	}
	for _, id := range defaults {
		add(id)
	}
	add(current)
	return out
}

func modelKindDefaults(kind string) []string {
	switch kind {
	case "realtime":
		return []string{"gpt-realtime-mini", "gpt-realtime"}
	case "stt":
		return []string{"gpt-4o-mini-transcribe", "gpt-4o-transcribe", "whisper-large-v3-turbo", "whisper-large-v3", "whisper-1"}
	case "llm":
		return []string{"gpt-4o-mini", "gpt-4o", "gemini-3.5-flash-lite", "gemini-3.8-flash", "openai/gpt-oss-120b"}
	case "tts":
		return []string{"gpt-4o-mini-tts", "gpt-4o-tts"}
	case "gemini_tts":
		return []string{"gemini-2.5-flash-preview-tts", "gemini-3.8-flash-tts"}
	case "vision":
		return []string{"gpt-4o-mini", "gpt-4o"}
	case "grok":
		return []string{"grok-voice-latest"}
	case "qwen":
		return []string{
			"qwen-audio-3.1-realtime-flash",
			"qwen-audio-3.1-realtime-plus",
			"qwen-audio-3.0-realtime-flash",
			"qwen-audio-3.0-realtime-plus",
		}
	case "gemini_live":
		return append([]string{}, geminiLiveModelDefaults...)
	default:
		return nil
	}
}

func filterModelsByKind(ids []string, kind string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		s := strings.ToLower(strings.TrimPrefix(id, "models/"))
		keep := false
		switch kind {
		case "realtime":
			keep = strings.Contains(s, "realtime")
		case "stt":
			keep = strings.Contains(s, "whisper") || strings.Contains(s, "transcribe")
		case "tts", "gemini_tts":
			keep = strings.Contains(s, "tts")
		case "vision", "llm":
			keep = isChatCompletionModel(s)
		case "grok":
			keep = strings.Contains(s, "grok") || strings.Contains(s, "voice")
		case "gemini_live":
			keep = isGeminiLiveModelID(s)
		default:
			keep = true
		}
		if keep {
			out = append(out, strings.TrimPrefix(id, "models/"))
		}
	}
	return out
}

func isChatCompletionModel(s string) bool {
	deny := []string{
		"whisper", "tts", "realtime", "embedding", "dall-e", "moderation",
		"image", "transcribe", "computer-use", "sora", "codex-mini",
	}
	for _, d := range deny {
		if strings.Contains(s, d) {
			return false
		}
	}
	return true
}

func listGrokModels(key string) ([]string, error) {
	return fetchModelIDs("https://api.x.ai/v1/models", func(h http.Header) {
		h.Set("Authorization", "Bearer "+key)
	})
}

func listModelsForKind(cfg OpenAICfg, kind string) (ids []string, source string, err error) {
	switch kind {
	case "gemini_live":
		key := cfg.geminiKey()
		if key == "" {
			return nil, "defaults", fmt.Errorf("Gemini API key chưa nhập")
		}
		ids, err = listGeminiLiveModels(key)
		return ids, "api", err
	case "gemini_tts":
		key := cfg.geminiKey()
		if key == "" {
			return nil, "defaults", fmt.Errorf("Gemini API key chưa nhập")
		}
		all, e := listGeminiModels(key)
		if e != nil {
			return nil, "defaults", e
		}
		return filterModelsByKind(all, "gemini_tts"), "api", nil
	case "grok":
		key := strings.TrimSpace(cfg.XAIAPIKey)
		if key == "" {
			return nil, "defaults", fmt.Errorf("xAI API key chưa nhập")
		}
		all, e := listGrokModels(key)
		if e != nil {
			return nil, "defaults", e
		}
		filtered := filterModelsByKind(all, "grok")
		if len(filtered) == 0 {
			filtered = all
		}
		return filtered, "api", nil
	case "qwen":
		// DashScope has no simple OpenAI-style /models list for Audio-Realtime;
		// return curated defaults (UI still shows them as a dropdown).
		if strings.TrimSpace(cfg.DashScopeAPIKey) == "" {
			return nil, "defaults", fmt.Errorf("DashScope API key chưa nhập")
		}
		return modelKindDefaults("qwen"), "defaults", nil
	case "stt":
		base, key := cfg.sttBase(), cfg.sttKey()
		if key == "" && !isLocalHost(urlHost(base)) {
			return nil, "defaults", fmt.Errorf("API key chưa nhập")
		}
		if urlHost(base) == geminiAPIHost {
			all, e := listGeminiModels(key)
			if e != nil {
				return nil, "defaults", e
			}
			return filterModelsByKind(all, "stt"), "api", nil
		}
		all, e := listModels(base, key)
		if e != nil {
			return nil, "defaults", e
		}
		filtered := filterModelsByKind(all, "stt")
		if len(filtered) == 0 {
			filtered = all
		}
		return filtered, "api", nil
	case "llm":
		base, key := cfg.llmBase(), cfg.llmKey()
		if key == "" && !isLocalHost(urlHost(base)) {
			return nil, "defaults", fmt.Errorf("API key chưa nhập")
		}
		if urlHost(base) == geminiAPIHost {
			all, e := listGeminiModels(key)
			if e != nil {
				return nil, "defaults", e
			}
			filtered := filterModelsByKind(all, "llm")
			if len(filtered) == 0 {
				filtered = all
			}
			return filtered, "api", nil
		}
		all, e := listModels(base, key)
		if e != nil {
			return nil, "defaults", e
		}
		filtered := filterModelsByKind(all, "llm")
		if len(filtered) == 0 {
			filtered = all
		}
		return filtered, "api", nil
	default: // realtime, tts, vision, …
		base, key := cfg.BaseURL, cfg.APIKey
		if strings.TrimSpace(base) == "" {
			base = defaultOpenAICfg().BaseURL
		}
		if key == "" && !isLocalHost(urlHost(base)) {
			return nil, "defaults", fmt.Errorf("OpenAI API key chưa nhập")
		}
		all, e := listModels(base, key)
		if e != nil {
			return nil, "defaults", e
		}
		filtered := filterModelsByKind(all, kind)
		if len(filtered) == 0 && (kind == "vision" || kind == "llm") {
			filtered = filterModelsByKind(all, "llm")
		}
		if len(filtered) == 0 {
			filtered = all
		}
		return filtered, "api", nil
	}
}

func currentModelForKind(cfg OpenAICfg, kind string) string {
	switch kind {
	case "realtime":
		return cfg.RealtimeModel
	case "stt":
		return cfg.STTModel
	case "llm":
		return cfg.LLMModel
	case "tts":
		return cfg.TTSModel
	case "gemini_tts":
		return cfg.GeminiTTSModel
	case "vision":
		return cfg.VisionModel
	case "grok":
		return cfg.GrokModel
	case "qwen":
		return cfg.QwenModel
	case "gemini_live":
		return cfg.GeminiLiveModel
	default:
		return ""
	}
}

func listModelsResponse(cfg OpenAICfg, kind string) map[string]interface{} {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		kind = "realtime"
	}
	current := currentModelForKind(cfg, kind)
	defaults := modelKindDefaults(kind)
	out := map[string]interface{}{
		"status":  "success",
		"kind":    kind,
		"current": current,
	}
	ids, source, err := listModelsForKind(cfg, kind)
	if err != nil || source != "api" {
		out["source"] = "defaults"
		out["models"] = mergeModelLists(nil, defaults, current)
		if err != nil {
			out["error"] = err.Error()
			if strings.Contains(err.Error(), "chưa nhập") {
				out["message"] = err.Error() + " — đang hiện model mặc định. Nhập key rồi bấm Tải danh sách."
			}
		}
		return out
	}
	out["source"] = "api"
	out["models"] = mergeModelLists(ids, defaults, current)
	return out
}

func fetchModelIDs(u string, auth func(http.Header)) ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	auth(req.Header)
	resp, err := openAITestClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &e) == nil && e.Error.Message != "" {
			return nil, fmt.Errorf("%s: %s", resp.Status, e.Error.Message)
		}
		return nil, fmt.Errorf("%s", resp.Status)
	}
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("bad /models response")
	}
	ids := make([]string, 0, len(out.Data)+len(out.Models))
	for _, m := range out.Data {
		ids = append(ids, m.ID)
	}
	for _, m := range out.Models {
		ids = append(ids, strings.TrimPrefix(m.Name, "models/"))
	}
	return ids, nil
}

// modelListed tolerates "models/" prefixes (Gemini's OpenAI layer).
func modelListed(ids []string, model string) bool {
	for _, id := range ids {
		if id == model || strings.TrimPrefix(id, "models/") == model {
			return true
		}
	}
	return false
}

// testOpenAI checks every stage the current mode uses, each against its own
// endpoint and key. Free engines without an account are reported as such.
func testOpenAI(cfg OpenAICfg) openAITestResult {
	res := openAITestResult{Status: "success", OK: true}
	cache := map[string][]string{}
	check := func(stage, base, key, model string) {
		c := openAICheck{Stage: stage, Host: urlHost(base), Model: model}
		defer func() {
			if !c.OK {
				res.OK = false
			}
			res.Checks = append(res.Checks, c)
		}()
		if key == "" && !isLocalHost(urlHost(base)) {
			c.Error = "API key chưa nhập"
			return
		}
		ck := base + "\x00" + key
		ids, ok := cache[ck]
		if !ok {
			var err error
			if ids, err = listModels(base, key); err != nil {
				c.Error = err.Error()
				return
			}
			cache[ck] = ids
		}
		c.OK, c.Models, c.Found = true, len(ids), modelListed(ids, model)
	}

	if cfg.Mode != openAIModePipeline {
		check("realtime", cfg.BaseURL, cfg.APIKey, cfg.RealtimeModel)
		return res
	}
	check("stt", cfg.sttBase(), cfg.sttKey(), cfg.STTModel)
	check("llm", cfg.llmBase(), cfg.llmKey(), cfg.LLMModel)
	switch cfg.TTSEngine {
	case ttsEngineGoogle, ttsEngineEdge:
		res.Checks = append(res.Checks, openAICheck{Stage: "tts", OK: true, Free: true, Model: cfg.TTSEngine})
	case ttsEngineGemini:
		c := openAICheck{Stage: "tts", Host: geminiAPIHost, Model: cfg.GeminiTTSModel}
		if key := cfg.geminiKey(); key == "" {
			c.Error = "Gemini API key chưa nhập"
		} else if ids, err := listGeminiModels(key); err != nil {
			c.Error = err.Error()
		} else {
			c.OK, c.Models, c.Found = true, len(ids), modelListed(ids, cfg.GeminiTTSModel)
		}
		if !c.OK {
			res.OK = false
		}
		res.Checks = append(res.Checks, c)
	default:
		if strings.TrimSpace(cfg.APIKey) == "" {
			res.Checks = append(res.Checks, openAICheck{Stage: "tts", OK: true, Free: true, Model: ttsEngineGoogle,
				Error: "không có OpenAI key, robot sẽ dùng giọng Google"})
		} else {
			check("tts", cfg.BaseURL, cfg.APIKey, cfg.TTSModel)
		}
	}
	return res
}

var geminiLiveURL = "wss://" + geminiAPIHost + "/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent"

// liveSetupError opens a Live session (setup only, no audio) — listing the
// model is not enough: a key can see models it has no quota for.
func liveSetupError(key, model string) error {
	h := http.Header{}
	h.Set("x-goog-api-key", key)
	d := websocket.Dialer{HandshakeTimeout: 10 * time.Second, Proxy: http.ProxyFromEnvironment}
	ws, resp, err := d.Dial(geminiLiveURL, h)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("%s", resp.Status)
		}
		return err
	}
	defer ws.Close()
	deadline := time.Now().Add(12 * time.Second)
	_ = ws.SetWriteDeadline(deadline)
	_ = ws.SetReadDeadline(deadline)
	setup := map[string]interface{}{"setup": map[string]interface{}{
		"model":            "models/" + strings.TrimPrefix(model, "models/"),
		"generationConfig": map[string]interface{}{"responseModalities": []string{"AUDIO"}},
	}}
	if err := ws.WriteJSON(setup); err != nil {
		return err
	}
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			var ce *websocket.CloseError
			if errors.As(err, &ce) && ce.Text != "" {
				return errors.New(ce.Text)
			}
			return err
		}
		var m struct {
			SetupComplete json.RawMessage `json:"setupComplete"`
		}
		if json.Unmarshal(data, &m) == nil && m.SetupComplete != nil {
			return nil
		}
	}
}

func testGeminiLive(cfg OpenAICfg) openAITestResult {
	c := openAICheck{Stage: "live", Host: geminiAPIHost, Model: cfg.GeminiLiveModel}
	key := cfg.geminiKey()
	switch {
	case key == "":
		c.Error = "Gemini API key chưa nhập"
	default:
		ids, err := listGeminiModels(key)
		if err != nil {
			c.Error = err.Error()
			break
		}
		c.Models, c.Found = len(ids), modelListed(ids, cfg.GeminiLiveModel)
		if !c.Found {
			c.OK = true
			break
		}
		if err := liveSetupError(key, cfg.GeminiLiveModel); err != nil {
			c.Error = err.Error()
			break
		}
		c.OK = true
	}
	return openAITestResult{Status: "success", OK: c.OK && c.Found, Checks: []openAICheck{c}}
}

var grokRealtimeURL = "wss://api.x.ai/v1/realtime"

// grokSetupError opens an xAI Voice session and configures it (no audio, so
// nothing is billed): proves the key, the model and the voice together.
func grokSetupError(key, model, voice string) error {
	h := http.Header{}
	h.Set("Authorization", "Bearer "+key)
	d := websocket.Dialer{HandshakeTimeout: 10 * time.Second, Proxy: http.ProxyFromEnvironment}
	ws, resp, err := d.Dial(grokRealtimeURL+"?model="+url.QueryEscape(model), h)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 400))
			resp.Body.Close()
			var e struct {
				Error interface{} `json:"error"`
			}
			if json.Unmarshal(body, &e) == nil && e.Error != nil {
				return fmt.Errorf("%s: %v", resp.Status, e.Error)
			}
			return fmt.Errorf("%s", resp.Status)
		}
		return err
	}
	defer ws.Close()
	deadline := time.Now().Add(12 * time.Second)
	_ = ws.SetWriteDeadline(deadline)
	_ = ws.SetReadDeadline(deadline)
	if err := ws.WriteJSON(map[string]interface{}{
		"type":    "session.update",
		"session": map[string]interface{}{"voice": voice},
	}); err != nil {
		return err
	}
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			var ce *websocket.CloseError
			if errors.As(err, &ce) && ce.Text != "" {
				return errors.New(ce.Text)
			}
			return err
		}
		var m struct {
			Type  string `json:"type"`
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		switch m.Type {
		case "session.updated":
			return nil
		case "error":
			return errors.New(m.Error.Message)
		}
	}
}

func testGrok(cfg OpenAICfg) openAITestResult {
	c := openAICheck{Stage: "grok", Host: urlHost(grokRealtimeURL), Model: cfg.GrokModel}
	if key := strings.TrimSpace(cfg.XAIAPIKey); key == "" {
		c.Error = "xAI API key chưa nhập"
	} else if err := grokSetupError(key, cfg.GrokModel, cfg.GrokVoice); err != nil {
		c.Error = err.Error()
	} else {
		c.OK, c.Found = true, true
	}
	return openAITestResult{Status: "success", OK: c.OK, Checks: []openAICheck{c}}
}

func qwenRealtimeURL(region string) string {
	if region == "cn" {
		return "wss://dashscope.aliyuncs.com/api-ws/v1/realtime"
	}
	return "wss://dashscope-intl.aliyuncs.com/api-ws/v1/realtime"
}

// qwenSetupError opens a DashScope Qwen-Audio-Realtime session (no audio billed).
func qwenSetupError(key, region, model, voice string) error {
	h := http.Header{}
	h.Set("Authorization", "Bearer "+key)
	h.Set("X-DashScope-DataInspection", "enable")
	d := websocket.Dialer{HandshakeTimeout: 10 * time.Second, Proxy: http.ProxyFromEnvironment}
	ws, resp, err := d.Dial(qwenRealtimeURL(region)+"?model="+url.QueryEscape(model), h)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 400))
			resp.Body.Close()
			var e struct {
				Error interface{} `json:"error"`
			}
			if json.Unmarshal(body, &e) == nil && e.Error != nil {
				return fmt.Errorf("%s: %v", resp.Status, e.Error)
			}
			return fmt.Errorf("%s", resp.Status)
		}
		return err
	}
	defer ws.Close()
	deadline := time.Now().Add(12 * time.Second)
	_ = ws.SetWriteDeadline(deadline)
	_ = ws.SetReadDeadline(deadline)
	if err := ws.WriteJSON(map[string]interface{}{
		"type": "session.update",
		"session": map[string]interface{}{
			"modalities":          []string{"text", "audio"},
			"voice":               voice,
			"input_audio_format":  "pcm",
			"output_audio_format": "pcm",
			"turn_detection":      map[string]interface{}{"type": "server_vad"},
		},
	}); err != nil {
		return err
	}
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			var ce *websocket.CloseError
			if errors.As(err, &ce) && ce.Text != "" {
				return errors.New(ce.Text)
			}
			return err
		}
		var m struct {
			Type  string `json:"type"`
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		switch m.Type {
		case "session.updated":
			return nil
		case "error":
			return errors.New(m.Error.Message)
		}
	}
}

func testQwen(cfg OpenAICfg) openAITestResult {
	host := urlHost(qwenRealtimeURL(cfg.QwenRegion))
	c := openAICheck{Stage: "qwen", Host: host, Model: cfg.QwenModel}
	if key := strings.TrimSpace(cfg.DashScopeAPIKey); key == "" {
		c.Error = "DashScope API key chưa nhập"
	} else if err := qwenSetupError(key, cfg.QwenRegion, cfg.QwenModel, cfg.QwenVoice); err != nil {
		c.Error = err.Error()
	} else {
		c.OK, c.Found = true, true
	}
	return openAITestResult{Status: "success", OK: c.OK, Checks: []openAICheck{c}}
}

type OpenAI struct{}

func NewOpenAI() *OpenAI { return &OpenAI{} }

func (m *OpenAI) Name() string { return "OpenAI" }
func (m *OpenAI) Description() string {
	return "OpenAI / Gemini Live / Grok / Qwen backends for AI giao tiếp (Realtime, STT→LLM→TTS, Gemini Live, Grok Voice, Qwen Live)."
}
func (m *OpenAI) Load() error { return nil }

func (m *OpenAI) HTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch {
	case vars.IsEndpoint(r, "status"):
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "config": openAIView(loadOpenAICfg())})

	case vars.IsEndpoint(r, "save"):
		if r.Method != http.MethodPost {
			vars.HTTPError(w, r, "POST required")
			return
		}
		cfg := loadOpenAICfg()
		if err := applyOpenAIForm(&cfg, r); err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		if err := saveOpenAICfg(cfg); err != nil {
			vars.HTTPError(w, r, "save failed: "+err.Error())
			return
		}
		applied := false
		if xz, err := loadXiaozhiCfg(); err == nil && xz.Enabled && isDirectAPIBackend(xz.Backend) {
			applied = applyVicCloudConfig()
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success", "config": openAIView(cfg), "applied": applied,
		})

	case vars.IsEndpoint(r, "test"):
		cfg := loadOpenAICfg()
		if r.Method == http.MethodPost {
			if err := applyOpenAIForm(&cfg, r); err != nil {
				vars.HTTPError(w, r, err.Error())
				return
			}
		}
		switch r.FormValue("target") {
		case xiaozhiBackendGemini:
			json.NewEncoder(w).Encode(testGeminiLive(cfg))
			return
		case xiaozhiBackendGrok:
			json.NewEncoder(w).Encode(testGrok(cfg))
			return
		case xiaozhiBackendQwen:
			json.NewEncoder(w).Encode(testQwen(cfg))
			return
		}
		json.NewEncoder(w).Encode(testOpenAI(cfg))

	case vars.IsEndpoint(r, "list_gemini_live_models"):
		// Kept for older UI; prefer list_models?kind=gemini_live.
		cfg := loadOpenAICfg()
		if r.Method == http.MethodPost {
			if err := applyOpenAIForm(&cfg, r); err != nil {
				vars.HTTPError(w, r, err.Error())
				return
			}
		}
		json.NewEncoder(w).Encode(listModelsResponse(cfg, "gemini_live"))

	case vars.IsEndpoint(r, "list_models"):
		cfg := loadOpenAICfg()
		if r.Method == http.MethodPost {
			if err := applyOpenAIForm(&cfg, r); err != nil {
				vars.HTTPError(w, r, err.Error())
				return
			}
		}
		kind := strings.TrimSpace(r.FormValue("kind"))
		if kind == "" {
			kind = "realtime"
		}
		json.NewEncoder(w).Encode(listModelsResponse(cfg, kind))

	case vars.IsEndpoint(r, "clear_memory"):
		if err := os.Remove(openAIMemoryPath); err != nil && !os.IsNotExist(err) {
			vars.HTTPError(w, r, err.Error())
			return
		}
		vars.HTTPSuccess(w, r)

	default:
		vars.HTTPError(w, r, "unknown endpoint")
	}
}
