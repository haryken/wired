package mods

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
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
}

func defaultXiaozhiCfg() XiaozhiCfg {
	return XiaozhiCfg{
		Enabled:               false,
		OTABaseURL:            "https://api.tenclass.net/",
		Endpoint:              "wss://api.tenclass.net/xiaozhi/v1/",
		ProtocolVersion:       1,
		AutoApplyOTAWebsocket: true,
		ConversationMode:      "continuous",
		IdleTimeoutSec:        20,
		TTSMode:               "xiaozhi",
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

// firstNonLoopbackMAC returns the first non-loopback hardware MAC address.
func firstNonLoopbackMAC() string {
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if len(iface.HardwareAddr) == 0 {
			continue
		}
		return iface.HardwareAddr.String()
	}
	return ""
}

// genUUIDv4 generates a random UUID v4 without external dependencies.
func genUUIDv4() string {
	var b [16]byte
	rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
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
		b, _ := json.Marshal(cfg)
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)

	case vars.IsEndpoint(r, "save"):
		cfg, _ := loadXiaozhiCfg()
		ensureXiaozhiDefaults(&cfg)
		if v := r.FormValue("ota_base_url"); v != "" {
			cfg.OTABaseURL = normalizeOTABaseURL(v)
		}
		// Endpoint/token are never set from the form — only from OTA responses.
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
		if v := r.FormValue("device_id"); v != "" {
			cfg.DeviceID = strings.TrimSpace(v)
		}
		if v := r.FormValue("client_id"); v != "" {
			cfg.ClientID = strings.TrimSpace(v)
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
			cfg.DeviceID = firstNonLoopbackMAC()
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
		changed := cfg.Enabled != want
		cfg.Enabled = want
		cfg.AutoApplyOTAWebsocket = true
		if want {
			if cfg.DeviceID == "" {
				cfg.DeviceID = firstNonLoopbackMAC()
			}
			if cfg.ClientID == "" {
				cfg.ClientID = genUUIDv4()
			}
			// Best-effort: pull WSS endpoint/token on first enable / mode switch on.
			if result, err := doOTAPost(cfg); err == nil {
				applyWebsocketFromOTAResult(&cfg, result)
			}
		}
		if err := saveXiaozhiCfg(cfg); err != nil {
			vars.HTTPError(w, r, "save failed: "+err.Error())
			return
		}
		restarted := false
		if changed {
			_ = exec.Command("/bin/systemctl", "restart", "vic-cloud").Start()
			restarted = true
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "success",
			"config":    cfg,
			"restarted": restarted,
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
