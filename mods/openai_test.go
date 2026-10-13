package mods

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func withTempOpenAIPaths(t *testing.T) string {
	t.Helper()
	dir := withTempXiaozhiPaths(t)
	oldCfg, oldMem := openAIConfigPath, openAIMemoryPath
	oldFlag, oldSup := xiaozhiReloadFlagPath, xiaozhiReloadSupportedPath
	openAIConfigPath = filepath.Join(dir, "persistent", "openai.json")
	openAIMemoryPath = filepath.Join(dir, "persistent", "openai_memory.json")
	xiaozhiReloadFlagPath = filepath.Join(dir, "run", "xiaozhi-reload")
	xiaozhiReloadSupportedPath = filepath.Join(dir, "run", "xiaozhi-reload-supported")
	_ = os.MkdirAll(filepath.Join(dir, "run"), 0775)
	_ = os.WriteFile(xiaozhiReloadSupportedPath, []byte("1"), 0644)
	t.Cleanup(func() {
		openAIConfigPath, openAIMemoryPath = oldCfg, oldMem
		xiaozhiReloadFlagPath, xiaozhiReloadSupportedPath = oldFlag, oldSup
	})
	return dir
}

func openAIPost(t *testing.T, endpoint string, form url.Values) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/mods/OpenAI/"+endpoint, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	NewOpenAI().HTTP(w, req)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return w, resp
}

func TestOpenAISaveKeepsAndMasksKey(t *testing.T) {
	withTempOpenAIPaths(t)
	w, resp := openAIPost(t, "save", url.Values{"api_key": {"sk-test-123456789abcd"}, "mode": {"pipeline"}})
	if w.Code != 200 {
		t.Fatalf("save HTTP %d %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "sk-test") {
		t.Fatal("API key leaked in save response")
	}
	view := resp["config"].(map[string]interface{})
	if view["has_key"] != true || view["key_hint"] != "…abcd" || view["mode"] != "pipeline" {
		t.Fatalf("view %v", view)
	}

	// Empty key field keeps the stored key.
	openAIPost(t, "save", url.Values{"api_key": {""}, "voice": {"alloy"}})
	cfg := loadOpenAICfg()
	if cfg.APIKey != "sk-test-123456789abcd" || cfg.Voice != "alloy" || cfg.Mode != "pipeline" {
		t.Fatalf("after partial save %+v", cfg)
	}
	if st, err := os.Stat(openAIConfigPath); err != nil || st.Mode().Perm() != 0640 {
		t.Fatalf("openai.json mode %v err %v", st.Mode().Perm(), err)
	}

	openAIPost(t, "save", url.Values{"clear_key": {"1"}})
	if loadOpenAICfg().APIKey != "" {
		t.Fatal("clear_key kept the key")
	}

	req := httptest.NewRequest("GET", "/api/mods/OpenAI/status", nil)
	sw := httptest.NewRecorder()
	NewOpenAI().HTTP(sw, req)
	if strings.Contains(sw.Body.String(), "api_key") {
		t.Fatalf("status exposes key field: %s", sw.Body.String())
	}
}

func TestOpenAISaveValidates(t *testing.T) {
	withTempOpenAIPaths(t)
	for _, form := range []url.Values{
		{"mode": {"magic"}},
		{"base_url": {"ftp://x"}},
		{"tts_engine": {"espeak"}},
		{"memory_turns": {"99"}},
		{"instructions": {strings.Repeat("a", openAIMaxInstruction+1)}},
	} {
		if w, _ := openAIPost(t, "save", form); w.Code == 200 {
			t.Fatalf("accepted %v", form)
		}
	}
	if _, err := os.Stat(openAIConfigPath); err == nil {
		t.Fatal("config written for rejected input")
	}
}

func TestOpenAISaveReloadsOnlyWhenActive(t *testing.T) {
	withTempOpenAIPaths(t)
	cfg := defaultXiaozhiCfg()
	if err := saveXiaozhiCfg(cfg); err != nil {
		t.Fatal(err)
	}
	if _, resp := openAIPost(t, "save", url.Values{"voice": {"ash"}}); resp["applied"] != false {
		t.Fatalf("reload while Xiaozhi backend: %v", resp)
	}
	cfg.Backend = xiaozhiBackendOpenAI
	_ = saveXiaozhiCfg(cfg)
	if _, resp := openAIPost(t, "save", url.Values{"voice": {"ash"}}); resp["applied"] != true {
		t.Fatalf("no reload while OpenAI backend: %v", resp)
	}
}

func TestXiaozhiSetEnabledBackend(t *testing.T) {
	withTempOpenAIPaths(t)
	_ = saveXiaozhiCfg(defaultXiaozhiCfg())
	before, _ := loadXiaozhiCfg()

	req := httptest.NewRequest("POST", "/api/mods/Xiaozhi/set_enabled?enabled=true&backend=openai", nil)
	w := httptest.NewRecorder()
	NewXiaozhi().HTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("HTTP %d %s", w.Code, w.Body.String())
	}
	cfg, _ := loadXiaozhiCfg()
	if cfg.Backend != xiaozhiBackendOpenAI || !cfg.Enabled {
		t.Fatalf("backend %q enabled %v", cfg.Backend, cfg.Enabled)
	}
	if cfg.DeviceID != before.DeviceID || cfg.IdentityMode != before.IdentityMode {
		t.Fatal("OpenAI switch touched the Xiaozhi identity")
	}
	if !flagExists() {
		t.Fatal("OpenAI backend must set the listen VAD flag")
	}
	if _, err := os.Stat(xiaozhiReloadFlagPath); err != nil {
		t.Fatal("backend change did not request a vic-cloud reload")
	}

	// Older clients picking a Xiaozhi preset switch back to Xiaozhi.
	req = httptest.NewRequest("POST", "/api/mods/Xiaozhi/set_enabled?enabled=true&identity_mode=vi_pool", nil)
	NewXiaozhi().HTTP(httptest.NewRecorder(), req)
	cfg, _ = loadXiaozhiCfg()
	if cfg.Backend != "" {
		t.Fatalf("identity_mode did not select Xiaozhi: %q", cfg.Backend)
	}
}

