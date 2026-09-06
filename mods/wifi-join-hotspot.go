package mods

// Hotspot / 192.168.4.1 WiFi join. Do not call functions from wifi-join-lan.go.
// Helpers here are hs*-prefixed copies so :8080 edits cannot change this path.

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

func (m *WifiSetup) tryHomeWifiFromHotspot(creds wifiCreds, prevSSID, prevSvc string) {
	time.Sleep(wifiAckHold)
	m.mu.Lock()
	m.phase = wifiPhaseTrying
	m.mu.Unlock()
	hsSetBusy(true)
	_ = os.WriteFile(wifiTryingFlag, []byte(creds.SSID+"\n"), 0644)
	hsSetHoldAP(true) // like BLE wifiWatcher->Disable()
	wifiLog("hotspot start (BLE-minimal) %s", hsSnapshot(creds.SSID))

	hsPurgeSSID(creds.SSID)
	_ = os.Remove(wifiPreferBleFlag)

	// Only extra vs BLE: open AP stole wlan0 — bring client stack back once.
	wifiLog("hotspot disable AP %s", hsSnapshot(creds.SSID))
	_ = hsDisableAP()
	_ = os.Remove(wifiSetupAPFlag)
	hsBringUpClientStack()
	_ = hsWaitWifiReady(8 * time.Second)
	if hsServiceID(creds.SSID) == "" {
		_ = hsScan()
		_ = hsWaitService(creds.SSID, 8*time.Second)
	}

	// Stock BLE: RegisterAgent + Service.Connect — that call alone does
	// assoc + DHCP; RTS does not poll IP / rewrite provision / re-Connect.
	connectErr := hsConnectLikeBLE(creds.SSID, creds.Pass, creds.Hidden)
	if connectErr != nil && !strings.Contains(connectErr.Error(), "invalid-key") {
		wifiLog("hotspot connect err=%v — one soft retry %s", connectErr, hsSnapshot(creds.SSID))
		_ = hsScan()
		time.Sleep(400 * time.Millisecond)
		connectErr = hsConnectLikeBLE(creds.SSID, creds.Pass, creds.Hidden)
	}
	wifiLog("hotspot connect done err=%v %s", connectErr, hsSnapshot(creds.SSID))

	if connectErr != nil {
		errMsg := "Sai mật khẩu hoặc không thấy mạng — thử lại."
		if strings.Contains(connectErr.Error(), "invalid-key") {
			errMsg = "Sai mật khẩu WiFi nhà — thử lại."
		} else if strings.Contains(connectErr.Error(), "không thấy service") {
			errMsg = "Không thấy mạng — kiểm tra SSID 2.4 GHz rồi thử lại."
		}
		m.finishJoinHotspot(false, creds.SSID, errMsg)
		return
	}

	// BLE: Connect success ⇒ done (optional short ONLINE poll). Never
	// re-Connect while associated — that aborted DHCP on robot logs.
	_ = hsWaitJoin(creds.SSID, 20*time.Second)
	if hsAPOn() {
		_ = hsDisableAP()
		_ = os.Remove(wifiSetupAPFlag)
	}
	if !hsAssociated(creds.SSID) && !hsJoined(creds.SSID) {
		wifiLog("hotspot connect ok but not associated %s", hsSnapshot(creds.SSID))
		m.finishJoinHotspot(false, creds.SSID, "Không giữ được kết nối WiFi nhà — thử lại.")
		return
	}

	hsDisableP2PTwin(creds.SSID)
	hsPreferOnlySSID(creds.SSID)
	// Persist in background — do not block / bounce like old wireos-wifi.config path.
	go func(ssid, pass string, hidden bool) {
		_ = hsWriteProvision(ssid, pass, hidden)
	}(creds.SSID, creds.Pass, creds.Hidden)

	wifiLog("hotspot ok (BLE-minimal) %s", hsSnapshot(creds.SSID))
	m.finishJoinHotspot(true, creds.SSID, "")
	go func() {
		time.Sleep(2 * time.Second)
		_ = exec.Command("systemctl", "restart", "chronyd").Run()
	}()
}

