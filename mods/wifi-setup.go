package mods

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/os-vector/wired/vars"
)

const (
	wifiConnmanConfig      = "/data/lib/connman/wireos-wifi.config"
	wifiSetupAPFlag        = "/run/wireos-setup-ap"
	wifiConnmanStateDir    = "/data/lib/connman"
	wifiOpenAPIP           = "10.3.141.1"
	wifiSetupAPBin         = "/usr/bin/vic-setup-ap"
	wifiScanCache          = "/run/wireos-wifi-scan.json"
	wifiPreferBleFlag      = "/run/wireos-prefer-ble"
	wifiPendingFile        = "/run/wireos-wifi-pending.json"
	wifiTryingFlag         = "/run/wireos-wifi-trying"
	wifiBusyFile           = "/run/wireos-wifi-busy"
	wifiHoldAPFlag         = "/run/wireos-wifi-hold-ap"
	wifiLastErrorFile      = "/run/wireos-wifi-last-error"
	wifiForceAPFlag        = "/run/wireos-force-ap"
	wifiBootWaitFlag       = "/run/wireos-wifi-boot-wait"
	wifiTraceFile          = "/run/wireos-wifi-trace.log"
	wifiJoinTimeout        = 28 * time.Second
	wifiLanTryTimeout      = 22 * time.Second
	wifiHotspotJoinTimeout = 40 * time.Second
	wifiAckHold            = 1200 * time.Millisecond
	wifiSavedGrace         = 90 * time.Second
)

type wifiPhase string

const (
	wifiPhaseIdle     wifiPhase = "idle"
	wifiPhaseAccepted wifiPhase = "accepted"
	wifiPhaseTrying   wifiPhase = "trying"
	wifiPhaseOK       wifiPhase = "ok"
	wifiPhaseFail     wifiPhase = "fail"
)

type wifiCreds struct {
	SSID   string
	Pass   string
	Hidden bool
}

// WifiSetup is the on-robot hotspot + captive portal (ConnMan tethering).
type WifiSetup struct {
	vars.Modification
	mu          sync.Mutex
	phase       wifiPhase
	lastErr     string
	attemptSSID string
	restoreAP   bool
	fromHotspot bool
	prevSSID    string
	prevSvc     string
	uiSource    string
}

func NewWifiSetup() *WifiSetup {
	return &WifiSetup{}
}

func (m *WifiSetup) Name() string { return "WifiSetup" }

func (m *WifiSetup) Description() string {
	return "Setup hotspot + WiFi portal when the robot has no home network."
}

func (m *WifiSetup) Load() error {
	m.phase = wifiPhaseIdle
	if b, err := os.ReadFile(wifiLastErrorFile); err == nil {
		m.phase = wifiPhaseFail
		m.lastErr = strings.TrimSpace(string(b))
	}
	go func() {
		time.Sleep(2 * time.Second)
		clearStaleWifiJoinFlags()
		time.Sleep(13 * time.Second)
		clearStaleWifiJoinFlags()
	}()
	http.HandleFunc("/wifi", m.servePage)
	http.HandleFunc("/wifi/", m.servePage)
	http.HandleFunc("/generate_204", m.captiveProbe)
	http.HandleFunc("/gen_204", m.captiveProbe)
	http.HandleFunc("/hotspot-detect.html", m.captiveProbe)
	http.HandleFunc("/library/test/success.html", m.captiveProbe)
	http.HandleFunc("/canonical.html", m.captiveProbe)
	http.HandleFunc("/ncsi.txt", m.captiveProbe)
	http.HandleFunc("/connecttest.txt", m.captiveProbe)
	http.HandleFunc("/success.txt", m.captiveProbe)
	http.HandleFunc("/kindle-wifi/wifiredirect.html", m.captiveProbe)
	go m.maybeStartSetupAP()
	return nil
}

