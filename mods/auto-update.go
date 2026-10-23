package mods

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/os-vector/wired/vars"
)

const (
	userInhibitPath      = "/data/data/user-do-not-auto-update"
	selfMadeBuildPath    = "/etc/do-not-auto-update"
	updateEngineEnvPath  = "/run/vic-switchboard/update-engine.env"
	updateEngineStateDir = "/run/update-engine"
	disableUpdateEngine  = "/run/vic-switchboard/disable-update-engine"
	otaLogPath           = "/data/wired/mods/AutoUpdate/ota.log"
)

// AutoUpdate handles URL-based OS OTA (and legacy auto-update inhibit toggles).
type AutoUpdate struct {
	vars.Modification
}

func NewAutoUpdate() *AutoUpdate {
	return &AutoUpdate{}
}

func (modu *AutoUpdate) Name() string {
	return "AutoUpdate"
}

func (modu *AutoUpdate) Description() string {
	return "OS update from URL and auto-update inhibit toggles."
}

func (modu *AutoUpdate) Load() error {
	_ = os.MkdirAll(filepath.Dir(otaLogPath), 0777)
	cleanupOtaUpload()
	return nil
}

func (m *AutoUpdate) HTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case vars.IsEndpoint(r, "isSelfMadeBuild"):
		if _, err := os.Stat(selfMadeBuildPath); err == nil {
			fmt.Fprint(w, "true")
		} else {
			fmt.Fprint(w, "false")
		}
	case vars.IsEndpoint(r, "isInhibitedByUser"):
		if _, err := os.Stat(userInhibitPath); err == nil {
			fmt.Fprint(w, "true")
		} else {
			fmt.Fprint(w, "false")
		}
	case vars.IsEndpoint(r, "setInhibited"):
		_ = os.WriteFile(userInhibitPath, []byte("true"), 0777)
		vars.HTTPSuccess(w, r)
	case vars.IsEndpoint(r, "setAllowed"):
		_ = os.Remove(userInhibitPath)
		vars.HTTPSuccess(w, r)
	case vars.IsEndpoint(r, "localOta"):
		serveLocalOta(w, r)
	case vars.IsEndpoint(r, "uploadOta"):
		m.uploadOta(w, r)
	case vars.IsEndpoint(r, "startFromURL"):
		m.startFromURL(w, r)
	case vars.IsEndpoint(r, "status"):
		m.writeStatus(w)
	case vars.IsEndpoint(r, "checkOfficial"):
		res := checkOfficialOta(r.FormValue("force") == "1")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(res)
	case vars.IsEndpoint(r, "log"):
		data, err := os.ReadFile(otaLogPath)
		if err != nil {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "(no ota.log yet)\n")
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(data)
	case vars.IsEndpoint(r, "reboot"):
		vars.HTTPSuccess(w, r)
		go func() {
			time.Sleep(800 * time.Millisecond)
			_ = exec.Command("/sbin/reboot").Start()
		}()
	default:
		vars.HTTPError(w, r, "404 not found")
	}
}

func (m *AutoUpdate) startFromURL(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimSpace(r.FormValue("url"))
	if raw == "" {
		vars.HTTPError(w, r, "url required")
		return
	}
	if err := validateOtaURL(raw); err != nil {
		vars.HTTPError(w, r, err.Error())
		return
	}
	if err := probeOtaURL(raw); err != nil {
		vars.HTTPError(w, r, err.Error())
		return
	}

	otaLogAppend(fmt.Sprintf("startFromURL url=%s", raw))
	if err := beginOta(raw); err != nil {
		vars.HTTPError(w, r, err.Error())
		return
	}
	vars.HTTPSuccess(w, r)
}