func (m *WifiSetup) finishJoinHotspot(ok bool, ssid, errMsg string) {
	_ = os.Remove(wifiPendingFile)
	m.mu.Lock()
	if ok {
		m.phase = wifiPhaseOK
		m.lastErr = ""
		_ = os.Remove(wifiLastErrorFile)
		_ = os.Remove(wifiForceAPFlag)
		m.pulseWifiFace("ok", wifiFaceHold)
		m.mu.Unlock()
		wifiLog("hotspot finish ok — hold AP until home IP (no re-Connect)")
		go func() {
			// Like BLE: after Connect success, just leave ConnMan alone.
			const ticks = 45
			okStreak := 0
			for i := 0; i < ticks; i++ {
				if hsAPOn() {
					_ = hsDisableAP()
					_ = os.Remove(wifiSetupAPFlag)
				}
				if hsJoined(ssid) {
					okStreak++
					if okStreak >= 2 {
						break
					}
				} else if !hsAssociated(ssid) {
					okStreak = 0
					wifiLog("hotspot grace lost assoc i=%d %s", i, hsSnapshot(ssid))
					_ = hsConnectPreferred(ssid, "")
				}
				time.Sleep(1 * time.Second)
			}
			_ = os.Remove(wifiTryingFlag)
			hsSetBusy(false)
			hsSetHoldAP(false)
			wifiLog("hotspot grace done ap=%v %s", hsAPOn(), hsSnapshot(ssid))
		}()
		return
	}
	m.phase = wifiPhaseFail
	m.lastErr = errMsg
	_ = os.WriteFile(wifiLastErrorFile, []byte(errMsg+"\n"), 0644)
	m.pulseWifiFace("fail", wifiFaceHold)
	m.mu.Unlock()
	// Wrong password / missing network / any hard fail → always bring the
	// open AP back so the phone can retry (required for hotspot mode).
	_ = os.Remove(wifiTryingFlag)
	hsSetBusy(false)
	wifiLog("hotspot restore AP failed=%q err=%q", ssid, errMsg)
	hsSetHoldAP(true)
	hsCleanupFailedJoin(ssid)
	hsWaitRadioIdle(5 * time.Second)
	_ = os.WriteFile(wifiForceAPFlag, []byte("1\n"), 0644)
	apName := hsRobotName()
	for i := 0; i < 6; i++ {
		wifiLog("hotspot AP start try=%d %s", i+1, hsSnapshot(""))
		_ = exec.Command(wifiSetupAPBin, "on", apName).Run()
		time.Sleep(2 * time.Second)
		if hsAPOn() {
			break
		}
	}
	hsSetHoldAP(false)
	wifiLog("hotspot AP restored ap=%v force=1 %s", hsAPOn(), hsSnapshot(""))
}

// hsStabilizeHome keeps killing any AP resurrection and re-asserting Connect
// until home WiFi stays associated+IP for a few seconds (or timeout).
func hsStabilizeHome(ssid string, d time.Duration) bool {
	deadline := time.Now().Add(d)
	stableNeeded := 4 // ~2s at 500ms
	stable := 0
	for time.Now().Before(deadline) {
		if hsAPOn() {
			wifiLog("hotspot stabilize: kill AP during home join")
			_ = hsDisableAP()
			_ = os.Remove(wifiSetupAPFlag)
			stable = 0
		}
		if hsJoined(ssid) {
			stable++
			if stable >= stableNeeded {
				return true
			}
		} else if hsAssociated(ssid) {
			// Associated / link-local — wait, do not Connect again.
			stable = 0
		} else {
			stable = 0
			if hsServiceID(ssid) == "" {
				_ = hsScan()
			}
			_ = hsConnectPreferred(ssid, "")
		}
		time.Sleep(500 * time.Millisecond)
	}
	return hsJoined(ssid) || hsAssociated(ssid)
}

// hsBringUpClientStack restarts wpa_supplicant + ConnMan after open-AP tore
// them down. Matches what used to happen only after the first Connect failure.
func hsBringUpClientStack() {
	wifiLog("hotspot bring-up client stack (restart wpa+connman)")
	_ = exec.Command("systemctl", "restart", "wpa_supplicant").Run()
	_ = exec.Command("systemctl", "restart", "connman").Run()
	time.Sleep(1500 * time.Millisecond)
	hsEnableWifi()
}