func (m *WifiSetup) maybeStartSetupAP() {
	started := time.Now()
	time.Sleep(8 * time.Second)
	for {
		if wifiJoinInProgress() {
			time.Sleep(2 * time.Second)
			continue
		}
		if preferBleWifi() {
			if tetheringOn() {
				_ = disableTethering()
			}
			time.Sleep(2 * time.Second)
			continue
		}
		hasOtherIP := false
		for _, ip := range lanIPs() {
			if ip != wifiOpenAPIP {
				hasOtherIP = true
				break
			}
		}
		if clientSSID() != "" || hasOtherIP {
			if tetheringOn() {
				_ = disableTethering()
			}
			_ = os.Remove(wifiForceAPFlag)
			_ = os.Remove(wifiBootWaitFlag)
			time.Sleep(5 * time.Second)
			continue
		}
		if forceSetupAP() {
			_ = os.Remove(wifiBootWaitFlag)
			if tetheringOn() {
				time.Sleep(2 * time.Second)
				continue
			}
			ssid := robotName()
			_ = exec.Command(wifiSetupAPBin, "on", ssid).Run()
			time.Sleep(5 * time.Second)
			continue
		}
		// Saved home WiFi: wait for ConnMan to join. Raising the hotspot
		// kills wpa_supplicant, so a reboot would never reconnect.
		if hasSavedWifi() && time.Since(started) < wifiSavedGrace {
			_ = os.WriteFile(wifiBootWaitFlag, []byte("1\n"), 0644)
			if tetheringOn() {
				_ = disableTethering()
			}
			time.Sleep(5 * time.Second)
			continue
		}
		_ = os.Remove(wifiBootWaitFlag)
		if tetheringOn() {
			time.Sleep(2 * time.Second)
			continue
		}
		if nets, err := m.scanNetworks(); err == nil && len(nets) > 0 {
			saveWifiScanCache(nets)
		}
		ssid := robotName()
		_ = exec.Command(wifiSetupAPBin, "on", ssid).Run()
		time.Sleep(5 * time.Second)
	}
}

func preferBleWifi() bool {
	_, err := os.Stat(wifiPreferBleFlag)
	return err == nil
}

func forceSetupAP() bool {
	_, err := os.Stat(wifiForceAPFlag)
	return err == nil
}

func hasSavedWifi() bool {
	if _, err := os.Stat(wifiConnmanConfig); err == nil {
		return true
	}
	ents, err := os.ReadDir(wifiConnmanStateDir)
	if err != nil {
		return false
	}
	for _, e := range ents {
		if e.IsDir() && strings.HasPrefix(e.Name(), "wifi") {
			return true
		}
	}
	return false
}

func (m *WifiSetup) HTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case vars.IsEndpoint(r, "status"):
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(m.status())
	case vars.IsEndpoint(r, "scan"):
		nets, err := m.scanNetworks()
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status":   "error",
				"message":  err.Error(),
				"networks": []wifiNet{},
				"ap":       tetheringOn(),
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "success",
			"networks": nets,
			"ap":       tetheringOn(),
			"cached":   tetheringOn() && len(nets) > 0,
		})
	case vars.IsEndpoint(r, "connect"):
		ssid := strings.TrimSpace(r.FormValue("ssid"))
		pass := strings.TrimSpace(r.FormValue("password"))
		if pass == "" {
			pass = strings.TrimSpace(r.FormValue("pass"))
		}
		hidden := r.FormValue("hidden") == "1" || r.FormValue("hidden") == "true"
		fromHotspot := requestFromHotspot(r)
		wifiLog("connect from=%q host=%q route=%s ssid=%q", r.FormValue("from"), r.Host, uiSourceName(fromHotspot), ssid)
		if err := m.acceptConnect(ssid, pass, hidden, fromHotspot); err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "success",
			"phase":        "accepted",
			"message":      "accepted",
			"ui":           uiSourceName(fromHotspot),
			"from_hotspot": fromHotspot,
		})
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	case vars.IsEndpoint(r, "forget"):
		if r.Method != http.MethodPost {
			vars.HTTPError(w, r, "POST required")
			return
		}
		ssid := strings.TrimSpace(r.FormValue("ssid"))
		if err := m.forgetSavedWifi(ssid); err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "forgotten",
			"ssid":    ssid,
			"saved":   listSavedWifi(clientSSID()),
		})
	default:
		vars.HTTPError(w, r, "404 not found")
	}
}

func (m *WifiSetup) status() map[string]interface{} {
	ap := tetheringOn()
	ssid, _ := setupAPCreds()
	m.mu.Lock()
	phase := string(m.phase)
	if phase == "" {
		phase = string(wifiPhaseIdle)
	}
	lastErr := m.lastErr
	attempt := m.attemptSSID
	fromHotspot := m.fromHotspot
	ui := m.uiSource
	m.mu.Unlock()
	if lastErr == "" {
		if b, err := os.ReadFile(wifiLastErrorFile); err == nil {
			lastErr = strings.TrimSpace(string(b))
		}
	}
	if ui == "" {
		ui = uiSourceName(fromHotspot || ap)
	}
	cur := clientSSID()
	return map[string]interface{}{
		"ap":           ap,
		"ap_ssid":      ssid,
		"ap_password":  "",
		"ap_open":      true,
		"ap_ip":        setupAPIP(),
		"client_ssid":  cur,
		"saved":        listSavedWifi(cur),
		"ips":          lanIPs(),
		"phase":        phase,
		"last_error":   lastErr,
		"pending_ssid": attempt,
		"ui":           ui,
		"from_hotspot": fromHotspot || ap,
	}
}