// beginOta hands url to update-engine, which streams it with curl.
func beginOta(raw string) error {
	// Clear switchboard guard so update-engine.service can start.
	_ = os.Remove(disableUpdateEngine)
	// Drop stale status from a prior inhibited auto-update so the UI does not
	// flash "auto-update inhibited" while the URL OTA is starting.
	_ = os.RemoveAll(updateEngineStateDir)

	// Drop robot stack first. Stopping anki-robot/vic-switchboard clears
	// /run/vic-switchboard, so the URL env must be written AFTER this stop.
	_ = exec.Command("/bin/systemctl", "stop", "anki-robot.target").Run()

	_ = os.MkdirAll(filepath.Dir(updateEngineEnvPath), 0777)
	env := strings.Join([]string{
		"UPDATE_ENGINE_ENABLED=True",
		"UPDATE_ENGINE_ALLOW_DOWNGRADE=True",
		"UPDATE_ENGINE_MAX_SLEEP=1",
		"UPDATE_ENGINE_DEBUG=True",
		"UPDATE_ENGINE_URL=" + raw,
		"",
	}, "\n")
	if err := os.WriteFile(updateEngineEnvPath, []byte(env), 0644); err != nil {
		return fmt.Errorf("cannot write update-engine.env: %v", err)
	}
	// Also seed oneshot.env — systemd reads it before vic-switchboard env.
	_ = os.WriteFile("/run/update-engine-oneshot.env", []byte(env), 0644)

	// Restart update-engine with our URL override (--no-block so the HTTP
	// handler does not sit on the long start-pre / download).
	_ = exec.Command("/bin/systemctl", "reset-failed", "update-engine.service").Run()
	if out, err := exec.Command("/bin/systemctl", "restart", "--no-block", "update-engine.service").CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		otaLogAppend("restart update-engine failed: " + msg)
		return fmt.Errorf("failed to start update-engine: %s", msg)
	}
	otaLogAppend("update-engine restarted")
	return nil
}

const (
	// update-engine appends ?emresn=... to the URL, which rules out file://.
	otaLocalURL = "http://127.0.0.1:8080/api/mods/AutoUpdate/localOta/upload.ota"
	// A full WireOS .ota is ~260 MB; /data is ~700 MB.
	otaUploadMax     = 600 << 20
	otaUploadReserve = 64 << 20
)

var (
	otaUploadMu   sync.Mutex
	otaUploadBusy bool
	// Overridable in tests.
	otaUploadDir   = "/data/wired/ota-upload"
	otaFreeBytesFn = freeBytes
)

func otaUploadFile() string {
	return filepath.Join(otaUploadDir, "upload.ota")
}

func (m *AutoUpdate) uploadOta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		vars.HTTPError(w, r, "POST the .ota file as the request body")
		return
	}
	otaUploadMu.Lock()
	if otaUploadBusy {
		otaUploadMu.Unlock()
		vars.HTTPError(w, r, "another upload is in progress")
		return
	}
	otaUploadBusy = true
	otaUploadMu.Unlock()
	defer func() {
		otaUploadMu.Lock()
		otaUploadBusy = false
		otaUploadMu.Unlock()
	}()

	if a := unitActive("update-engine.service"); a == "active" || a == "activating" {
		vars.HTTPError(w, r, "an update is already running")
		return
	}
	version, err := receiveOtaUpload(r.Body, r.ContentLength)
	if err != nil {
		otaLogAppend("uploadOta rejected: " + err.Error())
		vars.HTTPError(w, r, err.Error())
		return
	}
	otaLogAppend(fmt.Sprintf("uploadOta stored %d bytes, update_version=%s", r.ContentLength, version))
	if err := beginOta(otaLocalURL); err != nil {
		_ = os.RemoveAll(otaUploadDir)
		vars.HTTPError(w, r, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "update_version": version})
}

