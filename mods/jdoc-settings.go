package mods

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/digital-dream-labs/vector-go-sdk/pkg/vectorpb"
	"github.com/os-vector/wired/vars"
)

var ctx context.Context

type JdocSettings struct {
	vars.Modification
}

func NewJdocSettings() *JdocSettings {
	return &JdocSettings{}
}

func (modu *JdocSettings) Name() string {
	return "JdocSettings"
}

func (modu *JdocSettings) Description() string {
	return "A couple settings in case you are using the WireOS servers."
}

func (modu *JdocSettings) Load() error {
	ctx = context.Background()
	return nil
}

func (m *JdocSettings) HTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/mods/JdocSettings/") {
		return
	}
	switch strings.TrimPrefix(r.URL.Path, "/api/mods/JdocSettings/") {
	case "setLocation":
		location := r.FormValue("location")
		err := setLocation(location)
		if err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
	case "setTimezone":
		timezone := r.FormValue("timezone")
		err := setTimezone(timezone)
		if err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
	case "setFahrenheit":
		temp := r.FormValue("temp")
		if temp == "" {
			temp = r.FormValue("t")
		}
		var gib bool
		if temp == "f" {
			gib = true
		} else {
			gib = false
		}
		err := setFahrenheit(gib)
		if err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
	case "getVolume":
		vol, err := getMasterVolume()
		if err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		fmt.Fprintf(w, "%d", vol)
		return
	case "setVolume":
		level := r.FormValue("level")
		if level == "" {
			vars.HTTPError(w, r, "empty level")
			return
		}
		if err := setMasterVolume(level); err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
	case "getEyeColor":
		ec, err := getEyeColorSetting()
		if err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		b, _ := json.Marshal(ec)
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
		return
	case "setEyeColor":
		preset := r.FormValue("preset")
		if preset == "" {
			vars.HTTPError(w, r, "empty preset")
			return
		}
		if err := setPresetEyeColor(preset); err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
	case "setCustomEyeColor":
		hue := r.FormValue("hue")
		sat := r.FormValue("saturation")
		if hue == "" || sat == "" {
			vars.HTTPError(w, r, "hue and saturation required (0..1)")
			return
		}
		if err := setCustomEyeColor(hue, sat); err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
	case "getLocation":
		location, err := getLocation()
		if err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		w.Write([]byte(location))
		return
	case "getTimezone":
		timezone, err := getTimezone()
		if err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		w.Write([]byte(timezone))
		return
	case "getFahrenheit":
		temp, err := getFahrenheit()
		if err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		var ret string = "c"
		if temp {
			ret = "f"
		}
		w.Write([]byte(ret))
		return
	default:
		vars.HTTPError(w, r, "404 not found")
	}
	vars.HTTPSuccess(w, r)
}

func setLocation(location string) error {
	if location == "" {
		return errors.New("empty location")
	}
	return setSettingSDKstring("default_location", location)
}

func setTimezone(timezone string) error {
	if timezone == "" {
		return errors.New("empty time zone")
	}
	return setSettingSDKstring("time_zone", timezone)
}

func setFahrenheit(isF bool) error {
	return setSettingSDKintbool("temp_is_fahrenheit", fmt.Sprint(isF))
}

type eyeColorSettingResponse struct {
	IsCustom         bool    `json:"iscustom"`
	Preset           int     `json:"preset"`
	CustomHue        float32 `json:"hue"`
	CustomSaturation float32 `json:"saturation"`
}

func getMasterVolume() (int, error) {
	doc, err := pullRobotSettings()
	if err != nil {
		return 0, err
	}
	return doc.MasterVolume, nil
}

func setMasterVolume(level string) error {
	return setSettingSDKintbool("master_volume", level)
}

func getEyeColorSetting() (eyeColorSettingResponse, error) {
	var resp eyeColorSettingResponse
	doc, err := pullRobotSettings()
	if err != nil {
		return resp, err
	}
	resp.Preset = doc.EyeColor
	if doc.CustomEyeColor.Enabled {
		resp.IsCustom = true
		resp.CustomHue = float32(doc.CustomEyeColor.Hue)
		resp.CustomSaturation = float32(doc.CustomEyeColor.Saturation)
	}
	return resp, nil
}