func validateWifiCreds(ssid, pass string) error {
	ssid = strings.TrimSpace(ssid)
	if ssid == "" {
		return fmt.Errorf("SSID trống")
	}
	if strings.ContainsAny(ssid, "\n\r\"") || strings.ContainsAny(pass, "\n\r\"") {
		return fmt.Errorf("SSID/mật khẩu chứa ký tự không hợp lệ")
	}
	for _, r := range ssid + pass {
		if r == 0 || (!unicode.IsPrint(r) && !unicode.IsSpace(r)) {
			return fmt.Errorf("SSID/mật khẩu chứa ký tự không hợp lệ")
		}
	}
	if len(pass) > 0 && len(pass) < 8 {
		return fmt.Errorf("mật khẩu WiFi nhà phải từ 8 ký tự")
	}
	return nil
}

func (m *WifiSetup) acceptConnect(ssid, pass string, hidden bool, fromHotspot bool) error {
	ssid = strings.TrimSpace(ssid)
	if err := validateWifiCreds(ssid, pass); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.phase == wifiPhaseTrying {
		return fmt.Errorf("robot đang thử WiFi nhà, đợi xong rồi gửi lại")
	}
	m.fromHotspot = fromHotspot
	m.uiSource = uiSourceName(fromHotspot)
	m.restoreAP = fromHotspot
	if fromHotspot {
		m.prevSSID = hsClientSSID()
		m.prevSvc = hsServiceID(m.prevSSID)
	} else {
		m.prevSSID = lanClientSSID()
		m.prevSvc = lanServiceID(m.prevSSID)
	}
	m.phase = wifiPhaseAccepted
	m.lastErr = ""
	m.attemptSSID = ssid
	_ = os.Remove(wifiLastErrorFile)
	_ = writeWifiPending(ssid, hidden)
	creds := wifiCreds{SSID: ssid, Pass: pass, Hidden: hidden}
	prevSSID := m.prevSSID
	prevSvc := m.prevSvc
	if fromHotspot {
		go m.tryHomeWifiFromHotspot(creds, prevSSID, prevSvc)
	} else {
		go m.tryHomeWifiFromLAN(creds, prevSSID, prevSvc)
	}
	return nil
}

func wifiLog(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("WifiSetup: %s", msg)
	f, err := os.OpenFile(wifiTraceFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(f, "%s %s\n", time.Now().Format(time.RFC3339), msg)
	_ = f.Close()
}

func setWifiBusy(on bool) {
	if on {
		_ = os.WriteFile(wifiBusyFile, []byte("1\n"), 0644)
		return
	}
	_ = os.Remove(wifiBusyFile)
}

func setWifiHoldAP(on bool) {
	if on {
		_ = os.WriteFile(wifiHoldAPFlag, []byte("1\n"), 0644)
		return
	}
	_ = os.Remove(wifiHoldAPFlag)
}

func uiSourceName(fromHotspot bool) string {
	if fromHotspot {
		return "hotspot"
	}
	return "lan"
}

func requestFromHotspot(r *http.Request) bool {
	from := strings.ToLower(strings.TrimSpace(r.FormValue("from")))
	if from == "hotspot" {
		return true
	}
	if from == "lan" || from == "web" {
		return false
	}
	host := r.Host
	port := ""
	if i := strings.LastIndex(host, ":"); i >= 0 {
		port = host[i+1:]
		host = host[:i]
	}
	if port == "8080" {
		return false
	}
	if host == wifiOpenAPIP {
		return true
	}
	return tetheringOn()
}

func parseProvisionSSID(cfg []byte) string {
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

type savedWifiNet struct {
	SSID   string `json:"ssid"`
	Active bool   `json:"active"`
}

func splitConnmanServiceBlocks(txt string) []string {
	txt = strings.ReplaceAll(txt, "\r\n", "\n")
	parts := strings.Split(txt, "[service_")
	var blocks []string
	for i, p := range parts {
		if i == 0 {
			if strings.TrimSpace(p) != "" && parseProvisionSSID([]byte(p)) != "" {
				blocks = append(blocks, p)
			}
			continue
		}
		blocks = append(blocks, "[service_"+p)
	}
	return blocks
}

func parseProvisionSSIDs(cfg []byte) []string {
	var names []string
	seen := map[string]bool{}
	for _, block := range splitConnmanServiceBlocks(string(cfg)) {
		n := parseProvisionSSID([]byte(block))
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		names = append(names, n)
	}
	return names
}

func listSavedWifi(activeSSID string) []savedWifiNet {
	seen := map[string]bool{}
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		seen[name] = true
	}
	if b, err := os.ReadFile(wifiConnmanConfig); err == nil {
		for _, n := range parseProvisionSSIDs(b) {
			add(n)
		}
	}
	if ents, err := os.ReadDir(wifiConnmanStateDir); err == nil {
		for _, e := range ents {
			if !e.IsDir() || !strings.HasPrefix(e.Name(), "wifi") {
				continue
			}
			b, err := os.ReadFile(filepath.Join(wifiConnmanStateDir, e.Name(), "settings"))
			if err != nil {
				continue
			}
			add(parseProvisionSSID(b))
		}
	}
	if out, err := exec.Command("connmanctl", "services").CombinedOutput(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "*") {
				continue
			}
			name, _ := parseConnmanctlLine(line)
			add(name)
		}
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		ai, aj := names[i] == activeSSID, names[j] == activeSSID
		if ai != aj {
			return ai
		}
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})
	out := make([]savedWifiNet, 0, len(names))
	for _, n := range names {
		out = append(out, savedWifiNet{
			SSID:   n,
			Active: activeSSID != "" && n == activeSSID,
		})
	}
	return out
}