// hsRecoverAfterProvision waits for ConnMan to settle after rewriting
// wireos-wifi.config (often briefly State=failure / no IP).
func hsRecoverAfterProvision(ssid string, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if hsAPOn() {
			_ = hsDisableAP()
			_ = os.Remove(wifiSetupAPFlag)
		}
		if hsJoined(ssid) {
			// Need a couple of stable ticks after the bounce.
			ok := true
			for i := 0; i < 4; i++ {
				time.Sleep(500 * time.Millisecond)
				if !hsJoined(ssid) {
					ok = false
					break
				}
			}
			if ok {
				return true
			}
		}
		_ = hsConnectPreferred(ssid, "")
		time.Sleep(500 * time.Millisecond)
	}
	return hsJoined(ssid)
}

func hsPurgeSSID(ssid string) {
	if ssid == "" {
		return
	}
	_ = hsDisconnectNamed(ssid)
	_ = hsRemoveProvisionSSID(ssid)
	hsRemoveConnmanDirsNamed(ssid)
}

func hsCleanupFailedJoin(ssid string) {
	if ssid == "" {
		return
	}
	_ = hsDisconnectNamed(ssid)
	_ = hsRemoveProvisionSSID(ssid)
	hsRemoveConnmanDirsNamed(ssid)
}

func hsWaitRadioIdle(d time.Duration) {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if hsClientSSID() == "" && !hsHasClientIP() {
			return
		}
		time.Sleep(400 * time.Millisecond)
	}
}

func hsHasClientIP() bool {
	for _, ip := range hsAddrs() {
		if ip == wifiOpenAPIP || strings.HasPrefix(ip, "169.254.") {
			continue
		}
		return true
	}
	return false
}

func hsRemoveProvisionSSID(ssid string) error {
	b, err := os.ReadFile(wifiConnmanConfig)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var keep []string
	for _, block := range hsSplitServiceBlocks(string(b)) {
		if hsParseName([]byte(block)) == ssid {
			continue
		}
		if s := strings.TrimSpace(block); s != "" {
			keep = append(keep, s)
		}
	}
	if len(keep) == 0 {
		return os.Remove(wifiConnmanConfig)
	}
	return os.WriteFile(wifiConnmanConfig, []byte(strings.Join(keep, "\n\n")+"\n"), 0600)
}

func hsRemoveConnmanDirsNamed(ssid string) {
	ents, err := os.ReadDir(wifiConnmanStateDir)
	if err != nil {
		return
	}
	for _, e := range ents {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "wifi") {
			continue
		}
		dir := filepath.Join(wifiConnmanStateDir, e.Name())
		b, err := os.ReadFile(filepath.Join(dir, "settings"))
		if err != nil {
			continue
		}
		if hsParseName(b) != ssid {
			continue
		}
		_ = os.RemoveAll(dir)
	}
}

func hsSplitServiceBlocks(txt string) []string {
	txt = strings.ReplaceAll(txt, "\r\n", "\n")
	parts := strings.Split(txt, "[service_")
	var blocks []string
	for i, p := range parts {
		if i == 0 {
			if strings.TrimSpace(p) != "" && hsParseName([]byte(p)) != "" {
				blocks = append(blocks, p)
			}
			continue
		}
		blocks = append(blocks, "[service_"+p)
	}
	return blocks
}

func hsAssociated(ssid string) bool {
	if hsAPOn() || ssid == "" {
		return false
	}
	return hsClientSSID() == ssid
}

func hsJoined(ssid string) bool {
	if !hsAssociated(ssid) {
		return false
	}
	// BLE treats CONNECTED/ONLINE as success; ConnMan ready/online is enough
	// even if DHCP is a beat late (IP check alone raced with AP resurrect).
	st := hsServiceState(ssid)
	if st == "online" || st == "ready" {
		return true
	}
	// Mid-DHCP: association/configuration with link-local — not joined yet.
	for _, ip := range hsAddrs() {
		if ip == wifiOpenAPIP || strings.HasPrefix(ip, "169.254.") {
			continue
		}
		return true
	}
	return false
}