// receiveOtaUpload stores body as otaUploadFile and returns its manifest
// update_version. Nothing is left on disk when it fails.
func receiveOtaUpload(body io.Reader, size int64) (string, error) {
	if size <= 0 {
		return "", fmt.Errorf("file size unknown")
	}
	if size > otaUploadMax {
		return "", fmt.Errorf("file too large (%d MB, max %d MB)", size>>20, otaUploadMax>>20)
	}
	_ = os.RemoveAll(otaUploadDir)
	if err := os.MkdirAll(otaUploadDir, 0700); err != nil {
		return "", fmt.Errorf("cannot create upload dir: %v", err)
	}
	if free := otaFreeBytesFn(otaUploadDir); free < size+otaUploadReserve {
		_ = os.RemoveAll(otaUploadDir)
		return "", fmt.Errorf("not enough space on robot: need %d MB, free %d MB", (size+otaUploadReserve)>>20, free>>20)
	}
	part := otaUploadFile() + ".part"
	f, err := os.OpenFile(part, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		_ = os.RemoveAll(otaUploadDir)
		return "", fmt.Errorf("cannot write upload: %v", err)
	}
	n, err := io.Copy(f, io.LimitReader(body, size))
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil || n != size {
		_ = os.RemoveAll(otaUploadDir)
		return "", fmt.Errorf("upload interrupted (%d/%d bytes)", n, size)
	}
	version, err := otaFileVersion(part)
	if err != nil {
		_ = os.RemoveAll(otaUploadDir)
		return "", err
	}
	if err := os.Rename(part, otaUploadFile()); err != nil {
		_ = os.RemoveAll(otaUploadDir)
		return "", fmt.Errorf("cannot store upload: %v", err)
	}
	return version, nil
}

func otaFileVersion(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	head := make([]byte, otaHeadBytes)
	n, _ := io.ReadFull(f, head)
	version := otaManifestVersion(head[:n])
	if version == "" {
		return "", fmt.Errorf("not a WireOS .ota file (manifest.ini with update_version missing)")
	}
	return version, nil
}

func freeBytes(path string) int64 {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0
	}
	return int64(st.Bavail) * int64(st.Bsize)
}

// serveLocalOta feeds the uploaded file to update-engine on this robot only.
func serveLocalOta(w http.ResponseWriter, r *http.Request) {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	http.ServeFile(w, r, otaUploadFile())
}

// cleanupOtaUpload drops a leftover upload unless update-engine is reading it.
func cleanupOtaUpload() {
	if a := unitActive("update-engine.service"); a == "active" || a == "activating" {
		return
	}
	if _, err := os.Stat(otaUploadDir); err == nil {
		_ = os.RemoveAll(otaUploadDir)
	}
}

func validateOtaURL(raw string) error {
	if strings.ContainsAny(raw, " \t\r\n") {
		return fmt.Errorf("url contains invalid characters")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url must be http or https")
	}
	if u.Host == "" {
		return fmt.Errorf("url host required")
	}
	if u.User != nil {
		return fmt.Errorf("url must not contain userinfo")
	}
	base := strings.ToLower(filepath.Base(u.Path))
	if !strings.HasSuffix(base, ".ota") {
		return fmt.Errorf("url path must end with .ota")
	}
	return nil
}

func probeOtaURL(raw string) error {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodHead, raw, nil)
	if err != nil {
		return fmt.Errorf("cannot check url")
	}
	resp, err := client.Do(req)
	if err != nil {
		// Some hosts reject HEAD — fall back to ranged GET.
		req2, err2 := http.NewRequest(http.MethodGet, raw, nil)
		if err2 != nil {
			return fmt.Errorf("url not reachable")
		}
		req2.Header.Set("Range", "bytes=0-0")
		resp, err = client.Do(req2)
		if err != nil {
			return fmt.Errorf("url not reachable")
		}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 64))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("url not reachable (HTTP %d)", resp.StatusCode)
	}
	return nil
}

// The release asset names never change; a new build replaces the file.
// Checked in order: the first link that has the file is offered.
var officialOtaURLs = []string{
	"https://github.com/haryken/wirepodxiaozhi/releases/download/robot/vector.ota",
	"https://github.com/haryken/wire-os/releases/download/robot/vector.ota",
}

const (
	officialOtaCacheTTL = 10 * time.Minute
	// manifest.ini is the first tar member and only a few hundred bytes.
	otaHeadBytes = 4096
)

type officialOtaCheck struct {
	URL            string `json:"url"`
	Reachable      bool   `json:"reachable"`
	Exists         bool   `json:"exists"`
	HTTPStatus     int    `json:"http_status"`
	UpdateVersion  string `json:"update_version"`
	CurrentVersion string `json:"current_version"`
	// Newer is true when the file exists and its update_version differs from
	// the robot (or the version cannot be read). Same version = no offer.
	Newer     bool   `json:"newer"`
	Same      bool   `json:"same"`
	Size      int64  `json:"size"`
	CheckedAt string `json:"checked_at"`
	Error     string `json:"error,omitempty"`
}