// "Tự chọn AI" and "OpenAI Realtime" are separate choices on the page: both
// run as backend openai, the choice sets openai.json's mode.
func TestXiaozhiSetEnabledPipelineChoice(t *testing.T) {
	withTempOpenAIPaths(t)
	_ = saveXiaozhiCfg(defaultXiaozhiCfg())
	set := func(backend string) map[string]interface{} {
		w := httptest.NewRecorder()
		NewXiaozhi().HTTP(w, httptest.NewRequest("POST", "/api/mods/Xiaozhi/set_enabled?enabled=true&backend="+backend, nil))
		var resp map[string]interface{}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &resp) != nil {
			t.Fatalf("HTTP %d %s", w.Code, w.Body.String())
		}
		return resp
	}
	resp := set("pipeline")
	cfg, _ := loadXiaozhiCfg()
	if cfg.Backend != xiaozhiBackendOpenAI || loadOpenAICfg().Mode != openAIModePipeline {
		t.Fatalf("pipeline: backend %q mode %q", cfg.Backend, loadOpenAICfg().Mode)
	}
	if resp["config"].(map[string]interface{})["openai_mode"] != openAIModePipeline {
		t.Fatalf("view %v", resp["config"])
	}
	_ = os.Remove(xiaozhiReloadFlagPath)
	// Same backend, other mode: still a change the robot must pick up.
	resp = set("openai")
	if loadOpenAICfg().Mode != openAIModeRealtime || resp["applied"] != true && resp["restarted"] != true {
		t.Fatalf("realtime: mode %q resp %v", loadOpenAICfg().Mode, resp)
	}
	set("gemini")
	if loadOpenAICfg().Mode != openAIModeRealtime {
		t.Fatal("Gemini Live must not touch the OpenAI mode")
	}
}

