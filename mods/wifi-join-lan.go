package mods

// LAN / :8080 WiFi join. Do not call functions from wifi-join-hotspot.go.
// Helpers here are lan*-prefixed copies so hotspot edits cannot change this path.

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

func (m *WifiSetup) tryHomeWifiFromLAN(creds wifiCreds, prevSSID, prevSvc string) {
	time.Sleep(wifiAckHold)
	m.mu.Lock()
	m.phase = wifiPhaseTrying
	m.mu.Unlock()
	lanSetBusy(true)
	_ = os.WriteFile(wifiTryingFlag, []byte(creds.SSID+"\n"), 0644)
	lanSetHoldAP(true)
	wifiLog("lan start %s", lanSnapshot(creds.SSID))

	prev, _ := os.ReadFile(wifiConnmanConfig)
	if prevSSID == "" {
		prevSSID = lanParseName(prev)
	}
	if prevSvc == "" && prevSSID != "" {
		prevSvc = lanServiceID(prevSSID)
	}

	if cur := lanClientSSID(); cur != "" {
		wifiLog("lan drop current %q before try %q", cur, creds.SSID)
		_ = lanDisconnectNamed(cur)
		_ = lanWaitLeft(cur, 8*time.Second)
	}

	if err := lanWriteTryWithPrev(prev, creds); err != nil {
		wifiLog("lan write provision failed: %v", err)
		m.finishJoinLAN(false, prev, prevSSID, prevSvc, "không ghi được cấu hình WiFi")
		return
	}
	_ = os.Remove(wifiPreferBleFlag)
	go func() { _ = lanConnectNamed(creds.SSID) }()

	if lanWaitJoinOrFail(creds.SSID, wifiLanTryTimeout) {
		_ = lanWriteProvision(creds.SSID, creds.Pass, creds.Hidden)
		lanPreferOnlySSID(creds.SSID)
		wifiLog("lan ok %s", lanSnapshot(creds.SSID))
		go lanStabilize(creds.SSID)
		m.finishJoinLAN(true, prev, prevSSID, prevSvc, "")
		return
	}

	if prevSSID == "" {
		prevSSID = lanParseName(prev)
	}
	wifiLog("lan fail %s", lanSnapshot(creds.SSID))
	m.lanMarkFail("Sai mật khẩu — đang bắt lại WiFi cũ…")
	_ = lanDisconnectNamed(creds.SSID)
	lanSetHoldAP(true)
	back := lanRestorePrev(prev, prevSSID, prevSvc, creds.SSID)
	errMsg := "Sai mật khẩu — không vào được mạng mới."
	if back && prevSSID != "" {
		errMsg = "Sai mật khẩu — đã quay lại WiFi cũ: " + prevSSID
	} else if prevSSID != "" {
		errMsg = "Sai mật khẩu — đang bắt lại WiFi cũ: " + prevSSID
	}
	m.lanMarkFail(errMsg)
	m.finishJoinLAN(false, prev, prevSSID, prevSvc, errMsg)
}

func (m *WifiSetup) finishJoinLAN(ok bool, prevConfig []byte, prevSSID, prevSvc, errMsg string) {
	_ = os.Remove(wifiPendingFile)
	_ = os.Remove(wifiTryingFlag)
	lanSetBusy(false)
	m.mu.Lock()
	if ok {
		m.phase = wifiPhaseOK
		m.lastErr = ""
		_ = os.Remove(wifiLastErrorFile)
		m.mu.Unlock()
		lanSetHoldAP(false)
		wifiLog("lan finish ok")
		return
	}
	m.phase = wifiPhaseFail
	m.lastErr = errMsg
	_ = os.WriteFile(wifiLastErrorFile, []byte(errMsg+"\n"), 0644)
	m.mu.Unlock()
	wifiLog("lan finish fail %s", errMsg)
	if prevSSID != "" && lanJoined(prevSSID) {
		wifiLog("lan already back on prev %s", lanSnapshot(prevSSID))
		lanSetHoldAP(false)
		return
	}
	lanSetHoldAP(true)
	go lanKeepRetryPrev(prevSSID, "")
}