func hsWaitJoin(ssid string, d time.Duration) bool {
	deadline := time.Now().Add(d)
	n := 0
	assocLogged := false
	for time.Now().Before(deadline) {
		st := hsServiceState(ssid)
		if st == "failure" && !hsAssociated(ssid) {
			wifiLog("hotspot wait failure state ssid=%q", ssid)
			return false
		}
		if n%4 == 0 {
			wifiLog("hotspot wait tick ap=%v got=%q state=%q ips=%v svc=%s",
				hsAPOn(), hsClientSSID(), st, hsAddrs(), hsServiceID(ssid))
		}
		if hsJoined(ssid) {
			wifiLog("hotspot wait ok got=%q state=%q ips=%v", hsClientSSID(), st, hsAddrs())
			return true
		}
		// Robot log: already on Huynh 2.4 with ips=[] / 169.254 — DHCP in
		// progress. Re-Connect here aborted DHCP and we timed out as "fail".
		if hsAssociated(ssid) {
			if !assocLogged {
				wifiLog("hotspot associated — waiting DHCP (no reconnect) %s", hsSnapshot(ssid))
				assocLogged = true
			}
			n++
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if n > 0 && n%6 == 0 {
			if hsServiceID(ssid) == "" {
				_ = hsScan()
			}
			_ = hsConnectPreferred(ssid, "")
		}
		n++
		time.Sleep(500 * time.Millisecond)
	}
	st := hsServiceState(ssid)
	ok := hsJoined(ssid)
	wifiLog("hotspot wait timeout ok=%v assoc=%v ap=%v got=%q state=%q ips=%v",
		ok, hsAssociated(ssid), hsAPOn(), hsClientSSID(), st, hsAddrs())
	// Associated but DHCP slow: still success — hold AP off and let IP arrive.
	if !ok && hsAssociated(ssid) {
		wifiLog("hotspot wait: associated without LAN IP yet — treat as ok %s", hsSnapshot(ssid))
		return true
	}
	return ok
}

func hsWaitWifiReady(d time.Duration) bool {
	deadline := time.Now().Add(d)
	n := 0
	for time.Now().Before(deadline) {
		if _, err := os.Stat(wifiSetupAPFlag); err == nil {
			_ = hsDisableAP()
			_ = os.Remove(wifiSetupAPFlag)
		}
		hsEnableWifi()
		powered := hsWifiPowered()
		if n%3 == 0 {
			out, err := hsConnmanctl(3*time.Second, "services")
			wifiLog("hotspot wifi ready err=%v powered=%v services=%q", err, powered, strings.TrimSpace(string(out)))
		}
		if powered {
			return true
		}
		n++
		time.Sleep(400 * time.Millisecond)
	}
	return hsWifiPowered()
}

func hsConnmanctl(d time.Duration, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	return exec.CommandContext(ctx, "connmanctl", args...).CombinedOutput()
}

func hsWaitService(ssid string, d time.Duration) string {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		hsEnableWifi()
		if id := hsServiceID(ssid); id != "" {
			wifiLog("hotspot found service %s", id)
			return id
		}
		_ = hsScan()
		time.Sleep(600 * time.Millisecond)
	}
	return hsServiceID(ssid)
}

func hsEnableWifi() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = exec.CommandContext(ctx, "busctl", "--system", "call",
		"net.connman", "/net/connman/technology/wifi",
		"net.connman.Technology", "SetProperty", "sv", "Powered", "b", "true").Run()
	_ = exec.CommandContext(ctx, "connmanctl", "enable", "wifi").Run()
	_ = exec.Command("ip", "link", "set", "wlan0", "up").Run()
}

func hsWifiPowered() bool {
	out, err := hsConnmanctl(5*time.Second, "technologies")
	if err != nil {
		return false
	}
	s := string(out)
	i := strings.Index(s, "/net/connman/technology/wifi")
	if i < 0 {
		return false
	}
	chunk := s[i:]
	if j := strings.Index(chunk[1:], "/net/connman/technology/"); j >= 0 {
		chunk = chunk[:j+1]
	}
	return strings.Contains(chunk, "Powered = True")
}

func hsStabilize(ssid string) {
	defer func() {
		_ = os.Remove(wifiTryingFlag)
		hsSetBusy(false)
		hsSetHoldAP(false)
	}()
	if ssid == "" {
		return
	}
	deadline := time.Now().Add(40 * time.Second)
	for time.Now().Before(deadline) {
		if hsAPOn() {
			_ = hsDisableAP()
			_ = os.Remove(wifiSetupAPFlag)
		}
		got := hsClientSSID()
		if hsJoined(ssid) {
			time.Sleep(3 * time.Second)
			if hsJoined(ssid) {
				return
			}
		}
		if got != ssid {
			_ = hsConnectNamed(ssid)
		}
		time.Sleep(2 * time.Second)
	}
}