func TestXiaozhiBackendFor(t *testing.T) {
	cases := []struct{ cur, arg, ident, want string }{
		{"", "openai", "", "openai"},
		{"openai", "xiaozhi", "", ""},
		{"openai", "", "vi_pool", ""},
		{"openai", "", "", "openai"},
		{"", "gemini", "", "gemini"},
		{"gemini", "", "vi_pool", ""},
		{"", "grok", "", "grok"},
		{"grok", "", "", "grok"},
		{"", "pipeline", "", "openai"},
		{"", "", "", ""},
	}
	for _, c := range cases {
		if got := xiaozhiBackendFor(c.cur, c.arg, c.ident); got != c.want {
			t.Fatalf("%+v → %q", c, got)
		}
	}
}

func TestOpenAITestEndpoint(t *testing.T) {
	withTempOpenAIPaths(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer good" {
			w.WriteHeader(401)
			fmt.Fprint(w, `{"error":{"message":"Incorrect API key provided"}}`)
			return
		}
		fmt.Fprint(w, `{"data":[{"id":"gpt-realtime-mini"},{"id":"gpt-4o-mini"}]}`)
	}))
	defer srv.Close()

	_, resp := openAIPost(t, "test", url.Values{"api_key": {"good"}, "base_url": {srv.URL + "/v1"}})
	c := firstCheck(t, resp)
	if resp["ok"] != true || c["stage"] != "realtime" || c["model_found"] != true || c["models"] != float64(2) {
		t.Fatalf("good key: %v", resp)
	}
	_, resp = openAIPost(t, "test", url.Values{"api_key": {"bad"}, "base_url": {srv.URL + "/v1"}})
	if resp["ok"] != false || !strings.Contains(fmt.Sprint(firstCheck(t, resp)["error"]), "Incorrect API key") {
		t.Fatalf("bad key: %v", resp)
	}
	if _, err := os.Stat(openAIConfigPath); err == nil {
		t.Fatal("test endpoint must not save")
	}
}

func firstCheck(t *testing.T, resp map[string]interface{}) map[string]interface{} {
	t.Helper()
	checks, _ := resp["checks"].([]interface{})
	if len(checks) == 0 {
		t.Fatalf("no checks: %v", resp)
	}
	return checks[0].(map[string]interface{})
}

// A free pipeline (Groq STT+LLM, Edge voice) needs no OpenAI key and the
// Groq server never sees the OpenAI key.
func TestOpenAITestFreePipeline(t *testing.T) {
	withTempOpenAIPaths(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer gsk-test-free" {
			w.WriteHeader(401)
			fmt.Fprintf(w, `{"error":{"message":"got %q"}}`, r.Header.Get("Authorization"))
			return
		}
		fmt.Fprint(w, `{"data":[{"id":"whisper-large-v3-turbo"},{"id":"openai/gpt-oss-120b"}]}`)
	}))
	defer srv.Close()
	_, resp := openAIPost(t, "save", url.Values{
		"api_key": {"sk-openai-secret"}, "mode": {"pipeline"},
		"stt_base_url": {srv.URL + "/openai/v1"}, "stt_api_key": {"gsk-test-free"}, "stt_model": {"whisper-large-v3-turbo"},
		"llm_base_url": {srv.URL + "/openai/v1"}, "llm_model": {"openai/gpt-oss-120b"},
		"tts_engine": {"edge"}, "edge_voice": {"vi-VN-HoaiMyNeural"}, "language": {"AUTO"},
	})
	view := resp["config"].(map[string]interface{})
	if view["has_stt_key"] != true || view["stt_key_hint"] != "…free" || view["edge_voice"] != "vi-VN-HoaiMyNeural" || view["language"] != "auto" {
		t.Fatalf("view %v", view)
	}
	_, resp = openAIPost(t, "test", url.Values{})
	if resp["ok"] != true {
		t.Fatalf("free pipeline test: %v", resp)
	}
	checks := resp["checks"].([]interface{})
	if len(checks) != 3 || checks[2].(map[string]interface{})["free"] != true {
		t.Fatalf("checks %v", checks)
	}
	for _, c := range checks[:2] {
		if c.(map[string]interface{})["model_found"] != true {
			t.Fatalf("model not found: %v", c)
		}
	}

	_, resp = openAIPost(t, "save", url.Values{"api_key": {""}, "clear_key": {"1"}})
	if resp["config"].(map[string]interface{})["needs_openai_key"] != false {
		t.Fatal("free pipeline must not need the OpenAI key")
	}
	_, resp = openAIPost(t, "save", url.Values{"llm_base_url": {""}})
	if resp["config"].(map[string]interface{})["needs_openai_key"] != true {
		t.Fatal("OpenAI LLM without key must need the OpenAI key")
	}
}