var (
	officialOtaMu     sync.Mutex
	officialOtaCached *officialOtaCheck
	officialOtaTime   time.Time
	// Overridable in tests.
	officialOtaFetch     = fetchOtaHead
	currentAnkiVersionFn = currentAnkiVersion
)

func checkOfficialOta(force bool) officialOtaCheck {
	officialOtaMu.Lock()
	defer officialOtaMu.Unlock()
	if !force && officialOtaCached != nil && time.Since(officialOtaTime) < officialOtaCacheTTL {
		return *officialOtaCached
	}
	res := officialOtaCheck{
		URL:            officialOtaURLs[0],
		CurrentVersion: currentAnkiVersionFn(),
		CheckedAt:      time.Now().Format(time.RFC3339),
	}
	for _, u := range officialOtaURLs {
		status, total, head, err := officialOtaFetch(u)
		if err != nil {
			res.Error = err.Error()
			continue
		}
		res.Reachable = true
		res.HTTPStatus = status
		if status == http.StatusOK || status == http.StatusPartialContent {
			res.URL = u
			res.Exists = true
			res.Error = ""
			res.Size = total
			res.UpdateVersion = otaManifestVersion(head)
			res.Same = res.UpdateVersion != "" && compareOSVersions(res.UpdateVersion, res.CurrentVersion) == 0
			res.Newer = !res.Same
			break
		}
	}
	if res.Reachable && !res.Exists {
		res.Error = ""
	}
	officialOtaCached = &res
	officialOtaTime = time.Now()
	return res
}

func fetchOtaHead(raw string) (status int, total int64, head []byte, err error) {
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest(http.MethodGet, raw, nil)
	if err != nil {
		return 0, 0, nil, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=0-%d", otaHeadBytes-1))
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, nil, err
	}
	defer resp.Body.Close()
	head, _ = io.ReadAll(io.LimitReader(resp.Body, otaHeadBytes))
	total = resp.ContentLength
	if cr := resp.Header.Get("Content-Range"); cr != "" {
		if i := strings.LastIndex(cr, "/"); i >= 0 {
			if n, err := strconv.ParseInt(cr[i+1:], 10, 64); err == nil {
				total = n
			}
		}
	}
	return resp.StatusCode, total, head, nil
}

// otaManifestVersion reads update_version from manifest.ini, the first member
// of the .ota tar archive.
func otaManifestVersion(head []byte) string {
	if len(head) < 512 {
		return ""
	}
	name := strings.TrimRight(string(head[0:100]), "\x00")
	if filepath.Base(name) != "manifest.ini" {
		return ""
	}
	size, err := strconv.ParseInt(strings.Trim(string(head[124:136]), "\x00 "), 8, 64)
	if err != nil || size <= 0 {
		return ""
	}
	end := 512 + int(size)
	if end > len(head) {
		end = len(head)
	}
	for _, line := range strings.Split(string(head[512:end]), "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if ok && strings.EqualFold(strings.TrimSpace(k), "update_version") {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// compareOSVersions compares dotted versions like 3.0.1.999d numerically per
// part, then by any suffix letters.
func compareOSVersions(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var sa, sb string
		if i < len(pa) {
			sa = pa[i]
		}
		if i < len(pb) {
			sb = pb[i]
		}
		na, ra := splitVersionPart(sa)
		nb, rb := splitVersionPart(sb)
		if na != nb {
			if na > nb {
				return 1
			}
			return -1
		}
		if c := strings.Compare(ra, rb); c != 0 {
			return c
		}
	}
	return 0
}

func splitVersionPart(s string) (int, string) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	n, _ := strconv.Atoi(s[:i])
	return n, s[i:]
}

