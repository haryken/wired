package mods

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/os-vector/wired/vars"
)

var xiaozhiConfigPath = "/data/data/com.anki.victor/persistent/xiaozhi/xiaozhi.json"

// XiaozhiCfg mirrors the shared config JSON at xiaozhiConfigPath.
type XiaozhiCfg struct {
	Enabled               bool   `json:"enabled"`
	OTABaseURL            string `json:"ota_base_url"`
	Endpoint              string `json:"endpoint"`
	DeviceID              string `json:"device_id"`
	ClientID              string `json:"client_id"`
	Token                 string `json:"token"`
	ProtocolVersion       int    `json:"protocol_version"`
	AutoApplyOTAWebsocket bool   `json:"auto_apply_ota_websocket"`
	ConversationMode      string `json:"conversation_mode"`
	IdleTimeoutSec        int    `json:"idle_timeout_sec"`
	TTSMode               string `json:"tts_mode"`
	// GameGoogleTTSVI enables Google TTS game comments (always on in UI; kept for compat).
	GameGoogleTTSVI bool `json:"game_google_tts_vi"`
	// GameGoogleTTSLang is Google Translate TTS language (vi, en, zh-CN, …). Default vi.
	GameGoogleTTSLang string `json:"game_google_tts_lang,omitempty"`
	// IdentityMode: "vi_pool" = shared Vietnamese preset; "course" = English
	// teaching course pool; "custom" = xiaozhi.me pair.
	IdentityMode string `json:"identity_mode,omitempty"`
	// English-teaching course (see xiaozhi_course.go). Maps keyed by course
	// index; CourseUnit by "<course>.<book>" (unit 1-based).
	CourseIdx   int            `json:"course_idx,omitempty"`
	CourseVoice map[string]int `json:"course_voice,omitempty"`
	CourseBook  map[string]int `json:"course_book,omitempty"`
	CourseUnit  map[string]int `json:"course_unit,omitempty"`
	StudentName string         `json:"student_name,omitempty"`
	// ListenMode: "fixed" (default, ~6s mic) or "vad" (mic open until server STT).
	ListenMode string `json:"listen_mode,omitempty"`
	// MusicSource: "server" (default, Xiaozhi server music) or "youtube"
	// (vic-cloud self.music.play streaming from MusicServerURL).
	MusicSource    string `json:"music_source,omitempty"`
	MusicServerURL string `json:"music_server_url,omitempty"`
	// FaceGreet: vic-cloud greets ("xin chào" detect) when Vector sees a person arrive.
	FaceGreet bool `json:"face_greet,omitempty"`
}

const (
	xiaozhiListenFixed = "fixed"
	xiaozhiListenVAD   = "vad"

	xiaozhiMusicServer     = "server"
	xiaozhiMusicYouTube    = "youtube"
	xiaozhiMusicDefaultURL = "https://youtube.kytuoi.com"
)

// normalizeMusicServerURL mirrors vic-cloud's music.NormalizeServerURL.
func normalizeMusicServerURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return xiaozhiMusicDefaultURL
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "https://" + u
	}
	return strings.TrimRight(u, "/")
}

// Read by vic-anim / vic-engine / vic-cloud at stream start (tmpfs, owned anki:anki).
var xiaozhiListenVADFlagPath = "/run/vic-cloud/xiaozhi-listen-vad"

// syncXiaozhiListenVADFlag applies listen_mode to the next listen without
// waiting for vic-cloud to reload xiaozhi.json.
func syncXiaozhiListenVADFlag(cfg XiaozhiCfg) {
	if cfg.Enabled && cfg.ListenMode == xiaozhiListenVAD {
		if err := os.MkdirAll(filepath.Dir(xiaozhiListenVADFlagPath), 0775); err == nil {
			_ = os.WriteFile(xiaozhiListenVADFlagPath, []byte("1"), 0664)
		}
		return
	}
	_ = os.Remove(xiaozhiListenVADFlagPath)
}