func hsSnapshot(ssid string) string {
	return fmt.Sprintf("ssid=%q got=%q state=%q ap=%v ips=%v svc=%s",
		ssid, hsClientSSID(), hsServiceState(ssid), hsAPOn(), hsAddrs(), hsServiceID(ssid))
}

func hsSetBusy(on bool) {
	if on {
		_ = os.WriteFile(wifiBusyFile, []byte("1\n"), 0644)
		return
	}
	_ = os.Remove(wifiBusyFile)
}

func hsSetHoldAP(on bool) {
	if on {
		_ = os.WriteFile(wifiHoldAPFlag, []byte("1\n"), 0644)
		return
	}
	_ = os.Remove(wifiHoldAPFlag)
}

func hsDisableAP() error {
	_ = exec.Command(wifiSetupAPBin, "off").Run()
	_ = exec.Command("/anki/bin/vic-setup-ap", "off").Run()
	return nil
}

func hsRobotName() string {
	out, err := exec.Command("/bin/getprop", "anki.robot.name").Output()
	name := strings.TrimSpace(string(out))
	if err != nil || name == "" {
		return "Vector"
	}
	return name
}

func hsAPOn() bool {
	if _, err := os.Stat(wifiSetupAPFlag); err == nil {
		return true
	}
	for _, ip := range hsAddrs() {
		if ip == wifiOpenAPIP {
			return true
		}
	}
	return false
}

func hsParseName(cfg []byte) string {
	for _, line := range strings.Split(string(cfg), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Name = ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Name = "))
		}
		if strings.HasPrefix(line, "Name=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Name="))
		}
	}
	return ""
}

func hsClientSSID() string {
	if out, err := exec.Command("iw", "dev", "wlan0", "link").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "SSID:") {
				return strings.TrimSpace(strings.TrimPrefix(line, "SSID:"))
			}
		}
	}
	// D-Bus Name on wlan0 only — never trust p2p0 / broken connmanctl flag parse.
	if name := hsDBusClientSSID(); name != "" {
		return name
	}
	out, err := hsConnmanctl(5*time.Second, "services")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		// Online (*AO) before associated-only (*A ) — avoids p2p twin listed first.
		if !strings.Contains(line, "*AO") && !strings.Contains(line, "*A ") {
			continue
		}
		name, _ := hsParseCtl(line)
		if name != "" {
			return name
		}
	}
	return ""
}

func hsDBusClientSSID() string {
	bus, err := dbus.SystemBus()
	if err != nil {
		return ""
	}
	manager := bus.Object("net.connman", "/")
	var services [][]interface{}
	if err := manager.Call("net.connman.Manager.GetServices", 0).Store(&services); err != nil {
		return ""
	}
	for _, pair := range services {
		if len(pair) < 2 {
			continue
		}
		props, ok := pair[1].(map[string]dbus.Variant)
		if !ok {
			continue
		}
		if typ, _ := props["Type"].Value().(string); typ != "wifi" {
			continue
		}
		if hsEthernetIface(props) != "wlan0" {
			continue
		}
		st, _ := props["State"].Value().(string)
		if st != "online" && st != "ready" && st != "association" && st != "configuration" {
			continue
		}
		if name, _ := props["Name"].Value().(string); name != "" {
			return name
		}
	}
	return ""
}

func hsAddrs() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			ip, _, err := net.ParseCIDR(a.String())
			if err != nil || ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}
			ips = append(ips, ip.String())
		}
	}
	return ips
}

func hsParseCtl(line string) (name, svc string) {
	line = strings.TrimSpace(line)
	idx := strings.LastIndex(line, " wifi_")
	if idx < 0 {
		return "", ""
	}
	svc = strings.TrimSpace(line[idx+1:])
	left := strings.TrimSpace(line[:idx])
	// ConnMan pads flags: "*AO Name", "*A  Name", and log once showed "*Aa Name".
	// strings.Fields splits the flag token from the SSID reliably.
	if strings.HasPrefix(left, "*") {
		fields := strings.Fields(left)
		if len(fields) >= 2 {
			left = strings.Join(fields[1:], " ")
		} else {
			left = ""
		}
	}
	return left, svc
}

