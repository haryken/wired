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
	// IdentityMode: "vi_pool" = shared Vietnamese preset; "custom" = xiaozhi.me pair.
	IdentityMode string `json:"identity_mode,omitempty"`
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
	if err := os.MkdirAll(filepath.Dir(xiaozhiConfigPath), 0777); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(xiaozhiConfigPath, data, 0644)
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
	xiaozhiIdentityViPool  = "vi_pool"
	xiaozhiIdentityCustom  = "custom"
	xiaozhiTenclassOTA     = "https://api.tenclass.net/"
	xiaozhiTenclassWSS     = "wss://api.tenclass.net/xiaozhi/v1/"
	xiaozhiSkipRotatePath  = "/run/xiaozhi-skip-rotate"
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

func markXiaozhiSkipRotate() {
	_ = os.WriteFile(xiaozhiSkipRotatePath, []byte("1"), 0644)
}

func restartVicCloud() {
	_ = exec.Command("/bin/systemctl", "restart", "vic-cloud").Start()
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
		ident := strings.ToLower(strings.TrimSpace(r.FormValue("identity_mode")))
		if ident == "" {
			ident = strings.ToLower(strings.TrimSpace(cfg.IdentityMode))
		}
		if ident == xiaozhiIdentityViPool || (ident == "" && isXiaozhiViPool(cfg)) {
			applyXiaozhiViPoolDefaults(&cfg)
			if strings.TrimSpace(cfg.DeviceID) == "" {
				cfg.DeviceID = pickXiaozhiPoolMAC("")
			}
			if strings.TrimSpace(cfg.ClientID) == "" {
				cfg.ClientID = genUUIDv4()
			}
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
		if v := r.FormValue("game_google_tts_vi"); v != "" {
			cfg.GameGoogleTTSVI = v == "true" || v == "1" || v == "on"
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
		applyWebsocketFromOTAResult(&cfg, result)
		saveXiaozhiCfg(cfg)
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
		applyWebsocketFromOTAResult(&cfg, result)
		saveXiaozhiCfg(cfg)
		vars.HTTPSuccess(w, r)

	// set_enabled: Xiaozhi ↔ Vosk mode switch. Applies immediately and restarts
	// vic-cloud so Vosk is loaded/unloaded correctly (boot-time InitVosk path).
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
				// Defaults only — boot rotates MAC/UUID (unless skip file).
				applyXiaozhiViPoolDefaults(&cfg)
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
		restarted := false
		if changed || identChanged {
			restartVicCloud()
			restarted = true
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "success",
			"config":    cfg,
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
		markXiaozhiSkipRotate()
		restartVicCloud()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "success",
			"config":    cfg,
			"restarted": true,
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