func TestOpenAIGeminiKeyAndValidation(t *testing.T) {
	withTempOpenAIPaths(t)
	w, resp := openAIPost(t, "save", url.Values{"tts_engine": {"gemini"}, "gemini_api_key": {"AIza-secret-key-wxyz"}, "gemini_voice": {"Puck"}})
	if w.Code != 200 || strings.Contains(w.Body.String(), "AIza") {
		t.Fatalf("save %d %s", w.Code, w.Body.String())
	}
	view := resp["config"].(map[string]interface{})
	if view["has_gemini_key"] != true || view["gemini_key_hint"] != "…wxyz" || view["gemini_voice"] != "Puck" {
		t.Fatalf("view %v", view)
	}
	openAIPost(t, "save", url.Values{"clear_gemini_key": {"1"}})
	if loadOpenAICfg().GeminiAPIKey != "" {
		t.Fatal("clear_gemini_key kept the key")
	}
	for _, form := range []url.Values{
		{"edge_voice": {"x' onload='1"}},
		{"gemini_tts_model": {"../../evil?x"}},
		{"stt_base_url": {"file:///etc"}},
	} {
		if w, _ := openAIPost(t, "save", form); w.Code == 200 {
			t.Fatalf("accepted %v", form)
		}
	}
}

func TestOpenAIMissingKeys(t *testing.T) {
	c := OpenAICfg{Mode: openAIModePipeline, STTBaseURL: "https://api.groq.com/openai/v1",
		LLMBaseURL: "https://generativelanguage.googleapis.com/v1beta/openai", TTSEngine: ttsEngineEdge}
	if got := fmt.Sprint(openAIMissingKeys(c)); got != "[stt llm]" {
		t.Fatalf("free preset without keys: %s", got)
	}
	c.LLMBaseURL = "http://192.168.1.10:11434/v1"
	c.GroqAPIKey = "gsk"
	c.TTSEngine = ttsEngineGemini
	if got := fmt.Sprint(openAIMissingKeys(c)); got != "[tts]" {
		t.Fatalf("local LLM + gemini TTS without key: %s", got)
	}
	c.Mode = openAIModeRealtime
	if len(openAIMissingKeys(c)) != 0 {
		t.Fatal("realtime has no pipeline keys")
	}
}

func TestOpenAICfgKeyRouting(t *testing.T) {
	groq := "https://api.groq.com/openai/v1"
	gem := "https://generativelanguage.googleapis.com/v1beta/openai"
	c := OpenAICfg{APIKey: "sk", GroqAPIKey: "gsk", GeminiAPIKey: "gem", STTAPIKey: "custom-stt",
		STTBaseURL: groq, LLMBaseURL: gem}
	if c.sttKey() != "gsk" || c.llmKey() != "gem" {
		t.Fatalf("provider keys: stt %q llm %q", c.sttKey(), c.llmKey())
	}
	c.LLMBaseURL = groq
	if c.llmKey() != "gsk" {
		t.Fatal("switching the LLM to Groq must use the Groq key")
	}
	c.STTBaseURL = ""
	if c.sttKey() != "sk" {
		t.Fatal("OpenAI step uses the OpenAI key")
	}
	c.STTBaseURL, c.LLMBaseURL = "https://my.server/v1", "https://my.server/v1"
	if c.sttKey() != "custom-stt" || c.llmKey() != "custom-stt" {
		t.Fatal("same custom host should share the step key")
	}
	c.LLMBaseURL = "https://other.server/v1"
	if c.llmKey() != "" {
		t.Fatal("a different custom host must not get another server's key")
	}
}

