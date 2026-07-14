package mods

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/os-vector/wired/vars"
)

type Alexa struct {
	vars.Modification
}

func NewAlexa() *Alexa {
	return &Alexa{}
}

func (modu *Alexa) Name() string {
	return "Alexa"
}

func (modu *Alexa) Description() string {
	return "Amazon Alexa opt-in and button wake word settings."
}

func (modu *Alexa) Load() error {
	return nil
}

type alexaAuthJSON struct {
	Status struct {
		Code int `json:"code"`
	} `json:"status"`
	AuthState int    `json:"auth_state"`
	Extra     string `json:"extra"`
}

type alexaStatusPayload struct {
	AuthState      int    `json:"auth_state"`
	AuthLabel      string `json:"auth_label"`
	Extra          string `json:"extra"`
	ButtonWakeword int    `json:"button_wakeword"`
	FeatureEnabled bool   `json:"feature_enabled"`
}

func (m *Alexa) HTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/mods/Alexa/status":
		m.writeStatus(w, r)
	case "/api/mods/Alexa/optIn":
		enable := strings.EqualFold(r.URL.Query().Get("enable"), "true") || r.URL.Query().Get("enable") == "1"
		m.doOptIn(w, r, enable)
	case "/api/mods/Alexa/setButtonWake":
		mode := strings.TrimSpace(r.URL.Query().Get("mode"))
		if mode == "" {
			vars.HTTPError(w, r, "mode required (0=Hey Vector, 1=Alexa)")
			return
		}
		if err := setButtonWakeword(mode); err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		vars.HTTPSuccess(w, r)
	default:
		vars.HTTPError(w, r, "404 not found")
	}
}

func (m *Alexa) writeStatus(w http.ResponseWriter, r *http.Request) {
	auth, err := getAlexaAuthState()
	if err != nil {
		vars.HTTPError(w, r, err.Error())
		return
	}
	btn := 0
	if doc, err := pullRobotSettings(); err == nil {
		btn = doc.ButtonWakeword
	}
	payload := alexaStatusPayload{
		AuthState:      auth.AuthState,
		AuthLabel:      alexaAuthLabel(auth.AuthState),
		Extra:          auth.Extra,
		ButtonWakeword: btn,
		FeatureEnabled: true,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func (m *Alexa) doOptIn(w http.ResponseWriter, r *http.Request, enable bool) {
	if err := postAlexaOptIn(enable); err != nil {
		vars.HTTPError(w, r, err.Error())
		return
	}
	vars.HTTPSuccess(w, r)
}

func alexaAuthLabel(state int) string {
	switch state {
	case 0:
		return "Lỗi / không hợp lệ (Invalid)"
	case 1:
		return "Chưa liên kết (Not linked)"
	case 2:
		return "Đang kết nối Amazon... (Requesting auth)"
	case 3:
		return "Chờ nhập mã — xem mặt robot (Waiting for code)"
	case 4:
		return "Đã liên kết (Authorized)"
	default:
		return fmt.Sprintf("Trạng thái %d", state)
	}
}

func cloudPostJSON(path string, body []byte) ([]byte, error) {
	guid, err := vars.GetGUID()
	if err != nil {
		return nil, err
	}
	url := "https://localhost:443" + path
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+guid)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Transport: transCfg}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("cloud HTTP %d: %s", resp.StatusCode, string(out))
	}
	return out, nil
}

func getAlexaAuthState() (alexaAuthJSON, error) {
	var out alexaAuthJSON
	raw, err := cloudPostJSON("/v1/alexa_auth_state", []byte("{}"))
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	return out, nil
}

func postAlexaOptIn(optIn bool) error {
	body := []byte(`{"opt_in":false}`)
	if optIn {
		body = []byte(`{"opt_in":true}`)
	}
	_, err := cloudPostJSON("/v1/alexa_opt_in", body)
	return err
}

func setButtonWakeword(mode string) error {
	v, err := strconv.Atoi(mode)
	if err != nil || (v != 0 && v != 1) {
		return fmt.Errorf("mode must be 0 or 1")
	}
	return setSettingSDKRaw(fmt.Sprintf(`{"button_wakeword":%d}`, v))
}

func setSettingSDKRaw(settingsJSONObject string) error {
	url := "https://localhost:443/v1/update_settings"
	body := []byte(`{"update_settings":true,"settings":` + settingsJSONObject + `}`)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	guid, err := vars.GetGUID()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+guid)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Transport: transCfg}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
