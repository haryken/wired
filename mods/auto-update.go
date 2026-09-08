package mods

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	case vars.IsEndpoint(r, "startFromURL"):
		m.startFromURL(w, r)
	case vars.IsEndpoint(r, "status"):
		m.writeStatus(w)
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
		vars.HTTPError(w, r, "cannot write update-engine.env: "+err.Error())
		return
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
		vars.HTTPError(w, r, "failed to start update-engine: "+msg)
		return
	}
	otaLogAppend("update-engine restarted")
	vars.HTTPSuccess(w, r)
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