// Old files kept the Groq key in stt_api_key and the Gemini key in
// llm_api_key; they move to the provider fields once.
func TestOpenAILegacyKeysMigrate(t *testing.T) {
	withTempOpenAIPaths(t)
	legacy := `{"mode":"pipeline","stt_base_url":"https://api.groq.com/openai/v1","stt_api_key":"gsk-old-key-abcd",
		"llm_base_url":"https://generativelanguage.googleapis.com/v1beta/openai","llm_api_key":"AIza-old"}`
	_ = os.MkdirAll(filepath.Dir(openAIConfigPath), 0o755)
	if err := os.WriteFile(openAIConfigPath, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	c := loadOpenAICfg()
	if c.GroqAPIKey != "gsk-old-key-abcd" || c.GeminiAPIKey != "AIza-old" || c.STTAPIKey != "" || c.LLMAPIKey != "" {
		t.Fatalf("migrated %+v", c)
	}
	// The user switches step 2 to Groq: the Groq key follows, no retyping.
	_, resp := openAIPost(t, "save", url.Values{"llm_base_url": {"https://api.groq.com/openai/v1"}, "llm_model": {"openai/gpt-oss-120b"}})
	view := resp["config"].(map[string]interface{})
	if view["groq_key_hint"] != "…abcd" || view["has_llm_key"] != false || fmt.Sprint(view["missing_keys"]) != "[]" {
		t.Fatalf("view %v", view)
	}
	// Once saved in the new layout, a custom-server key is never moved.
	openAIPost(t, "save", url.Values{"stt_base_url": {"https://my.server/v1"}, "stt_api_key": {"srv-key"}})
	openAIPost(t, "save", url.Values{"stt_base_url": {"https://api.groq.com/openai/v1"}})
	if c := loadOpenAICfg(); c.GroqAPIKey != "gsk-old-key-abcd" || c.STTAPIKey != "srv-key" || c.sttKey() != "gsk-old-key-abcd" {
		t.Fatalf("after switching back %+v", c)
	}
	// A key typed into a Groq step from an older page lands in the Groq slot.
	openAIPost(t, "save", url.Values{"stt_api_key": {"gsk-new-key-efgh"}})
	if c := loadOpenAICfg(); c.GroqAPIKey != "gsk-new-key-efgh" || c.STTAPIKey != "srv-key" {
		t.Fatalf("old-page key %+v", c)
	}
}

func TestXiaozhiSetEnabledGeminiBackend(t *testing.T) {
	withTempOpenAIPaths(t)
	_ = saveXiaozhiCfg(defaultXiaozhiCfg())
	before, _ := loadXiaozhiCfg()
	req := httptest.NewRequest("POST", "/api/mods/Xiaozhi/set_enabled?enabled=true&backend=gemini", nil)
	w := httptest.NewRecorder()
	NewXiaozhi().HTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("HTTP %d %s", w.Code, w.Body.String())
	}
	cfg, _ := loadXiaozhiCfg()
	if cfg.Backend != xiaozhiBackendGemini || !cfg.Enabled {
		t.Fatalf("backend %q enabled %v", cfg.Backend, cfg.Enabled)
	}
	if cfg.DeviceID != before.DeviceID || cfg.IdentityMode != before.IdentityMode {
		t.Fatal("Gemini switch touched the Xiaozhi identity")
	}
	if !flagExists() {
		t.Fatal("Gemini backend must set the listen VAD flag")
	}
	if _, resp := openAIPost(t, "save", url.Values{"gemini_live_voice": {"Puck"}}); resp["applied"] != true {
		t.Fatalf("no reload while Gemini backend: %v", resp)
	}
	if got := xiaozhiBackendFor("gemini", "", ""); got != "gemini" {
		t.Fatalf("current gemini backend not kept: %q", got)
	}
}

