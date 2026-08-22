package mods

// ConnMan Agent + Service.Connect — same credential path as stock BLE
// (Wifi::ConnectWiFiBySsid in switchboard anki-wifi).

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

const hsConnmanAgentPath = "/wireos/connman_agent"

type hsConnmanAgent struct {
	passphrase string
	ssid       string
	mu         sync.Mutex
	invalidKey bool
	errText    string
}

func (a *hsConnmanAgent) Release() *dbus.Error { return nil }

func (a *hsConnmanAgent) Cancel() *dbus.Error { return nil }

func (a *hsConnmanAgent) ReportError(service dbus.ObjectPath, errMsg string) *dbus.Error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.errText = errMsg
	if errMsg == "invalid-key" {
		a.invalidKey = true
	}
	wifiLog("hotspot agent ReportError svc=%s err=%s", service, errMsg)
	return nil
}

func (a *hsConnmanAgent) RequestInput(service dbus.ObjectPath, fields map[string]dbus.Variant) (map[string]dbus.Variant, *dbus.Error) {
	out := map[string]dbus.Variant{}
	a.mu.Lock()
	pass := a.passphrase
	name := a.ssid
	a.mu.Unlock()
	if _, ok := fields["Passphrase"]; ok && pass != "" {
		out["Passphrase"] = dbus.MakeVariant(pass)
	}
	if _, ok := fields["Name"]; ok && name != "" {
		out["Name"] = dbus.MakeVariant(name)
	}
	wifiLog("hotspot agent RequestInput svc=%s fields=%v", service, keysOfVariants(fields))
	return out, nil
}

func keysOfVariants(m map[string]dbus.Variant) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

// hsConnectLikeBLE mirrors switchboard ConnectWiFiBySsid:
// find wifi service by Name on wlan0 → RegisterAgent(pass) → Service.Connect.
func hsConnectLikeBLE(ssid, pass string, hidden bool) error {
	ssid = strings.TrimSpace(ssid)
	if ssid == "" {
		return fmt.Errorf("empty ssid")
	}
	bus, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("system bus: %w", err)
	}

	svcPath, err := hsFindWifiServicePath(bus, ssid, hidden)
	if err != nil {
		return err
	}
	wifiLog("hotspot BLE-style connect path=%s ssid=%q", svcPath, ssid)

	agent := &hsConnmanAgent{passphrase: pass, ssid: ssid}
	if err := bus.Export(agent, hsConnmanAgentPath, "net.connman.Agent"); err != nil {
		return fmt.Errorf("export agent: %w", err)
	}
	defer func() { _ = bus.Export(nil, hsConnmanAgentPath, "net.connman.Agent") }()

	manager := bus.Object("net.connman", "/")
	if call := manager.Call("net.connman.Manager.RegisterAgent", 0, dbus.ObjectPath(hsConnmanAgentPath)); call.Err != nil {
		return fmt.Errorf("RegisterAgent: %w", call.Err)
	}
	defer manager.Call("net.connman.Manager.UnregisterAgent", 0, dbus.ObjectPath(hsConnmanAgentPath))

	svc := bus.Object("net.connman", svcPath)
	done := make(chan error, 1)
	go func() {
		// Long timeout: Agent RequestInput happens during Connect.
		c := svc.Go("net.connman.Service.Connect", 0, nil)
		select {
		case call := <-c.Done:
			if call.Err != nil {
				done <- call.Err
			} else {
				done <- nil
			}
		case <-time.After(45 * time.Second):
			done <- fmt.Errorf("Connect timeout")
		}
	}()

	err = <-done
	agent.mu.Lock()
	invalid := agent.invalidKey
	agentErr := agent.errText
	agent.mu.Unlock()
	if invalid {
		return fmt.Errorf("invalid-key")
	}
	if err != nil {
		if agentErr != "" {
			return fmt.Errorf("connect: %v (%s)", err, agentErr)
		}
		return fmt.Errorf("connect: %w", err)
	}
	return nil
}

