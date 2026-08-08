package mods

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/digital-dream-labs/vector-go-sdk/pkg/vectorpb"
	"github.com/os-vector/wired/vars"
)

// Battery exposes live robot battery state (WirePod-style volts → %).
type Battery struct {
	vars.Modification
}

func NewBattery() *Battery {
	return &Battery{}
}

func (m *Battery) Name() string { return "Battery" }

func (m *Battery) Description() string {
	return "Live battery percentage from BatteryState (volts curve)"
}

func (m *Battery) Load() error { return nil }

func (m *Battery) HTTP(w http.ResponseWriter, r *http.Request) {
	if vars.IsEndpoint(r, "get") || vars.IsEndpoint(r, "summary") {
		info, err := readBatteryInfo()
		if err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		if vars.IsEndpoint(r, "summary") {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte(info.Summary))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(info)
		return
	}
	vars.HTTPError(w, r, "404 not found")
}

type batteryInfo struct {
	Percent             int     `json:"percent"`
	Volts               float32 `json:"volts"`
	Level               int32   `json:"level"`
	Charging            bool    `json:"charging"`
	OnCharger           bool    `json:"on_charger"`
	ChargeState         string  `json:"charge_state"` // charging | full_on_charger | on_charger | discharging
	SuggestedChargerSec float32 `json:"suggested_charger_sec,omitempty"`
	Summary             string  `json:"summary"`
	SummaryEN           string  `json:"summary_en"`
}

func readBatteryInfo() (*batteryInfo, error) {
	resp, err := fetchBatteryState()
	if err != nil {
		// vic-cloud may have restarted; drop stale conn and retry once.
		vars.InvalidateVec()
		resp, err = fetchBatteryState()
		if err != nil {
			return nil, err
		}
	}
	return batteryInfoFromResp(resp), nil
}

func fetchBatteryState() (*vectorpb.BatteryStateResponse, error) {
	v, err := vars.GetVec()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return v.Conn.BatteryState(ctx, &vectorpb.BatteryStateRequest{})
}

func batteryInfoFromResp(resp *vectorpb.BatteryStateResponse) *batteryInfo {
	volts := resp.GetBatteryVolts()
	level := int32(resp.GetBatteryLevel())
	onCharger := resp.GetIsOnChargerPlatform()
	charging := resp.GetIsCharging()
	suggest := resp.GetSuggestedChargerSec()
	percent := batteryPercentage(float64(volts))

	// Mirror WirePod battery.js: LOW + off dock → treat as critically low.
	if level == 1 && !onCharger {
		if percent > 15 {
			percent = 15
		}
	}
	if onCharger && !charging && volts >= 4.05 {
		percent = 100
	}

	info := &batteryInfo{
		Percent:             percent,
		Volts:               volts,
		Level:               level,
		Charging:            charging,
		OnCharger:           onCharger,
		SuggestedChargerSec: suggest,
	}
	info.ChargeState = chargeState(info)
	info.Summary, info.SummaryEN = batterySummaries(info)
	return info
}

func chargeState(info *batteryInfo) string {
	switch {
	case info.OnCharger && info.Percent >= 99 && !info.Charging:
		return "full_on_charger"
	case info.Charging:
		return "charging"
	case info.OnCharger:
		return "on_charger"
	default:
		return "discharging"
	}
}

// batteryPercentage ports chipper/webroot/js/battery.js getBatteryPercentage.
func batteryPercentage(voltage float64) int {
	const (
		maxVoltage = 4.1
		midVoltage = 3.85
		minVoltage = 3.5
	)
	var percentage float64
	switch {
	case voltage >= maxVoltage:
		percentage = 100
	case voltage >= midVoltage:
		scaled := (voltage - midVoltage) / (maxVoltage - midVoltage)
		percentage = 80 + 20*math.Log10(1+scaled*9)
	case voltage >= minVoltage:
		scaled := (voltage - minVoltage) / (midVoltage - minVoltage)
		percentage = 80 * math.Log10(1+scaled*9)
	case voltage == 0:
		// Off-charger boot sometimes omits volts — assume mid charge.
		percentage = 70
	default:
		percentage = 0
	}
	p := int(math.Round(percentage))
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	return p
}

func batterySummaries(info *batteryInfo) (vi, en string) {
	if info == nil {
		return "Không đọc được pin.", "Unable to read battery."
	}
	p := strconv.Itoa(info.Percent)
	switch info.ChargeState {
	case "full_on_charger":
		return "Pin đã đầy 100%. Đang trên đế sạc, không còn nạp.",
			"Battery is full at 100%. On the charger, not actively charging."
	case "charging":
		return "Pin còn khoảng " + p + "%. Đang sạc.",
			"Battery about " + p + "%. Currently charging."
	case "on_charger":
		return "Pin còn khoảng " + p + "%. Đang trên đế sạc.",
			"Battery about " + p + "%. On the charger dock."
	case "discharging":
		if info.Percent <= 15 {
			return "Pin còn khoảng " + p + "%. Không đang sạc. Pin yếu, nên về đế sạc.",
				"Battery about " + p + "%. Not charging. Low battery — go home to charge."
		}
		return "Pin còn khoảng " + p + "%. Không đang sạc.",
			"Battery about " + p + "%. Not charging."
	default:
		return "Pin còn khoảng " + p + "%.",
			"Battery about " + p + "%."
	}
}