type otaStatus struct {
	Phase                string `json:"phase"`
	Percent              int    `json:"percent"`
	UpdateVersion        string `json:"update_version,omitempty"`
	CurrentVersion       string `json:"current_version,omitempty"`
	UnitActive           string `json:"unit_active,omitempty"`
	Progress             int64  `json:"progress,omitempty"`
	Expected             int64  `json:"expected,omitempty"`
	ExpectedDownloadSize int64  `json:"expected_download_size,omitempty"`
	Error                string `json:"error,omitempty"`
	Done                 bool   `json:"done"`
	Journal              string `json:"journal,omitempty"`
}

func (m *AutoUpdate) writeStatus(w http.ResponseWriter) {
	st := otaStatus{
		Phase:                readTrim(filepath.Join(updateEngineStateDir, "phase")),
		CurrentVersion:       currentAnkiVersion(),
		UnitActive:           unitActive("update-engine.service"),
		Done:                 fileNonEmpty(filepath.Join(updateEngineStateDir, "done")),
		Progress:             readInt64(filepath.Join(updateEngineStateDir, "progress")),
		Expected:             readInt64(filepath.Join(updateEngineStateDir, "expected-size")),
		ExpectedDownloadSize: readInt64(filepath.Join(updateEngineStateDir, "expected-download-size")),
		UpdateVersion:        readManifestVersion(filepath.Join(updateEngineStateDir, "manifest.ini")),
		Error:                readTrim(filepath.Join(updateEngineStateDir, "error")),
		Journal:              journalTail("update-engine.service", 40),
	}
	if st.Phase == "" {
		st.Phase = "waiting"
	}
	// "Unclean exit" is the service pre-seed; ignore until a real phase progresses.
	if st.Error == "Unclean exit" && !st.Done && st.Progress == 0 {
		st.Error = ""
	}
	// Leftover from blocked background auto-update; URL OTAs set UPDATE_ENGINE_URL
	// and never take the "auto" inhibit path — do not surface as a failed manual update.
	if st.Error == "auto-update inhibited" && !st.Done && st.Progress == 0 {
		st.Error = ""
		if st.Phase == "starting" || st.Phase == "waiting" {
			st.Phase = "waiting"
		}
	}
	// If exit_code exists and non-zero, prefer mapping.
	if code := readTrim(filepath.Join(updateEngineStateDir, "exit_code")); code != "" && code != "0" {
		if st.Error == "" {
			st.Error = "update-engine exit_code=" + code
		}
	}
	if st.Error != "" && st.UnitActive != "active" && st.UnitActive != "activating" {
		cleanupOtaUpload()
	}
	if st.Expected > 0 {
		st.Percent = int((st.Progress * 100) / st.Expected)
		if st.Percent > 100 {
			st.Percent = 100
		}
		if st.Percent < 0 {
			st.Percent = 0
		}
	} else if st.Done {
		st.Percent = 100
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(st)
}

func currentAnkiVersion() string {
	out, err := exec.Command("/bin/getprop", "ro.anki.version").Output()
	if err == nil {
		v := strings.TrimSpace(string(out))
		if v != "" {
			return v
		}
	}
	b, err := os.ReadFile("/anki/etc/version")
	if err == nil {
		return strings.TrimSpace(string(b))
	}
	return ""
}

func unitActive(unit string) string {
	out, err := exec.Command("/bin/systemctl", "is-active", unit).Output()
	if err != nil {
		return strings.TrimSpace(string(out))
	}
	return strings.TrimSpace(string(out))
}

func journalTail(unit string, n int) string {
	out, err := exec.Command("/bin/journalctl", "-u", unit, "-n", strconv.Itoa(n), "--no-pager").CombinedOutput()
	if err != nil {
		return ""
	}
	return string(out)
}

func readManifestVersion(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		low := strings.ToLower(line)
		if strings.HasPrefix(low, "version=") || strings.HasPrefix(low, "update_version=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func readTrim(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func readInt64(path string) int64 {
	s := readTrim(path)
	if s == "" {
		return 0
	}
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func fileNonEmpty(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Size() > 0
}

func otaLogAppend(line string) {
	_ = os.MkdirAll(filepath.Dir(otaLogPath), 0777)
	f, err := os.OpenFile(otaLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	ts := time.Now().Format(time.RFC3339)
	fmt.Fprintf(f, "%s %s\n", ts, line)
}