func hsFindWifiServicePath(bus *dbus.Conn, ssid string, hidden bool) (dbus.ObjectPath, error) {
	manager := bus.Object("net.connman", "/")
	var services [][]interface{}
	if err := manager.Call("net.connman.Manager.GetServices", 0).Store(&services); err != nil {
		return "", fmt.Errorf("GetServices: %w", err)
	}
	var hiddenPath dbus.ObjectPath
	for _, pair := range services {
		if len(pair) < 2 {
			continue
		}
		path, ok := pair[0].(dbus.ObjectPath)
		if !ok {
			if s, ok2 := pair[0].(string); ok2 {
				path = dbus.ObjectPath(s)
			} else {
				continue
			}
		}
		props, ok := pair[1].(map[string]dbus.Variant)
		if !ok {
			continue
		}
		typ, _ := props["Type"].Value().(string)
		if typ != "wifi" {
			continue
		}
		ifaceOK := false
		if eth, ok := props["Ethernet"].Value().(map[string]dbus.Variant); ok {
			if iface, ok := eth["Interface"].Value().(string); ok && iface == "wlan0" {
				ifaceOK = true
			}
		}
		// Some ConnMan builds nest Ethernet differently; fall back to path check.
		if !ifaceOK && strings.Contains(string(path), "wifi_") {
			ifaceOK = true
		}
		if !ifaceOK {
			continue
		}
		name, hasName := "", false
		if v, ok := props["Name"]; ok {
			if s, ok2 := v.Value().(string); ok2 {
				name, hasName = s, true
			}
		}
		if hasName && name == ssid {
			if st, _ := props["State"].Value().(string); st == "online" || st == "ready" {
				return path, nil
			}
			return path, nil
		}
		if hidden && !hasName && hiddenPath == "" {
			hiddenPath = path
		}
	}
	if hidden && hiddenPath != "" {
		return hiddenPath, nil
	}
	return "", fmt.Errorf("không thấy service WiFi %q — quét lại", ssid)
}

// listConnmanWifiNets uses Manager.GetServices Name (same D-Bus path as
// hsFindWifiServicePath). connmanctl text columns split "Huynh 2.4" → "2.4".
func listConnmanWifiNets() []wifiNet {
	bus, err := dbus.SystemBus()
	if err != nil {
		return nil
	}
	manager := bus.Object("net.connman", "/")
	var services [][]interface{}
	if err := manager.Call("net.connman.Manager.GetServices", 0).Store(&services); err != nil {
		return nil
	}
	var nets []wifiNet
	seen := map[string]wifiNet{}
	for _, pair := range services {
		if len(pair) < 2 {
			continue
		}
		path, ok := pair[0].(dbus.ObjectPath)
		if !ok {
			if s, ok2 := pair[0].(string); ok2 {
				path = dbus.ObjectPath(s)
			} else {
				continue
			}
		}
		props, ok := pair[1].(map[string]dbus.Variant)
		if !ok {
			continue
		}
		typ, _ := props["Type"].Value().(string)
		if typ != "wifi" {
			continue
		}
		if freq := dbusUint(props["Frequency"]); freq >= 5000 {
			continue
		}
		name := ""
		if v, ok := props["Name"]; ok {
			name, _ = v.Value().(string)
		}
		name = strings.TrimSpace(name)
		if decoded := ssidFromConnmanPath(string(path)); decoded != "" {
			if name == "" || len(decoded) > len(name) || strings.HasSuffix(decoded, " "+name) {
				name = decoded
			}
		}
		if name == "" {
			continue
		}
		n := wifiNet{
			SSID:   name,
			Signal: int(dbusUint(props["Strength"])),
			Secure: connmanPropsSecure(props),
		}
		if n.Signal <= 0 {
			n.Signal = 50
		}
		if old, ok := seen[name]; !ok || n.Signal > old.Signal {
			seen[name] = n
		}
	}
	for _, n := range seen {
		nets = append(nets, n)
	}
	return nets
}

func connmanPropsSecure(props map[string]dbus.Variant) bool {
	v, ok := props["Security"]
	if !ok {
		return false
	}
	switch sec := v.Value().(type) {
	case []string:
		for _, s := range sec {
			if s == "psk" || s == "ieee8021x" || s == "wep" || s == "wpa" || s == "rsn" {
				return true
			}
		}
	case []interface{}:
		for _, x := range sec {
			s, _ := x.(string)
			if s == "psk" || s == "ieee8021x" || s == "wep" || s == "wpa" || s == "rsn" {
				return true
			}
		}
	}
	return false
}

func dbusUint(v dbus.Variant) uint32 {
	switch n := v.Value().(type) {
	case byte:
		return uint32(n)
	case uint16:
		return uint32(n)
	case uint32:
		return n
	case int16:
		if n > 0 {
			return uint32(n)
		}
	case int32:
		if n > 0 {
			return uint32(n)
		}
	}
	return 0
}