func TestOpenAIGeminiLiveFields(t *testing.T) {
	withTempOpenAIPaths(t)
	_, resp := openAIPost(t, "save", url.Values{})
	view := resp["config"].(map[string]interface{})
	if view["gemini_live_model"] != "gemini-3.8-live" || view["gemini_live_voice"] != "Kore" || view["has_gemini_live_key"] != false {
		t.Fatalf("defaults %v", view)
	}
	// The Gemini LLM key (free preset) also opens Gemini Live.
	_, resp = openAIPost(t, "save", url.Values{"llm_base_url": {"https://" + geminiAPIHost + "/v1beta/openai"},
		"llm_api_key": {"AIza-llm-key-1234"}, "gemini_live_model": {"gemini-2.5-flash-native-audio-latest"}, "gemini_live_voice": {"Aoede"}})
	view = resp["config"].(map[string]interface{})
	if view["has_gemini_live_key"] != true || view["gemini_live_model"] != "gemini-2.5-flash-native-audio-latest" || view["gemini_live_voice"] != "Aoede" {
		t.Fatalf("view %v", view)
	}
	for _, form := range []url.Values{
		{"gemini_live_model": {"x y"}},
		{"gemini_live_voice": {"<b>"}},
	} {
		if w, _ := openAIPost(t, "save", form); w.Code == 200 {
			t.Fatalf("accepted %v", form)
		}
	}
}

// The Live check lists models, then opens a real setup: a listed model can
// still be refused (no free-tier quota).
func TestGeminiLiveTestEndpoint(t *testing.T) {
	withTempOpenAIPaths(t)
	up := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-goog-api-key") != "AIza-good" {
			w.WriteHeader(400)
			fmt.Fprint(w, `{"error":{"message":"API key not valid"}}`)
			return
		}
		if r.URL.Path == "/v1beta/models" {
			fmt.Fprint(w, `{"models":[{"name":"models/gemini-3.1-flash-live-preview"},{"name":"models/gemini-3.8-live"}]}`)
			return
		}
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		var m struct {
			Setup struct {
				Model string `json:"model"`
			} `json:"setup"`
		}
		if c.ReadJSON(&m) != nil {
			return
		}
		if m.Setup.Model == "models/gemini-3.1-flash-live-preview" {
			_ = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "You exceeded your current quota"))
			return
		}
		_ = c.WriteMessage(websocket.BinaryMessage, []byte(`{"setupComplete":{}}`))
	}))
	defer srv.Close()
	oldBase, oldURL := geminiModelsBase, geminiLiveURL
	geminiModelsBase = srv.URL + "/v1beta"
	geminiLiveURL = "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	defer func() { geminiModelsBase, geminiLiveURL = oldBase, oldURL }()

	_, resp := openAIPost(t, "test", url.Values{"target": {"gemini"}, "gemini_api_key": {"AIza-good"}})
	c := firstCheck(t, resp)
	if resp["ok"] != true || c["stage"] != "live" || c["model_found"] != true {
		t.Fatalf("free model: %v", resp)
	}
	_, resp = openAIPost(t, "test", url.Values{"target": {"gemini"}, "gemini_api_key": {"AIza-good"}, "gemini_live_model": {"gemini-3.1-flash-live-preview"}})
	if resp["ok"] != false || !strings.Contains(fmt.Sprint(firstCheck(t, resp)["error"]), "quota") {
		t.Fatalf("no-quota model: %v", resp)
	}
	_, resp = openAIPost(t, "test", url.Values{"target": {"gemini"}, "gemini_api_key": {"AIza-bad"}})
	if resp["ok"] != false || !strings.Contains(fmt.Sprint(firstCheck(t, resp)["error"]), "API key not valid") {
		t.Fatalf("bad key: %v", resp)
	}
	_, resp = openAIPost(t, "test", url.Values{"target": {"gemini"}})
	if resp["ok"] != false {
		t.Fatalf("no key: %v", resp)
	}
}