func (m *WifiSetup) lanMarkFail(msg string) {
	m.mu.Lock()
	m.phase = wifiPhaseFail
	m.lastErr = msg
	m.mu.Unlock()
	_ = os.WriteFile(wifiLastErrorFile, []byte(msg+"\n"), 0644)
}

func lanRestorePrev(prev []byte, prevSSID, prevSvc, failedSSID string) bool {
	wifiLog("lan restore start prev=%q failed=%q oldsvc=%s", prevSSID, failedSSID, prevSvc)
	if failedSSID != "" {
		_ = lanDisconnectNamed(failedSSID)
		if failedSSID != prevSSID {
			lanSetAutoConnectNamed(failedSSID, false)
			_ = lanConnmanAutoConnect(failedSSID, false)
		}
		lanRemoveConnmanDirsNamed(failedSSID)
	}
	if len(prev) > 0 {
		_ = os.WriteFile(wifiConnmanConfig, prev, 0600)
	} else if failedSSID != "" && failedSSID != prevSSID {
		_ = lanRemoveProvisionSSID(failedSSID)
	}
	if prevSSID == "" {
		prevSSID = lanParseName(prev)
	}
	if prevSSID == "" {
		ok := lanClientSSID() != "" && !lanAPOn()
		wifiLog("lan restore no prev ssid ok=%v %s", ok, lanSnapshot(""))
		return ok
	}
	lanPreferOnlySSID(prevSSID)
	lanSetAutoConnectNamed(prevSSID, true)
	time.Sleep(500 * time.Millisecond)
	if pass := lanPassphraseForSSID(prev, prevSSID); pass != "" {
		_ = lanSetPassphrase(prevSSID, pass)
	}
	_ = lanConnmanAutoConnect(prevSSID, true)
	_ = lanScan()
	time.Sleep(1200 * time.Millisecond)
	_ = lanConnectNamed(prevSSID)
	if lanWaitJoined(prevSSID, 20*time.Second) {
		wifiLog("lan restore ok %s", lanSnapshot(prevSSID))
		return true
	}
	_ = lanConnectNamed(prevSSID)
	ok := lanWaitJoined(prevSSID, 12*time.Second)
	wifiLog("lan restore done ok=%v %s", ok, lanSnapshot(prevSSID))
	return ok
}

func lanKeepRetryPrev(ssid, svc string) {
	defer func() {
		_ = os.Remove(wifiTryingFlag)
		lanSetBusy(false)
		lanSetHoldAP(false)
	}()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		if ssid != "" && lanJoined(ssid) {
			return
		}
		if lanClientSSID() != "" && !lanAPOn() {
			return
		}
		if ssid != "" {
			lanSetAutoConnectNamed(ssid, true)
			_ = lanConnmanAutoConnect(ssid, true)
			_ = lanConnectNamed(ssid)
		} else {
			_ = lanConnectPreferred(ssid, svc)
		}
		time.Sleep(4 * time.Second)
	}
}