func defaultXiaozhiCfg() XiaozhiCfg {
	return XiaozhiCfg{
		Enabled:               true,
		OTABaseURL:            "https://api.tenclass.net/",
		Endpoint:              "wss://api.tenclass.net/xiaozhi/v1/",
		ProtocolVersion:       1,
		AutoApplyOTAWebsocket: true,
		ConversationMode:      "continuous",
		IdleTimeoutSec:        20,
		TTSMode:               "xiaozhi",
		GameGoogleTTSVI:       true,
		GameGoogleTTSLang:     "vi",
		IdentityMode:          xiaozhiIdentityViPool,
	}
}

// ensureXiaozhiDefaults fills first-install blanks without wiping existing data.
func ensureXiaozhiDefaults(cfg *XiaozhiCfg) {
	if strings.TrimSpace(cfg.OTABaseURL) == "" {
		cfg.OTABaseURL = "https://api.tenclass.net/"
	} else {
		cfg.OTABaseURL = normalizeOTABaseURL(cfg.OTABaseURL)
	}
	cfg.AutoApplyOTAWebsocket = true
	if cfg.TTSMode != "xiaozhi" {
		cfg.TTSMode = "xiaozhi"
	}
	if cfg.ConversationMode == "" {
		cfg.ConversationMode = "continuous"
	}
	if cfg.IdleTimeoutSec <= 0 {
		cfg.IdleTimeoutSec = 20
	}
	if cfg.ProtocolVersion == 0 {
		cfg.ProtocolVersion = 1
	}
	// Game Google TTS is always available; language picked in Games lobby.
	cfg.GameGoogleTTSVI = true
	if strings.TrimSpace(cfg.GameGoogleTTSLang) == "" {
		cfg.GameGoogleTTSLang = "vi"
	}
}

func loadXiaozhiCfg() (XiaozhiCfg, error) {
	cfg := defaultXiaozhiCfg()
	data, err := os.ReadFile(xiaozhiConfigPath)
	if err != nil {
		return cfg, err
	}
	data = bytes.TrimSpace(data)
	if err := json.Unmarshal(data, &cfg); err != nil {
		// Tolerate truncated files missing a final '}' (seen on robot writes).
		if !bytes.HasSuffix(data, []byte("}")) {
			if err2 := json.Unmarshal(append(data, '}'), &cfg); err2 == nil {
				_ = saveXiaozhiCfg(cfg) // rewrite a valid file
				return cfg, nil
			}
		}
		return cfg, err
	}
	return cfg, nil
}

func saveXiaozhiCfg(cfg XiaozhiCfg) error {
	if err := vars.MkdirAll(filepath.Dir(xiaozhiConfigPath)); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(xiaozhiConfigPath, data, 0644); err != nil {
		return err
	}
	syncXiaozhiListenVADFlag(cfg)
	return nil
}

// normalizeOTABaseURL ensures https scheme and trailing slash.
func normalizeOTABaseURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return u
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "https://" + u
	}
	if !strings.HasSuffix(u, "/") {
		u += "/"
	}
	return u
}

// genRandomMAC returns a random locally-administered unicast MAC (aa:bb:cc:dd:ee:ff).
// Used when Device ID is left blank — not the robot NIC MAC (user may have typed that before).
func genRandomMAC() string {
	var b [6]byte
	rand.Read(b[:])
	b[0] = (b[0] | 0x02) & 0xfe // locally administered, unicast
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", b[0], b[1], b[2], b[3], b[4], b[5])
}