func TestOpenAIGrokFields(t *testing.T) {
	withTempOpenAIPaths(t)
	_, resp := openAIPost(t, "save", url.Values{})
	view := resp["config"].(map[string]interface{})
	if view["grok_model"] != "grok-voice-latest" || view["grok_voice"] != "eve" || view["has_xai_key"] != false {
		t.Fatalf("defaults %v", view)
	}
	_, resp = openAIPost(t, "save", url.Values{"xai_api_key": {"xai-secret-key-9876"}, "grok_voice": {"ara"}})
	view = resp["config"].(map[string]interface{})
	if view["has_xai_key"] != true || view["xai_key_hint"] != "…9876" || view["grok_voice"] != "ara" {
		t.Fatalf("view %v", view)
	}
	if strings.Contains(fmt.Sprint(view), "xai-secret") {
		t.Fatal("the xAI key leaked into the view")
	}
	for _, form := range []url.Values{
		{"grok_model": {"x y"}},
		{"grok_voice": {"<b>"}},
	} {
		if w, _ := openAIPost(t, "save", form); w.Code == 200 {
			t.Fatalf("accepted %v", form)
		}
	}

	_ = saveXiaozhiCfg(defaultXiaozhiCfg())
	req := httptest.NewRequest("POST", "/api/mods/Xiaozhi/set_enabled?enabled=true&backend=grok", nil)
	NewXiaozhi().HTTP(httptest.NewRecorder(), req)
	if cfg, _ := loadXiaozhiCfg(); cfg.Backend != xiaozhiBackendGrok || !cfg.Enabled || !flagExists() {
		t.Fatalf("grok backend %q enabled %v", cfg.Backend, cfg.Enabled)
	}
}

func TestGrokTestEndpoint(t *testing.T) {
	withTempOpenAIPaths(t)
	up := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer xai-good" {
			w.WriteHeader(401)
			fmt.Fprint(w, `{"error":"Incorrect API key provided"}`)
			return
		}
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		_ = c.WriteMessage(websocket.TextMessage, []byte(`{"type":"session.created"}`))
		var m struct {
			Session struct {
				Voice string `json:"voice"`
			} `json:"session"`
		}
		if c.ReadJSON(&m) != nil {
			return
		}
		if m.Session.Voice != "eve" {
			_ = c.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","error":{"message":"unknown voice"}}`))
			return
		}
		_ = c.WriteMessage(websocket.TextMessage, []byte(`{"type":"session.updated"}`))
	}))
	defer srv.Close()
	old := grokRealtimeURL
	grokRealtimeURL = "ws" + strings.TrimPrefix(srv.URL, "http")
	defer func() { grokRealtimeURL = old }()

	_, resp := openAIPost(t, "test", url.Values{"target": {"grok"}, "xai_api_key": {"xai-good"}})
	c := firstCheck(t, resp)
	if resp["ok"] != true || c["stage"] != "grok" || c["model_found"] != true {
		t.Fatalf("good key: %v", resp)
	}
	_, resp = openAIPost(t, "test", url.Values{"target": {"grok"}, "xai_api_key": {"xai-good"}, "grok_voice": {"nobody"}})
	if resp["ok"] != false || !strings.Contains(fmt.Sprint(firstCheck(t, resp)["error"]), "unknown voice") {
		t.Fatalf("bad voice: %v", resp)
	}
	_, resp = openAIPost(t, "test", url.Values{"target": {"grok"}, "xai_api_key": {"xai-bad"}})
	if resp["ok"] != false || !strings.Contains(fmt.Sprint(firstCheck(t, resp)["error"]), "Incorrect API key") {
		t.Fatalf("bad key: %v", resp)
	}
}