func lanWaitJoined(ssid string, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if lanJoined(ssid) {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func lanWaitLeft(ssid string, d time.Duration) bool {
	if ssid == "" {
		return true
	}
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if lanClientSSID() != ssid {
			wifiLog("lan left %q got=%q", ssid, lanClientSSID())
			return true
		}
		time.Sleep(300 * time.Millisecond)
	}
	left := lanClientSSID() != ssid
	wifiLog("lan wait-left done left=%v got=%q", left, lanClientSSID())
	return left
}

func lanWaitJoinOrFail(ssid string, d time.Duration) bool {
	sawIdle := lanClientSSID() != ssid
	deadline := time.Now().Add(d)
	n := 0
	for time.Now().Before(deadline) {
		if lanAPOn() {
			time.Sleep(400 * time.Millisecond)
			continue
		}
		got := lanClientSSID()
		st := lanServiceState(ssid)
		if got != ssid {
			sawIdle = true
		}
		if n%5 == 0 {
			wifiLog("lan wait tick idle=%v got=%q state=%q ips=%v svc=%s",
				sawIdle, got, st, lanAddrs(), lanServiceID(ssid))
		}
		if st == "failure" {
			wifiLog("lan wait failure state ssid=%q", ssid)
			return false
		}
		if sawIdle && got == ssid && (st == "ready" || st == "online") {
			wifiLog("lan wait ok got=%q state=%q", got, st)
			return true
		}
		n++
		time.Sleep(400 * time.Millisecond)
	}
	if lanAPOn() {
		return false
	}
	got := lanClientSSID()
	st := lanServiceState(ssid)
	ok := sawIdle && got == ssid && (st == "ready" || st == "online")
	wifiLog("lan wait timeout ok=%v idle=%v got=%q state=%q ips=%v", ok, sawIdle, got, st, lanAddrs())
	return ok
}

func lanStabilize(ssid string) {
	defer func() {
		_ = os.Remove(wifiTryingFlag)
		lanSetBusy(false)
		lanSetHoldAP(false)
	}()
	if ssid == "" {
		return
	}
	deadline := time.Now().Add(40 * time.Second)
	for time.Now().Before(deadline) {
		got := lanClientSSID()
		if got == ssid {
			time.Sleep(3 * time.Second)
			if lanClientSSID() == ssid {
				return
			}
		}
		if got != ssid {
			_ = lanConnectNamed(ssid)
		}
		time.Sleep(2 * time.Second)
	}
}

func lanSnapshot(ssid string) string {
	return fmt.Sprintf("ssid=%q got=%q state=%q ap=%v ips=%v svc=%s",
		ssid, lanClientSSID(), lanServiceState(ssid), lanAPOn(), lanAddrs(), lanServiceID(ssid))
}

func lanSetBusy(on bool) {
	if on {
		_ = os.WriteFile(wifiBusyFile, []byte("1\n"), 0644)
		return
	}
	_ = os.Remove(wifiBusyFile)
}

func lanSetHoldAP(on bool) {
	if on {
		_ = os.WriteFile(wifiHoldAPFlag, []byte("1\n"), 0644)
		return
	}
	_ = os.Remove(wifiHoldAPFlag)
}

func lanAPOn() bool {
	if _, err := os.Stat(wifiSetupAPFlag); err == nil {
		return true
	}
	for _, ip := range lanAddrs() {
		if ip == wifiOpenAPIP {
			return true
		}
	}
	return false
}

func lanJoined(ssid string) bool {
	got := lanClientSSID()
	if got == ssid {
		return true
	}
	if lanAPOn() {
		return false
	}
	if got != "" && got != ssid {
		return false
	}
	for _, ip := range lanAddrs() {
		if ip == wifiOpenAPIP || strings.HasPrefix(ip, "169.254.") {
			continue
		}
		return true
	}
	return false
}

func lanParseName(cfg []byte) string {
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

func lanClientSSID() string {
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
		name, _ := lanParseCtl(line)
		if name != "" {
			return name
		}
	}
	return ""
}

func lanAddrs() []string {
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

func lanParseCtl(line string) (name, svc string) {
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

func lanServiceID(ssid string) string {
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
		name, svc := lanParseCtl(line)
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

func lanConnectService(svc string) error {
	if svc == "" {
		return nil
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

func lanConnectNamed(ssid string) error {
	return lanConnectService(lanServiceID(ssid))
}

func lanConnectPreferred(ssid, svc string) error {
	if svc != "" {
		if err := lanConnectService(svc); err == nil {
			return nil
		}
	}
	if ssid == "" {
		return nil
	}
	if lanServiceID(ssid) == "" {
		_ = lanScan()
		time.Sleep(1500 * time.Millisecond)
	}
	return lanConnectNamed(ssid)
}

func lanDisconnectNamed(ssid string) error {
	svc := lanServiceID(ssid)
	if svc == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "connmanctl", "disconnect", svc).Run()
}

func lanConnmanAutoConnect(ssid string, on bool) error {
	svc := lanServiceID(ssid)
	if svc == "" {
		return nil
	}
	val := "no"
	if on {
		val = "yes"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "connmanctl", "config", svc, "--autoconnect", val).Run()
}

func lanServiceState(ssid string) string {
	svc := lanServiceID(ssid)
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

func lanScan() error {
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

func lanServiceBlock(id string, creds wifiCreds, autoConnect bool) []byte {
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
	ac := "false"
	if autoConnect {
		ac = "true"
	}
	body := "[service_" + id + "]\n" +
		"Type = wifi\n" +
		"Name = " + creds.SSID + "\n" +
		"SSID = " + hexSSID + "\n" +
		"Security = " + sec + "\n" +
		passLine +
		hiddenLine +
		"IPv4 = dhcp\n" +
		"AutoConnect = " + ac + "\n"
	return []byte(body)
}

func lanWriteProvision(ssid, pass string, hidden bool) error {
	if err := os.MkdirAll(wifiConnmanStateDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(wifiConnmanConfig, lanServiceBlock("wireos", wifiCreds{
		SSID: ssid, Pass: pass, Hidden: hidden,
	}, true), 0600)
}

func lanWriteTryWithPrev(prev []byte, creds wifiCreds) error {
	if err := os.MkdirAll(wifiConnmanStateDir, 0755); err != nil {
		return err
	}
	prevTxt := strings.TrimSpace(string(prev))
	same := strings.Contains(prevTxt, "Name = "+creds.SSID+"\n") ||
		strings.Contains(prevTxt, "Name = "+creds.SSID+"\r") ||
		strings.Contains(prevTxt, "Name="+creds.SSID)
	if prevTxt == "" || same {
		return lanWriteProvision(creds.SSID, creds.Pass, creds.Hidden)
	}
	body := prevTxt + "\n\n" + string(lanServiceBlock("wireos_try", creds, true))
	return os.WriteFile(wifiConnmanConfig, []byte(body), 0600)
}

func lanPreferOnlySSID(ssid string) {
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

func lanSetAutoConnectNamed(ssid string, on bool) {
	if ssid == "" {
		return
	}
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
		if !strings.Contains(txt, "Name = "+ssid) && !strings.Contains(txt, "Name="+ssid) {
			continue
		}
		out := txt
		if on {
			out = strings.ReplaceAll(out, "AutoConnect = false", "AutoConnect = true")
			out = strings.ReplaceAll(out, "AutoConnect=false", "AutoConnect=true")
			if !strings.Contains(strings.ToLower(out), "autoconnect") {
				out += "\nAutoConnect = true\n"
			}
		} else {
			out = strings.ReplaceAll(out, "AutoConnect = true", "AutoConnect = false")
			out = strings.ReplaceAll(out, "AutoConnect=true", "AutoConnect=false")
		}
		_ = os.WriteFile(p, []byte(out), 0600)
	}
}

func lanPassphraseForSSID(cfg []byte, ssid string) string {
	if ssid == "" || len(cfg) == 0 {
		return ""
	}
	for _, block := range lanSplitServiceBlocks(string(cfg)) {
		if lanParseName([]byte(block)) != ssid {
			continue
		}
		for _, line := range strings.Split(block, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Passphrase = ") {
				return strings.TrimSpace(strings.TrimPrefix(line, "Passphrase = "))
			}
			if strings.HasPrefix(line, "Passphrase=") {
				return strings.TrimSpace(strings.TrimPrefix(line, "Passphrase="))
			}
		}
	}
	return ""
}

func lanSetPassphrase(ssid, pass string) error {
	if ssid == "" || pass == "" {
		return nil
	}
	svc := lanServiceID(ssid)
	if svc == "" {
		_ = lanScan()
		time.Sleep(1200 * time.Millisecond)
		svc = lanServiceID(ssid)
	}
	if svc == "" {
		return fmt.Errorf("no service for %s", ssid)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "connmanctl", "config", svc, "--passphrase", pass).Run()
}

func lanRemoveProvisionSSID(ssid string) error {
	b, err := os.ReadFile(wifiConnmanConfig)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var keep []string
	for _, block := range lanSplitServiceBlocks(string(b)) {
		if lanParseName([]byte(block)) == ssid {
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

func lanRemoveConnmanDirsNamed(ssid string) {
	if ssid == "" {
		return
	}
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
		if lanParseName(b) != ssid {
			continue
		}
		_ = os.RemoveAll(dir)
	}
}

func lanSplitServiceBlocks(txt string) []string {
	txt = strings.ReplaceAll(txt, "\r\n", "\n")
	parts := strings.Split(txt, "[service_")
	var blocks []string
	for i, p := range parts {
		if i == 0 {
			if strings.TrimSpace(p) != "" && lanParseName([]byte(p)) != "" {
				blocks = append(blocks, p)
			}
			continue
		}
		blocks = append(blocks, "[service_"+p)
	}
	return blocks
}