func hsServiceID(ssid string) string {
	if ssid == "" {
		return ""
	}
	bus, err := dbus.SystemBus()
	if err == nil {
		if path, err := hsFindWifiServicePath(bus, ssid, false); err == nil {
			return strings.TrimPrefix(string(path), "/net/connman/service/")
		}
	}
	out, err := hsConnmanctl(5*time.Second, "services")
	if err != nil {
		return ""
	}
	hexSSID := strings.ToLower(hex.EncodeToString([]byte(ssid)))
	var hexHit string
	for _, line := range strings.Split(string(out), "\n") {
		name, svc := hsParseCtl(line)
		if svc == "" {
			continue
		}
		if name == ssid {
			return svc
		}
		if hexHit == "" && hexSSID != "" && strings.Contains(strings.ToLower(svc), hexSSID) {
			hexHit = svc
		}
	}
	return hexHit
}

func hsConnectService(svc string) error {
	if svc == "" {
		return fmt.Errorf("empty service")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := exec.CommandContext(ctx, "connmanctl", "connect", svc).Run(); err == nil {
		return nil
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	return exec.CommandContext(ctx2, "busctl", "--system", "call",
		"net.connman", "/net/connman/service/"+svc,
		"net.connman.Service", "Connect").Run()
}

func hsConnectNamed(ssid string) error {
	return hsConnectService(hsServiceID(ssid))
}

func hsDisconnectNamed(ssid string) error {
	svc := hsServiceID(ssid)
	if svc == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "connmanctl", "disconnect", svc).Run()
}

func hsConnectPreferred(ssid, svc string) error {
	if svc != "" {
		if err := hsConnectService(svc); err == nil {
			return nil
		}
	}
	if ssid == "" {
		return nil
	}
	if hsServiceID(ssid) == "" {
		_ = hsScan()
		time.Sleep(1500 * time.Millisecond)
	}
	return hsConnectNamed(ssid)
}

func hsServiceState(ssid string) string {
	svc := hsServiceID(ssid)
	if svc == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "busctl", "--system", "get-property",
		"net.connman", "/net/connman/service/"+svc, "net.connman.Service", "State").Output()
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(out))
	s = strings.TrimPrefix(s, "s ")
	return strings.Trim(s, `"`)
}

func hsScan() error {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	if err := exec.CommandContext(ctx, "connmanctl", "scan", "wifi").Run(); err == nil {
		return nil
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel2()
	return exec.CommandContext(ctx2, "busctl", "--system", "call",
		"net.connman", "/net/connman/technology/wifi",
		"net.connman.Technology", "Scan").Run()
}

func hsServiceBlock(id string, creds wifiCreds) []byte {
	hexSSID := hex.EncodeToString([]byte(creds.SSID))
	sec := "psk"
	passLine := "Passphrase = " + creds.Pass + "\n"
	if creds.Pass == "" {
		sec = "none"
		passLine = ""
	}
	hiddenLine := ""
	if creds.Hidden {
		hiddenLine = "Hidden = true\n"
	}
	body := "[service_" + id + "]\n" +
		"Type = wifi\n" +
		"Name = " + creds.SSID + "\n" +
		"SSID = " + hexSSID + "\n" +
		"Security = " + sec + "\n" +
		passLine +
		hiddenLine +
		"IPv4 = dhcp\n"
	return []byte(body)
}

func hsWriteProvision(ssid, pass string, hidden bool) error {
	if err := os.MkdirAll(wifiConnmanStateDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(wifiConnmanConfig, hsServiceBlock("wireos", wifiCreds{
		SSID: ssid, Pass: pass, Hidden: hidden,
	}), 0600)
}

func hsPreferOnlySSID(ssid string) {
	ents, err := os.ReadDir(wifiConnmanStateDir)
	if err != nil {
		return
	}
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(wifiConnmanStateDir, e.Name(), "settings")
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		txt := string(b)
		if !strings.Contains(txt, "Type = wifi") && !strings.Contains(txt, "Type=wifi") {
			continue
		}
		if strings.Contains(txt, "Name = "+ssid) || strings.Contains(txt, "Name="+ssid) {
			continue
		}
		out := strings.ReplaceAll(txt, "AutoConnect = true", "AutoConnect = false")
		out = strings.ReplaceAll(out, "AutoConnect=true", "AutoConnect=false")
		if !strings.Contains(strings.ToLower(out), "autoconnect") {
			out += "\nAutoConnect = false\n"
		}
		_ = os.WriteFile(p, []byte(out), 0600)
	}
}