// Drop join flags left over from a crash or wired restart mid-join.
func clearStaleWifiJoinFlags() {
	if _, err := os.Stat(wifiPendingFile); err == nil {
		return
	}
	cur := clientSSID()
	ap := tetheringOn()
	if !wifiJoinInProgress() {
		return
	}
	if cur != "" && !ap {
		_ = os.Remove(wifiTryingFlag)
		setWifiBusy(false)
		setWifiHoldAP(false)
		return
	}
	if ap || cur == "" {
		_ = os.Remove(wifiTryingFlag)
		setWifiBusy(false)
		if ap {
			setWifiHoldAP(false)
		}
	}
}

func removeProvisionSSID(ssid string) error {
	b, err := os.ReadFile(wifiConnmanConfig)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var keep []string
	for _, block := range splitConnmanServiceBlocks(string(b)) {
		if parseProvisionSSID([]byte(block)) == ssid {
			continue
		}
		if s := strings.TrimSpace(block); s != "" {
			keep = append(keep, s)
		}
	}
	if len(keep) == 0 {
		_ = os.Remove(wifiConnmanConfig)
		return nil
	}
	return os.WriteFile(wifiConnmanConfig, []byte(strings.Join(keep, "\n\n")+"\n"), 0600)
}

func removeConnmanDirsNamed(ssid string) {
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
		if parseProvisionSSID(b) != ssid {
			continue
		}
		_ = os.RemoveAll(dir)
	}
}

func (m *WifiSetup) forgetSavedWifi(ssid string) error {
	ssid = strings.TrimSpace(ssid)
	if ssid == "" {
		return fmt.Errorf("SSID trống")
	}
	if err := validateWifiCreds(ssid, ""); err != nil {
		return err
	}
	if wifiJoinInProgress() {
		return fmt.Errorf("đang đổi WiFi, thử lại sau")
	}
	m.mu.Lock()
	phase := m.phase
	m.mu.Unlock()
	if phase == wifiPhaseTrying || phase == wifiPhaseAccepted {
		return fmt.Errorf("đang đổi WiFi, thử lại sau")
	}
	cur := clientSSID()
	if cur != "" && cur == ssid {
		return fmt.Errorf("không xóa mạng đang kết nối")
	}
	svc := connmanServiceID(ssid)
	if svc != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		_ = exec.CommandContext(ctx, "connmanctl", "disconnect", svc).Run()
		_ = exec.CommandContext(ctx, "connmanctl", "config", svc, "--remove").Run()
		cancel()
	}
	if err := removeProvisionSSID(ssid); err != nil {
		return err
	}
	removeConnmanDirsNamed(ssid)
	return nil
}

func setAutoConnectNamed(ssid string, on bool) {
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



func wifiJoinInProgress() bool {
	if _, err := os.Stat(wifiTryingFlag); err == nil {
		return true
	}
	if _, err := os.Stat(wifiPendingFile); err == nil {
		return true
	}
	// Hold AP suppress during LAN prev-wifi restore after trying flag cleared.
	if _, err := os.Stat(wifiHoldAPFlag); err == nil {
		return true
	}
	return false
}

func writeWifiPending(ssid string, hidden bool) error {
	b, err := json.Marshal(map[string]interface{}{
		"ssid":   ssid,
		"hidden": hidden,
		"phase":  "accepted",
	})
	if err != nil {
		return err
	}
	return os.WriteFile(wifiPendingFile, b, 0600)
}

func joinedHomeWifi(ssid string) bool {
	got := clientSSID()
	if got == ssid {
		return true
	}
	if tetheringOn() {
		return false
	}
	if got != "" && got != ssid {
		return false
	}
	for _, ip := range lanIPs() {
		if ip == wifiOpenAPIP || strings.HasPrefix(ip, "169.254.") {
			continue
		}
		return true
	}
	return false
}

func (m *WifiSetup) servePage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		ssid := strings.TrimSpace(r.FormValue("ssid"))
		pass := strings.TrimSpace(r.FormValue("pass"))
		if pass == "" {
			pass = strings.TrimSpace(r.FormValue("password"))
		}
		err := m.acceptConnect(ssid, pass, false, requestFromHotspot(r))
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(wifiPageHTML(m.status(), err.Error(), false)))
			return
		}
		http.Redirect(w, r, "/wifi?wifi=sent", http.StatusSeeOther)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	// Serve the SPA. Do not 302 to /#wifi — browsers omit the hash, so that loops.
	http.ServeFile(w, r, "/etc/wired/webroot/index.html")
}

