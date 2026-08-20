package mods

// Hotspot / 10.3.141.1 WiFi join. Do not call functions from wifi-join-lan.go.
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
)

func (m *WifiSetup) tryHomeWifiFromHotspot(creds wifiCreds, prevSSID, prevSvc string) {
	time.Sleep(wifiAckHold)
	m.mu.Lock()
	m.phase = wifiPhaseTrying
	m.mu.Unlock()
	hsSetBusy(true)
	_ = os.WriteFile(wifiTryingFlag, []byte(creds.SSID+"\n"), 0644)
	wifiLog("hotspot start %s", hsSnapshot(creds.SSID))
	hsPurgeSSID(creds.SSID)

	if err := hsWriteProvision(creds.SSID, creds.Pass, creds.Hidden); err != nil {
		wifiLog("hotspot write provision failed: %v", err)
		m.finishJoinHotspot(false, creds.SSID, "không ghi được cấu hình WiFi")
		return
	}
	hsPreferOnlySSID(creds.SSID)
	_ = os.Remove(wifiPreferBleFlag)
	wifiLog("hotspot disable AP %s", hsSnapshot(creds.SSID))
	_ = hsDisableAP()
	_ = os.Remove(wifiSetupAPFlag)
	if !hsWaitWifiReady(15 * time.Second) {
		wifiLog("hotspot wifi not powered %s", hsSnapshot(creds.SSID))
	}
	svc := hsWaitService(creds.SSID, 20*time.Second)
	if svc == "" {
		wifiLog("hotspot no service after scan %s", hsSnapshot(creds.SSID))
	} else {
		wifiLog("hotspot connect svc=%s", svc)
		if err := hsConnectService(svc); err != nil {
			wifiLog("hotspot connect err=%v %s", err, hsSnapshot(creds.SSID))
		}
	}

	if hsWaitJoin(creds.SSID, wifiHotspotJoinTimeout) {
		if hsAPOn() {
			_ = hsDisableAP()
			_ = os.Remove(wifiSetupAPFlag)
		}
		wifiLog("hotspot ok %s", hsSnapshot(creds.SSID))
		go hsStabilize(creds.SSID)
		m.finishJoinHotspot(true, creds.SSID, "")
		return
	}

	errMsg := "Sai mật khẩu hoặc không thấy mạng — thử lại."
	wifiLog("hotspot fail %s", hsSnapshot(creds.SSID))
	m.finishJoinHotspot(false, creds.SSID, errMsg)
}

func (m *WifiSetup) finishJoinHotspot(ok bool, ssid, errMsg string) {
	_ = os.Remove(wifiPendingFile)
	_ = os.Remove(wifiTryingFlag)
	hsSetBusy(false)
	m.mu.Lock()
	if ok {
		m.phase = wifiPhaseOK
		m.lastErr = ""
		_ = os.Remove(wifiLastErrorFile)
		_ = os.Remove(wifiForceAPFlag)
		m.pulseWifiFace("ok", wifiFaceHold)
		m.mu.Unlock()
		hsSetHoldAP(false)
		wifiLog("hotspot finish ok")
		return
	}
	m.phase = wifiPhaseFail
	m.lastErr = errMsg
	_ = os.WriteFile(wifiLastErrorFile, []byte(errMsg+"\n"), 0644)
	m.pulseWifiFace("fail", wifiFaceHold)
	m.mu.Unlock()
	wifiLog("hotspot restore AP failed=%q", ssid)
	hsSetHoldAP(true)
	hsCleanupFailedJoin(ssid)
	hsWaitRadioIdle(8 * time.Second)
	_ = os.WriteFile(wifiForceAPFlag, []byte("1\n"), 0644)
	apName := hsRobotName()
	for i := 0; i < 6; i++ {
		wifiLog("hotspot AP start try=%d %s", i+1, hsSnapshot(""))
		_ = exec.Command(wifiSetupAPBin, "on", apName).Run()
		time.Sleep(3 * time.Second)
		if hsAPOn() {
			break
		}
	}
	hsSetHoldAP(false)
	wifiLog("hotspot AP restored ap=%v force=1 %s", hsAPOn(), hsSnapshot(""))
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

func hsJoined(ssid string) bool {
	if hsAPOn() || ssid == "" {
		return false
	}
	if hsClientSSID() != ssid {
		return false
	}
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
	for time.Now().Before(deadline) {
		st := hsServiceState(ssid)
		if st == "failure" {
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
	wifiLog("hotspot wait timeout ok=%v ap=%v got=%q state=%q ips=%v", ok, hsAPOn(), hsClientSSID(), st, hsAddrs())
	return ok
}

func hsWaitWifiReady(d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(wifiSetupAPFlag); err == nil {
			_ = hsDisableAP()
			_ = os.Remove(wifiSetupAPFlag)
		}
		hsEnableWifi()
		out, err := exec.Command("connmanctl", "services").CombinedOutput()
		powered := hsWifiPowered()
		wifiLog("hotspot wifi ready err=%v powered=%v services=%q", err, powered, strings.TrimSpace(string(out)))
		if powered {
			return true
		}
		time.Sleep(800 * time.Millisecond)
	}
	return hsWifiPowered()
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
		time.Sleep(1200 * time.Millisecond)
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
	out, err := exec.Command("connmanctl", "technologies").CombinedOutput()
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
	out, err := exec.Command("connmanctl", "services").CombinedOutput()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "*A") {
			continue
		}
		name, _ := hsParseCtl(line)
		if name != "" {
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
	if strings.HasPrefix(left, "*") {
		rest := left[1:]
		i := 0
		for i < len(rest) && i < 3 {
			c := rest[i]
			if c != 'A' && c != 'O' && c != 'I' && c != 'R' && c != 'P' && c != 'C' {
				break
			}
			i++
		}
		if i < len(rest) && rest[i] == ' ' {
			left = strings.TrimSpace(rest[i+1:])
		} else if i == len(rest) {
			left = ""
		}
	}
	return left, svc
}

func hsServiceID(ssid string) string {
	if ssid == "" {
		return ""
	}
	out, err := exec.Command("connmanctl", "services").CombinedOutput()
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