func setPresetEyeColor(preset string) error {
	url := "https://localhost:443/v1/update_settings"
	body := []byte(fmt.Sprintf(
		`{"update_settings":true,"settings":{"custom_eye_color":{"enabled":false,"hue":0,"saturation":0},"eye_color":%s}}`,
		preset,
	))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
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

func setCustomEyeColor(hue, sat string) error {
	url := "https://localhost:443/v1/update_settings"
	body := []byte(fmt.Sprintf(
		`{"update_settings":true,"settings":{"custom_eye_color":{"enabled":true,"hue":%s,"saturation":%s}}}`,
		hue, sat,
	))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
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

func pullRobotSettings() (robotSettingsJson, error) {
	v, err := vars.GetVec()
	if err != nil {
		return robotSettingsJson{}, err
	}
	r, err := v.Conn.PullJdocs(ctx,
		&vectorpb.PullJdocsRequest{
			JdocTypes: []vectorpb.JdocType{
				vectorpb.JdocType_ROBOT_SETTINGS,
			},
		},
	)
	if err != nil {
		return robotSettingsJson{}, err
	}
	if len(r.NamedJdocs) == 0 {
		return robotSettingsJson{}, errors.New("empty jdocs")
	}
	var decodedDoc robotSettingsJson
	if err := json.Unmarshal([]byte(r.NamedJdocs[0].Doc.JsonDoc), &decodedDoc); err != nil {
		return robotSettingsJson{}, err
	}
	return decodedDoc, nil
}

func getLocation() (string, error) {
	v, err := vars.GetVec()
	if err != nil {
		return "", err
	}
	r, err := v.Conn.PullJdocs(ctx,
		&vectorpb.PullJdocsRequest{
			JdocTypes: []vectorpb.JdocType{
				vectorpb.JdocType_ROBOT_SETTINGS,
			},
		},
	)
	if err != nil {
		return "", err
	}
	doc := r.NamedJdocs[0].Doc.JsonDoc
	var decodedDoc robotSettingsJson
	json.Unmarshal([]byte(doc), &decodedDoc)
	return decodedDoc.DefaultLocation, nil
}

func getTimezone() (string, error) {
	v, err := vars.GetVec()
	if err != nil {
		return "", err
	}
	r, err := v.Conn.PullJdocs(ctx,
		&vectorpb.PullJdocsRequest{
			JdocTypes: []vectorpb.JdocType{
				vectorpb.JdocType_ROBOT_SETTINGS,
			},
		},
	)
	if err != nil {
		return "", err
	}
	doc := r.NamedJdocs[0].Doc.JsonDoc
	var decodedDoc robotSettingsJson
	json.Unmarshal([]byte(doc), &decodedDoc)
	return decodedDoc.TimeZone, nil
}

func getFahrenheit() (bool, error) {
	v, err := vars.GetVec()
	if err != nil {
		return false, err
	}
	r, err := v.Conn.PullJdocs(ctx,
		&vectorpb.PullJdocsRequest{
			JdocTypes: []vectorpb.JdocType{
				vectorpb.JdocType_ROBOT_SETTINGS,
			},
		},
	)
	if err != nil {
		return false, err
	}
	doc := r.NamedJdocs[0].Doc.JsonDoc
	var decodedDoc robotSettingsJson
	json.Unmarshal([]byte(doc), &decodedDoc)
	return decodedDoc.TempIsFahrenheit, nil
}

var transCfg = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // ignore SSL warnings
}

func setSettingSDKstring(setting string, value string) error {
	url := "https://localhost:443/v1/update_settings"
	var updateJSON = []byte(`{"update_settings": true, "settings": {"` + setting + `": "` + value + `" } }`)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(updateJSON))
	guid, err := vars.GetGUID()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+guid)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Transport: transCfg}
	_, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	return nil
}

func setSettingSDKintbool(setting string, value string) error {
	url := "https://localhost:443/v1/update_settings"
	var updateJSON = []byte(`{"update_settings": true, "settings": {"` + setting + `": ` + value + ` } }`)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(updateJSON))
	guid, err := vars.GetGUID()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+guid)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Transport: transCfg}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	return nil
}

type robotSettingsJson struct {
	ButtonWakeword int  `json:"button_wakeword"`
	Clock24Hour    bool `json:"clock_24_hour"`
	CustomEyeColor struct {
		Enabled    bool    `json:"enabled"`
		Hue        float64 `json:"hue"`
		Saturation float64 `json:"saturation"`
	} `json:"custom_eye_color"`
	DefaultLocation  string `json:"default_location"`
	DistIsMetric     bool   `json:"dist_is_metric"`
	EyeColor         int    `json:"eye_color"`
	Locale           string `json:"locale"`
	MasterVolume     int    `json:"master_volume"`
	TempIsFahrenheit bool   `json:"temp_is_fahrenheit"`
	TimeZone         string `json:"time_zone"`
}