func (m *WifiSetup) captiveProbe(w http.ResponseWriter, r *http.Request) {
	if tetheringOn() {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// WrapCaptive is a no-op pass-through. Do not HTTP-redirect "/" to a hash URL:
// Safari requests "/" again and reports "too many redirects".
func WrapCaptive(next http.Handler) http.Handler {
	return next
}

func tetheringOn() bool {
	if _, err := os.Stat(wifiSetupAPFlag); err == nil {
		return true
	}
	for _, ip := range lanIPs() {
		if ip == wifiOpenAPIP {
			return true
		}
	}
	return false
}

func disableTethering() error {
	_ = exec.Command(wifiSetupAPBin, "off").Run()
	_ = exec.Command("/anki/bin/vic-setup-ap", "off").Run()
	return nil
}

func connmanScan() error {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	err := exec.CommandContext(ctx, "busctl", "--system", "call",
		"net.connman", "/net/connman/technology/wifi",
		"net.connman.Technology", "Scan").Run()
	if err == nil {
		return nil
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel2()
	return exec.CommandContext(ctx2, "connmanctl", "scan", "wifi").Run()
}

func parseConnmanctlLine(line string) (name, svc string) {
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

func connmanServiceID(ssid string) string {
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
		name, svc := parseConnmanctlLine(line)
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

func connmanConnectService(svc string) error {
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

func connmanConnectNamed(ssid string) error {
	return connmanConnectService(connmanServiceID(ssid))
}

func connmanConnectPreferred(ssid, svc string) error {
	if svc != "" {
		if err := connmanConnectService(svc); err == nil {
			return nil
		}
	}
	if ssid == "" {
		return nil
	}
	if connmanServiceID(ssid) == "" {
		_ = connmanScan()
		time.Sleep(1500 * time.Millisecond)
	}
	return connmanConnectNamed(ssid)
}

func connmanDisconnectNamed(ssid string) error {
	svc := connmanServiceID(ssid)
	if svc == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "connmanctl", "disconnect", svc).Run()
}

func connmanSetAutoConnect(ssid string, on bool) error {
	svc := connmanServiceID(ssid)
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

func connmanServiceState(ssid string) string {
	svc := connmanServiceID(ssid)
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

func connmanServiceBlock(id string, creds wifiCreds, autoConnect bool) []byte {
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

func writeConnmanProvision(ssid, pass string, hidden bool) error {
	if err := os.MkdirAll(wifiConnmanStateDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(wifiConnmanConfig, connmanServiceBlock("wireos", wifiCreds{
		SSID: ssid, Pass: pass, Hidden: hidden,
	}, true), 0600)
}

// Keep the current home SSID provisioned while trying a new one from :8080.
func writeConnmanTryWithPrev(prev []byte, creds wifiCreds) error {
	if err := os.MkdirAll(wifiConnmanStateDir, 0755); err != nil {
		return err
	}
	prevTxt := strings.TrimSpace(string(prev))
	same := strings.Contains(prevTxt, "Name = "+creds.SSID+"\n") ||
		strings.Contains(prevTxt, "Name = "+creds.SSID+"\r") ||
		strings.Contains(prevTxt, "Name="+creds.SSID)
	if prevTxt == "" || same {
		return writeConnmanProvision(creds.SSID, creds.Pass, creds.Hidden)
	}
	body := prevTxt + "\n\n" + string(connmanServiceBlock("wireos_try", creds, true))
	return os.WriteFile(wifiConnmanConfig, []byte(body), 0600)
}

// Turn off AutoConnect on other saved networks so "change WiFi" actually switches.
func preferOnlySSID(ssid string) {
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

func setupAPCreds() (ssid, pass string) {
	ssid = robotName()
	pass = ""
	b, err := os.ReadFile(wifiSetupAPFlag)
	if err != nil {
		return ssid, pass
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	if len(lines) >= 1 && strings.TrimSpace(lines[0]) != "" {
		ssid = strings.TrimSpace(lines[0])
	}
	return ssid, pass
}

func robotName() string {
	out, err := exec.Command("/bin/getprop", "anki.robot.name").Output()
	name := strings.TrimSpace(string(out))
	if err != nil || name == "" {
		return "Vector"
	}
	return name
}

func setupAPIP() string {
	for _, ip := range lanIPs() {
		if ip == wifiOpenAPIP {
			return ip
		}
	}
	return wifiOpenAPIP
}

func clientSSID() string {
	if out, err := exec.Command("iwgetid", "-r").Output(); err == nil {
		if s := strings.TrimSpace(string(out)); s != "" {
			return s
		}
	}
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
		name, _ := parseConnmanctlLine(line)
		if name != "" {
			return name
		}
	}
	return ""
}

func lanIPs() []string {
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

type wifiNet struct {
	SSID   string `json:"ssid"`
	Signal int    `json:"signal"`
	Secure bool   `json:"secure"`
}

func (m *WifiSetup) scanNetworks() ([]wifiNet, error) {
	if tetheringOn() {
		return loadWifiScanCache(), nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var nets []wifiNet
	if out, err := exec.CommandContext(ctx, "iwlist", "wlan0", "scan").CombinedOutput(); err == nil {
		nets = mergeWifiNets(nets, parseIwlist(string(out)))
	}
	if len(nets) == 0 {
		if out, err := exec.CommandContext(ctx, "iw", "dev", "wlan0", "scan").CombinedOutput(); err == nil {
			nets = mergeWifiNets(nets, parseIwScan(string(out)))
		}
	}
	if len(nets) == 0 {
		_ = connmanScan()
		if out, err := exec.CommandContext(ctx, "connmanctl", "services").CombinedOutput(); err == nil {
			nets = mergeWifiNets(nets, parseConnmanctl(string(out)))
		}
	}
	sort.Slice(nets, func(i, j int) bool { return nets[i].Signal > nets[j].Signal })
	if len(nets) > 40 {
		nets = nets[:40]
	}
	if len(nets) > 0 {
		saveWifiScanCache(nets)
	}
	return nets, nil
}

func saveWifiScanCache(nets []wifiNet) {
	b, err := json.Marshal(nets)
	if err != nil {
		return
	}
	_ = os.WriteFile(wifiScanCache, b, 0644)
}

func loadWifiScanCache() []wifiNet {
	b, err := os.ReadFile(wifiScanCache)
	if err != nil {
		return nil
	}
	var nets []wifiNet
	if json.Unmarshal(b, &nets) != nil {
		return nil
	}
	return nets
}

func mergeWifiNets(dst, src []wifiNet) []wifiNet {
	by := map[string]wifiNet{}
	for _, n := range dst {
		if n.SSID == "" {
			continue
		}
		by[n.SSID] = n
	}
	for _, n := range src {
		if n.SSID == "" {
			continue
		}
		old, ok := by[n.SSID]
		if !ok || n.Signal > old.Signal {
			by[n.SSID] = n
		} else if n.Secure {
			old.Secure = true
			by[n.SSID] = old
		}
	}
	out := make([]wifiNet, 0, len(by))
	for _, n := range by {
		out = append(out, n)
	}
	return out
}

func unescapeIwSSID(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if i+3 < len(s) && s[i] == '\\' && (s[i+1] == 'x' || s[i+1] == 'X') {
			v, err := strconv.ParseUint(s[i+2:i+4], 16, 8)
			if err == nil {
				b.WriteByte(byte(v))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func parseIwlist(out string) []wifiNet {
	var nets []wifiNet
	var cur wifiNet
	freqOK := true
	started := false
	flush := func() {
		if started && cur.SSID != "" && freqOK {
			nets = append(nets, cur)
		}
	}
	for _, line := range strings.Split(out, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "Cell ") {
			flush()
			cur = wifiNet{}
			freqOK = true
			started = true
			continue
		}
		if strings.HasPrefix(t, "Frequency:") && strings.Contains(t, "5.") {
			freqOK = false
		}
		if strings.HasPrefix(t, "ESSID:") {
			s := strings.TrimPrefix(t, "ESSID:")
			s = strings.Trim(s, `"`)
			if s != "" && s != "<hidden>" {
				cur.SSID = unescapeIwSSID(s)
			}
		}
		if strings.HasPrefix(t, "Encryption key:") {
			cur.Secure = strings.Contains(t, "on")
		}
		if i := strings.Index(t, "Signal level="); i >= 0 {
			rest := t[i+len("Signal level="):]
			if j := strings.IndexAny(rest, " /"); j > 0 {
				if v, err := strconv.Atoi(rest[:j]); err == nil {
					if strings.Contains(rest, "dBm") || v < 0 {
						cur.Signal = dbmToPct(v)
					} else {
						cur.Signal = v
					}
				}
			} else if v, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(rest), "dBm")); err == nil {
				cur.Signal = dbmToPct(v)
			}
		}
		if i := strings.Index(t, "Quality="); i >= 0 && cur.Signal == 0 {
			rest := t[i+len("Quality="):]
			var a, b int
			if _, err := fmt.Sscanf(rest, "%d/%d", &a, &b); err == nil && b > 0 {
				cur.Signal = a * 100 / b
			}
		}
	}
	flush()
	return nets
}

func parseIwScan(out string) []wifiNet {
	var nets []wifiNet
	var cur wifiNet
	freqOK := true
	started := false
	flush := func() {
		if started && cur.SSID != "" && freqOK {
			nets = append(nets, cur)
		}
	}
	for _, line := range strings.Split(out, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "BSS ") {
			flush()
			cur = wifiNet{}
			freqOK = true
			started = true
			continue
		}
		if strings.HasPrefix(t, "freq:") {
			var f int
			fmt.Sscanf(strings.TrimPrefix(t, "freq:"), "%d", &f)
			if f >= 5000 {
				freqOK = false
			}
		}
		if strings.HasPrefix(t, "signal:") {
			var dbm float64
			fmt.Sscanf(strings.TrimPrefix(t, "signal:"), "%f", &dbm)
			cur.Signal = dbmToPct(int(dbm))
		}
		if strings.HasPrefix(t, "SSID:") {
			s := strings.TrimSpace(strings.TrimPrefix(t, "SSID:"))
			if s != "" {
				cur.SSID = s
			}
		}
		if strings.HasPrefix(t, "RSN:") || strings.HasPrefix(t, "WPA:") || strings.Contains(t, "Privacy") {
			cur.Secure = true
		}
	}
	flush()
	return nets
}

func parseConnmanctl(out string) []wifiNet {
	var nets []wifiNet
	for _, line := range strings.Split(out, "\n") {
		idx := strings.LastIndex(line, " wifi_")
		if idx < 0 {
			continue
		}
		path := strings.TrimSpace(line[idx+1:])
		name := strings.TrimSpace(line[:idx])
		name = strings.TrimLeft(name, "*AOR ")
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		secure := strings.Contains(path, "_psk") ||
			strings.Contains(path, "_ieee8021x") ||
			strings.Contains(path, "_wep")
		nets = append(nets, wifiNet{SSID: name, Signal: 50, Secure: secure})
	}
	return nets
}

func dbmToPct(dbm int) int {
	// Typical: -30 dBm ≈ 100, -90 dBm ≈ 0
	if dbm > 0 {
		return dbm
	}
	n := (dbm + 90) * 100 / 60
	if n < 0 {
		return 0
	}
	if n > 100 {
		return 100
	}
	return n
}

func wifiWaitHTML() string {
	return `<!doctype html><html lang="vi"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta http-equiv="refresh" content="2;url=/wifi?wifi=sent">
<title>Vector · Đã gửi WiFi</title>
<link rel="stylesheet" href="/style.css">
</head><body>
<div id="wifiToast" class="wifi-toast wifi-toast-ok" role="status">
<span class="wifi-toast-mark">✓</span>
<span>Đã gửi thành công!</span>
</div>
<div class="container" style="padding:24px">
<h2>Đã gửi thành công</h2>
<p>Robot đã nhận WiFi nhà. Hotspot sẽ tắt sau giây lát để thử nối (~30 giây).</p>
<p>Nếu <b>không</b> thấy hotspot trở lại: đã vào WiFi nhà — mở <code>http://&lt;IP-robot&gt;:8080/</code>.</p>
<p>Nếu hotspot bật lại: sai mật khẩu, vào lại trang này để sửa.</p>
</div></body></html>`
}

func wifiPageHTML(st map[string]interface{}, errMsg string, _ bool) string {
	ap, _ := st["ap"].(bool)
	apSSID, _ := st["ap_ssid"].(string)
	apIP, _ := st["ap_ip"].(string)
	cur, _ := st["client_ssid"].(string)
	mode := "Đã có mạng LAN"
	if ap {
		mode = "Hotspot cấu hình — nối WiFi nhà bên dưới"
	}
	banner := ""
	if errMsg != "" {
		banner = `<p class="help-warn">` + html.EscapeString(errMsg) + `</p>`
	} else if last, _ := st["last_error"].(string); last != "" {
		banner = `<p class="help-warn">` + html.EscapeString(last) + `</p>`
	}
	curEsc := html.EscapeString(cur)
	if curEsc == "" {
		curEsc = "(chưa nối)"
	}
	return `<!doctype html><html lang="vi"><head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Vector · WiFi</title>
<link rel="stylesheet" href="/style.css">
</head><body>
<div class="container" style="padding:20px;max-width:520px;margin:0 auto">
<h2>Vector · WiFi nhà</h2>
<p>` + html.EscapeString(mode) + ` · chỉ <b>2.4 GHz</b></p>
` + banner + `
<div class="help-box">
<p><b>Hotspot robot</b></p>
<p>SSID: <code>` + html.EscapeString(apSSID) + `</code><br>
Mật khẩu hotspot: <b>không — mạng mở, bấm Kết nối</b><br>
Trang này: <code>http://` + html.EscapeString(apIP) + `/wifi</code></p>
<p>WiFi hiện tại: <code>` + curEsc + `</code></p>
</div>
<p id="wifiScanHint">Bấm quét, chọn mạng, rồi nhập mật khẩu.</p>
<p><button type="button" id="wifiScanBtn" onclick="wifiScanPortal()">Quét mạng</button></p>
<div id="wifiNetworkList" class="wifi-net-list"></div>
<form method="post" action="/wifi" onsubmit="return checkWifi()">
<p><label>Tên WiFi nhà (SSID) — hoặc gõ tay nếu mạng ẩn<br>
<input name="ssid" id="ssid" required autocomplete="off" autocapitalize="off" spellcheck="false" style="width:100%;min-height:44px"></label></p>
<p><label>Mật khẩu WiFi nhà<br>
<input name="pass" id="pass" type="password" minlength="8" style="width:100%;min-height:44px"></label></p>
<p id="wifiFormMsg" class="wifi-banner wifi-banner-ok" style="display:none"></p>
<p><button type="submit">Áp dụng mạng</button></p>
</form>
<p style="font-size:13px;color:#888">ConnMan tethering: một radio — đang hotspot thì quét có thể trống, hãy gõ tay SSID. Sau khi áp dụng, hotspot tắt.</p>
</div>
<script>
var wifiOpen=false;
function wifiScanPortal(){
  var list=document.getElementById('wifiNetworkList');
  var hint=document.getElementById('wifiScanHint');
  var btn=document.getElementById('wifiScanBtn');
  if(btn) btn.disabled=true;
  if(hint) hint.textContent='Đang quét mạng 2.4 GHz…';
  fetch('/api/mods/WifiSetup/scan').then(function(r){return r.json()}).then(function(j){
    var nets=j.networks||[];
    list.innerHTML='';
    if(!nets.length){
      hint.textContent=j.ap?'Đang hotspot — quét có thể trống. Gõ tay SSID bên dưới.':'Không thấy mạng. Gõ tay SSID.';
      return;
    }
    hint.textContent='Chọn mạng rồi nhập mật khẩu.';
    nets.forEach(function(n){
      var b=document.createElement('button');
      b.type='button';
      b.className='wifi-net';
      b.innerHTML='<span>'+escapeHtml(n.ssid)+'</span><span class="wifi-net-meta">'+(n.secure?'🔒 ':'mở ')+(n.signal||0)+'%</span>';
      b.onclick=function(){
        document.querySelectorAll('.wifi-net').forEach(function(x){x.classList.remove('selected')});
        b.classList.add('selected');
        document.getElementById('ssid').value=n.ssid;
        wifiOpen=!n.secure;
        var p=document.getElementById('pass');
        if(wifiOpen){p.removeAttribute('minlength');p.placeholder='Mạng mở — để trống';}
        else{p.setAttribute('minlength','8');p.placeholder='';p.focus();}
      };
      list.appendChild(b);
    });
  }).catch(function(){
    if(hint) hint.textContent='Không quét được. Gõ tay SSID.';
  }).then(function(){ if(btn) btn.disabled=false; });
}
function escapeHtml(s){return String(s).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c];});}
var wifiFormArmed=false;
function checkWifi(){
  var p=document.getElementById('pass');
  if(!wifiOpen && p.value.length<8){alert('Mật khẩu WiFi nhà phải từ 8 ký tự');return false;}
  if(wifiFormArmed) return true;
  var msg=document.getElementById('wifiFormMsg');
  if(msg){ msg.style.display='block'; msg.className='wifi-banner wifi-banner-wait'; msg.textContent='Đang gửi WiFi cho robot…'; }
  var btn=document.querySelector('form button[type="submit"]');
  if(btn){ btn.disabled=true; btn.textContent='Đang gửi…'; }
  wifiFormArmed=true;
  requestAnimationFrame(function(){
    requestAnimationFrame(function(){
      document.querySelector('form').submit();
    });
  });
  return false;
}
wifiScanPortal();
</script>
</body></html>`
}