// genUUIDv4 generates a random UUID v4 without external dependencies.
func genUUIDv4() string {
	var b [16]byte
	rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

const (
	xiaozhiIdentityViPool = "vi_pool"
	xiaozhiIdentityCustom = "custom"
	xiaozhiTenclassOTA    = "https://api.tenclass.net/"
	xiaozhiTenclassWSS    = "wss://api.tenclass.net/xiaozhi/v1/"
	xiaozhiSkipRotatePath = "/run/xiaozhi-skip-rotate"
)

var xiaozhiViPoolMACs = []string{
	"1c:db:d4:b5:73:3c",
	"58:a0:23:a6:fe:31",
	"a8:b5:44:dd:e3:cf",
	"1c:db:d4:b5:74:7c",
	"1c:db:d4:b5:6a:d8",
	"1c:db:d4:b5:74:54",
	"1c:db:d4:b5:72:ec",
	"1c:db:d4:b5:71:d4",
	"1c:db:d4:b5:74:d4",
	"1c:db:d4:a9:48:84",
	"1c:db:d4:a9:59:b4",
	"1c:db:d4:a9:5b:a0",
	"28:df:eb:02:6c:7d",
	"bc:fc:e7:8a:d8:06",
	"dc:b4:d9:0c:a4:9c",
	"dc:b4:d9:0c:a4:80",
	"dc:b4:d9:0c:a6:00",
	"dc:b4:d9:03:4e:f4",
	"dc:b4:d9:0c:a5:38",
	"dc:b4:d9:03:43:38",
}

func macInXiaozhiViPool(mac string) bool {
	mac = strings.ToLower(strings.TrimSpace(mac))
	for _, m := range xiaozhiViPoolMACs {
		if m == mac {
			return true
		}
	}
	return false
}

func isXiaozhiViPool(cfg XiaozhiCfg) bool {
	mode := strings.ToLower(strings.TrimSpace(cfg.IdentityMode))
	if mode == xiaozhiIdentityCustom {
		return false
	}
	if mode == xiaozhiIdentityViPool {
		return true
	}
	return mode == "" && macInXiaozhiViPool(cfg.DeviceID)
}

func applyXiaozhiViPoolDefaults(cfg *XiaozhiCfg) {
	cfg.IdentityMode = xiaozhiIdentityViPool
	cfg.OTABaseURL = xiaozhiTenclassOTA
	cfg.Endpoint = xiaozhiTenclassWSS
	cfg.AutoApplyOTAWebsocket = true
	cfg.TTSMode = "xiaozhi"
	if cfg.ConversationMode == "" {
		cfg.ConversationMode = "continuous"
	}
}

func pickXiaozhiPoolMAC(avoid string) string {
	avoid = strings.ToLower(strings.TrimSpace(avoid))
	n := len(xiaozhiViPoolMACs)
	if n == 0 {
		return genRandomMAC()
	}
	var b [1]byte
	_, _ = rand.Read(b[:])
	start := int(b[0]) % n
	for i := 0; i < n; i++ {
		mac := xiaozhiViPoolMACs[(start+i)%n]
		if mac != avoid {
			return mac
		}
	}
	return xiaozhiViPoolMACs[start]
}

func applyXiaozhiViPool(cfg *XiaozhiCfg) {
	applyXiaozhiViPoolDefaults(cfg)
	cfg.DeviceID = pickXiaozhiPoolMAC(cfg.DeviceID)
	cfg.ClientID = genUUIDv4()
	cfg.Token = ""
}

// ensureXiaozhiViPool switches to vi_pool, drawing a pool MAC when the current
// one belongs elsewhere (e.g. a course pool) — vic-cloud no longer restarts to
// rotate it at boot.
func ensureXiaozhiViPool(cfg *XiaozhiCfg) {
	if macInXiaozhiViPool(cfg.DeviceID) && strings.TrimSpace(cfg.ClientID) != "" {
		applyXiaozhiViPoolDefaults(cfg)
		return
	}
	applyXiaozhiViPool(cfg)
}

func markXiaozhiSkipRotate() {
	_ = os.WriteFile(xiaozhiSkipRotatePath, []byte("1"), 0644)
}

// vic-switchboard keeps its gateway datagram socket peered to the dead
// vic-cloud, so the new gateway gets EPERM (no :443) until it restarts too.
func restartVicCloud() {
	_ = exec.Command("/bin/systemctl", "restart", "vic-cloud", "vic-switchboard").Start()
}

// buildOTABody builds the JSON body sent to /xiaozhi/ota/ (ESP32-style).
func buildOTABody(deviceID, clientID string) []byte {
	hostname, _ := os.Hostname()
	payload := map[string]interface{}{
		"version":                2,
		"language":               "en-US",
		"mac_address":            deviceID,
		"uuid":                   clientID,
		"platform":               "linux",
		"arch":                   "arm",
		"hostname":               hostname,
		"minimum_free_heap_size": 0,
		"application": map[string]interface{}{
			"name":    "wire-os",
			"version": "1.0",
		},
		"board": map[string]interface{}{
			"type":   "wire-os",
			"name":   "vector",
			"vendor": "anki",
		},
	}
	b, _ := json.Marshal(payload)
	return b
}

// doOTAPost POSTs system info to {ota_base_url}xiaozhi/ota/ and returns the parsed response.
func doOTAPost(cfg XiaozhiCfg) (map[string]interface{}, error) {
	if cfg.DeviceID == "" || cfg.ClientID == "" {
		return nil, fmt.Errorf("device_id and client_id are required for OTA")
	}
	otaURL := cfg.OTABaseURL + "xiaozhi/ota/"
	body := buildOTABody(cfg.DeviceID, cfg.ClientID)

	req, err := http.NewRequest("POST", otaURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Device-Id", cfg.DeviceID)
	req.Header.Set("Client-Id", cfg.ClientID)
	req.Header.Set("User-Agent", "wire-os/1.0")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("Activation-Version", "1")

	c := &http.Client{Timeout: 20 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OTA HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("OTA JSON: %w", err)
	}
	return result, nil
}

// applyWebsocketFromOTAResult updates endpoint/token from OTA response websocket block.
func applyWebsocketFromOTAResult(cfg *XiaozhiCfg, result map[string]interface{}) {
	ws, ok := result["websocket"].(map[string]interface{})
	if !ok {
		return
	}
	if u, ok := ws["url"].(string); ok && strings.TrimSpace(u) != "" {
		cfg.Endpoint = strings.TrimSpace(u)
	}
	if t, ok := ws["token"].(string); ok && strings.TrimSpace(t) != "" {
		cfg.Token = strings.TrimSpace(t)
	}
}

// Xiaozhi is the wired mod for Xiaozhi AI voice integration.
type Xiaozhi struct {
	vars.Modification
}

func NewXiaozhi() *Xiaozhi { return &Xiaozhi{} }

func (m *Xiaozhi) Name() string        { return "Xiaozhi" }
func (m *Xiaozhi) Description() string { return "Xiaozhi AI voice assistant integration." }
func (m *Xiaozhi) Load() error         { return nil }

func (m *Xiaozhi) HTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case vars.IsEndpoint(r, "get"):
		cfg, err := loadXiaozhiCfg()
		if err != nil {
			cfg = defaultXiaozhiCfg()
		}
		ensureXiaozhiDefaults(&cfg)
		if isXiaozhiViPool(cfg) && strings.TrimSpace(cfg.IdentityMode) == "" {
			cfg.IdentityMode = xiaozhiIdentityViPool
		}
		b, _ := json.Marshal(cfg)
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)

	case vars.IsEndpoint(r, "save"):
		cfg, _ := loadXiaozhiCfg()
		ensureXiaozhiDefaults(&cfg)
		before := cfg
		ident := strings.ToLower(strings.TrimSpace(r.FormValue("identity_mode")))
		if ident == "" {
			ident = strings.ToLower(strings.TrimSpace(cfg.IdentityMode))
		}
		if ident == xiaozhiIdentityViPool || (ident == "" && isXiaozhiViPool(cfg)) {
			ensureXiaozhiViPool(&cfg)
		} else if ident == xiaozhiIdentityCourse {
			applyXiaozhiCourseIdentity(&cfg, false)
		} else {
			cfg.IdentityMode = xiaozhiIdentityCustom
			if v := r.FormValue("ota_base_url"); v != "" {
				cfg.OTABaseURL = normalizeOTABaseURL(v)
			}
			if v := strings.TrimSpace(r.FormValue("device_id")); v == "" {
				cfg.DeviceID = genRandomMAC()
			} else {
				cfg.DeviceID = v
			}
			if v := strings.TrimSpace(r.FormValue("client_id")); v == "" {
				cfg.ClientID = genUUIDv4()
			} else {
				cfg.ClientID = v
			}
		}
		if v := r.FormValue("enabled"); v != "" {
			cfg.Enabled = v == "true" || v == "1" || v == "on"
		}
		cfg.AutoApplyOTAWebsocket = true
		if v := r.FormValue("idle_timeout_sec"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				cfg.IdleTimeoutSec = n
			}
		}
		cfg.TTSMode = "xiaozhi"
		if v := r.FormValue("conversation_mode"); v != "" {
			cfg.ConversationMode = strings.TrimSpace(v)
		}
		cfg.GameGoogleTTSVI = true
		if err := saveXiaozhiCfg(cfg); err != nil {
			vars.HTTPError(w, r, "save failed: "+err.Error())
			return
		}
		applied := false
		if xiaozhiConnectionChanged(before, cfg) {
			applied = applyVicCloudConfig()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"config":  cfg,
			"applied": applied,
		})
		return

	case vars.IsEndpoint(r, "generate_code"):
		cfg, _ := loadXiaozhiCfg()
		ensureXiaozhiDefaults(&cfg)
		if cfg.DeviceID == "" {
			cfg.DeviceID = genRandomMAC()
		}
		if cfg.ClientID == "" {
			cfg.ClientID = genUUIDv4()
		}
		result, err := doOTAPost(cfg)
		if err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		code := ""
		if act, ok := result["activation"].(map[string]interface{}); ok {
			if c, ok := act["code"].(string); ok {
				code = c
			}
		}
		// Always auto-apply websocket url + token from OTA.
		before := cfg
		applyWebsocketFromOTAResult(&cfg, result)
		saveXiaozhiCfg(cfg)
		if xiaozhiConnectionChanged(before, cfg) {
			applyVicCloudConfig()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":      code,
			"device_id": cfg.DeviceID,
			"client_id": cfg.ClientID,
			"endpoint":  cfg.Endpoint,
		})

	case vars.IsEndpoint(r, "refresh"):
		cfg, err := loadXiaozhiCfg()
		if err != nil {
			vars.HTTPError(w, r, "config load: "+err.Error())
			return
		}
		ensureXiaozhiDefaults(&cfg)
		result, err := doOTAPost(cfg)
		if err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		before := cfg
		applyWebsocketFromOTAResult(&cfg, result)
		saveXiaozhiCfg(cfg)
		if xiaozhiConnectionChanged(before, cfg) {
			applyVicCloudConfig()
		}
		vars.HTTPSuccess(w, r)

	// set_enabled: Xiaozhi ↔ Vosk mode / identity preset switch, applied by
	// vic-cloud in place (Vosk is loaded lazily / unloaded there).
	case vars.IsEndpoint(r, "set_enabled"):
		cfg, _ := loadXiaozhiCfg()
		ensureXiaozhiDefaults(&cfg)
		v := strings.TrimSpace(r.FormValue("enabled"))
		if v == "" {
			vars.HTTPError(w, r, "enabled required")
			return
		}
		want := v == "true" || v == "1" || v == "on"
		ident := strings.ToLower(strings.TrimSpace(r.FormValue("identity_mode")))
		if want && ident == "" {
			if isXiaozhiViPool(cfg) || strings.TrimSpace(cfg.IdentityMode) == "" {
				ident = xiaozhiIdentityViPool
			} else {
				ident = xiaozhiIdentityCustom
			}
		}
		prevIdent := strings.ToLower(strings.TrimSpace(cfg.IdentityMode))
		changed := cfg.Enabled != want
		identChanged := want && ident != "" && ident != prevIdent
		cfg.Enabled = want
		cfg.AutoApplyOTAWebsocket = true
		if want {
			if ident == xiaozhiIdentityViPool {
				ensureXiaozhiViPool(&cfg)
			} else if ident == xiaozhiIdentityCourse {
				applyXiaozhiCourseIdentity(&cfg, identChanged)
			} else {
				cfg.IdentityMode = xiaozhiIdentityCustom
				if cfg.DeviceID == "" {
					cfg.DeviceID = genRandomMAC()
				}
				if cfg.ClientID == "" {
					cfg.ClientID = genUUIDv4()
				}
				if result, err := doOTAPost(cfg); err == nil {
					applyWebsocketFromOTAResult(&cfg, result)
				}
			}
		}
		if err := saveXiaozhiCfg(cfg); err != nil {
			vars.HTTPError(w, r, "save failed: "+err.Error())
			return
		}
		applied, restarted := false, false
		if changed || identChanged {
			applied = applyVicCloudConfig()
			restarted = !applied
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "success",
			"config":    cfg,
			"applied":   applied,
			"restarted": restarted,
		})
		return

	case vars.IsEndpoint(r, "renew_pool"):
		cfg, _ := loadXiaozhiCfg()
		ensureXiaozhiDefaults(&cfg)
		applyXiaozhiViPool(&cfg)
		cfg.Enabled = true
		if result, err := doOTAPost(cfg); err == nil {
			applyWebsocketFromOTAResult(&cfg, result)
		}
		if err := saveXiaozhiCfg(cfg); err != nil {
			vars.HTTPError(w, r, "save failed: "+err.Error())
			return
		}
		applied := applyVicCloudConfig()
		if !applied {
			// Restart fallback: keep the MAC just picked instead of rotating at boot.
			markXiaozhiSkipRotate()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "success",
			"config":    cfg,
			"applied":   applied,
			"restarted": !applied,
		})
		return

	case vars.IsEndpoint(r, "courses"):
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"courses": xzCourses})
		return

	// set_course: English-teaching lesson (course, voice, book, unit, student
	// name). Course/voice changes draw a new pool MAC and re-apply vic-cloud
	// (robot greets); book/unit/name are read live by the MCP tools.
	case vars.IsEndpoint(r, "set_course"):
		cfg, _ := loadXiaozhiCfg()
		ensureXiaozhiDefaults(&cfg)
		var u xzCourseUpdate
		intArg := func(name string) *int {
			v := strings.TrimSpace(r.FormValue(name))
			if v == "" {
				return nil
			}
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil
			}
			return &n
		}
		u.Idx, u.Voice, u.Book, u.Unit = intArg("course_idx"), intArg("voice"), intArg("book"), intArg("unit")
		if _, ok := r.Form["student_name"]; ok {
			name := r.FormValue("student_name")
			u.StudentName = &name
		}
		before := cfg
		identityChanged := applyXiaozhiCourseUpdate(&cfg, u)
		if !identityChanged && xiaozhiConnectionChanged(before, cfg) {
			identityChanged = true
		}
		// vic-cloud runs the OTA check for the new MAC when it applies.
		if err := saveXiaozhiCfg(cfg); err != nil {
			vars.HTTPError(w, r, "save failed: "+err.Error())
			return
		}
		applied := false
		if identityChanged {
			applied = applyVicCloudConfig()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":           "success",
			"config":           cfg,
			"identity_changed": identityChanged,
			"applied":          applied,
		})
		return

	case vars.IsEndpoint(r, "set_game_google_tts_vi"):
		// Kept for old clients — Google game TTS is always on.
		cfg, _ := loadXiaozhiCfg()
		ensureXiaozhiDefaults(&cfg)
		cfg.GameGoogleTTSVI = true
		if lang := strings.TrimSpace(r.FormValue("lang")); lang != "" {
			cfg.GameGoogleTTSLang = normalizeGoogleTTSLang(lang)
		}
		if err := saveXiaozhiCfg(cfg); err != nil {
			vars.HTTPError(w, r, "save failed: "+err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"config": cfg,
		})
		return

	case vars.IsEndpoint(r, "set_listen_mode"):
		mode := strings.ToLower(strings.TrimSpace(r.FormValue("mode")))
		if mode != xiaozhiListenFixed && mode != xiaozhiListenVAD {
			vars.HTTPError(w, r, "mode must be fixed or vad")
			return
		}
		cfg, _ := loadXiaozhiCfg()
		ensureXiaozhiDefaults(&cfg)
		cfg.ListenMode = mode
		if err := saveXiaozhiCfg(cfg); err != nil {
			vars.HTTPError(w, r, "save failed: "+err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"config": cfg,
		})
		return

	// set_face_greet: vic-cloud reads face_greet from disk when a person
	// arrives, so no reload (which would itself greet) is needed.
	case vars.IsEndpoint(r, "set_face_greet"):
		v := strings.TrimSpace(r.FormValue("enabled"))
		if v == "" {
			vars.HTTPError(w, r, "enabled required")
			return
		}
		cfg, _ := loadXiaozhiCfg()
		ensureXiaozhiDefaults(&cfg)
		cfg.FaceGreet = v == "true" || v == "1" || v == "on"
		if err := saveXiaozhiCfg(cfg); err != nil {
			vars.HTTPError(w, r, "save failed: "+err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"config": cfg,
		})
		return

	case vars.IsEndpoint(r, "set_music"):
		source := strings.ToLower(strings.TrimSpace(r.FormValue("source")))
		if source != xiaozhiMusicServer && source != xiaozhiMusicYouTube {
			vars.HTTPError(w, r, "source must be server or youtube")
			return
		}
		cfg, _ := loadXiaozhiCfg()
		ensureXiaozhiDefaults(&cfg)
		cfg.MusicSource = source
		cfg.MusicServerURL = normalizeMusicServerURL(r.FormValue("url"))
		if err := saveXiaozhiCfg(cfg); err != nil {
			vars.HTTPError(w, r, "save failed: "+err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"config": cfg,
		})
		return

	case vars.IsEndpoint(r, "test_music_server"):
		base := strings.TrimSpace(r.FormValue("url"))
		if base == "" {
			cfg, _ := loadXiaozhiCfg()
			base = cfg.MusicServerURL
		}
		query := strings.TrimSpace(r.FormValue("q"))
		if query == "" {
			query = "Lac Troi Son Tung"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testMusicServer(normalizeMusicServerURL(base), query))
		return

	case vars.IsEndpoint(r, "set_game_google_tts_lang"):
		cfg, _ := loadXiaozhiCfg()
		ensureXiaozhiDefaults(&cfg)
		lang := strings.TrimSpace(r.FormValue("lang"))
		if lang == "" {
			vars.HTTPError(w, r, "lang required")
			return
		}
		cfg.GameGoogleTTSVI = true
		cfg.GameGoogleTTSLang = normalizeGoogleTTSLang(lang)
		if err := saveXiaozhiCfg(cfg); err != nil {
			vars.HTTPError(w, r, "save failed: "+err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"config": cfg,
		})
		return

	case vars.IsEndpoint(r, "unpair"):
		// Deprecated: unbind on xiaozhi.me. Kept for old clients; clears local token only.
		cfg, err := loadXiaozhiCfg()
		if err != nil {
			vars.HTTPError(w, r, "config load: "+err.Error())
			return
		}
		cfg.Token = ""
		saveXiaozhiCfg(cfg)
		vars.HTTPSuccess(w, r)

	default:
		vars.HTTPError(w, r, "unknown endpoint")
	}
}
