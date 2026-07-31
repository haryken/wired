package mods

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/os-vector/wired/vars"
)

// JournalLogs streams allowlisted systemd journal units to the :8080 UI.
type JournalLogs struct {
	vars.Modification
}

type journalUnitInfo struct {
	Unit   string `json:"unit"`
	Label  string `json:"label"`
	Active string `json:"active"`
}

var journalAllowlist = []struct {
	Unit  string
	Label string
}{
	{"vic-cloud", "vic-cloud"},
	{"vic-anim", "vic-anim"},
	{"vic-engine", "vic-engine"},
	{"vic-robot", "vic-robot"},
	{"vic-switchboard", "vic-switchboard"},
	{"wired", "wired"},
	{"update-engine", "update-engine"},
}

func NewJournalLogs() *JournalLogs {
	return &JournalLogs{}
}

func (m *JournalLogs) Name() string {
	return "JournalLogs"
}

func (m *JournalLogs) Description() string {
	return "Live journalctl logs for robot services"
}

func (m *JournalLogs) Load() error {
	return nil
}

func journalUnitAllowed(unit string) bool {
	unit = strings.TrimSpace(unit)
	for _, u := range journalAllowlist {
		if u.Unit == unit {
			return true
		}
	}
	return false
}

func (m *JournalLogs) HTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/mods/"+m.Name()) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/mods/"+m.Name()+"/")
	switch path {
	case "units":
		m.handleUnits(w, r)
	case "tail":
		m.handleTail(w, r)
	case "follow":
		m.handleFollow(w, r)
	default:
		vars.HTTPError(w, r, "not found")
	}
}

func (m *JournalLogs) handleUnits(w http.ResponseWriter, r *http.Request) {
	out := make([]journalUnitInfo, 0, len(journalAllowlist))
	for _, u := range journalAllowlist {
		out = append(out, journalUnitInfo{
			Unit:   u.Unit,
			Label:  u.Label,
			Active: unitActive(u.Unit + ".service"),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (m *JournalLogs) handleTail(w http.ResponseWriter, r *http.Request) {
	unit := strings.TrimSpace(r.URL.Query().Get("unit"))
	if !journalUnitAllowed(unit) {
		vars.HTTPError(w, r, "unit not allowed")
		return
	}
	n := 200
	if s := r.URL.Query().Get("n"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			n = v
		}
	}
	if n > 500 {
		n = 500
	}
	cmd := exec.Command("/bin/journalctl", "-u", unit, "-n", strconv.Itoa(n), "--no-pager", "-o", "short-iso")
	out, err := cmd.CombinedOutput()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if err != nil && len(out) == 0 {
		http.Error(w, "journalctl failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(out)
}

func (m *JournalLogs) handleFollow(w http.ResponseWriter, r *http.Request) {
	unit := strings.TrimSpace(r.URL.Query().Get("unit"))
	if !journalUnitAllowed(unit) {
		vars.HTTPError(w, r, "unit not allowed")
		return
	}
	n := 200
	if s := r.URL.Query().Get("n"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			n = v
		}
	}
	if n > 500 {
		n = 500
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	cmd := exec.CommandContext(r.Context(),
		"/bin/journalctl",
		"-u", unit,
		"-n", strconv.Itoa(n),
		"-f",
		"--no-pager",
		"-o", "short-iso",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		writeSSE(w, flusher, "logerr", "stdout pipe: "+err.Error())
		return
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		writeSSE(w, flusher, "logerr", "start journalctl: "+err.Error())
		return
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	type sseMsg struct {
		event string
		data  string
	}
	msgs := make(chan sseMsg, 64)
	go func() {
		defer close(msgs)
		sc := bufio.NewScanner(stdout)
		buf := make([]byte, 0, 64*1024)
		sc.Buffer(buf, 1024*1024)
		for sc.Scan() {
			select {
			case <-r.Context().Done():
				return
			case msgs <- sseMsg{event: "log", data: sc.Text()}:
			}
		}
		if err := sc.Err(); err != nil && r.Context().Err() == nil {
			select {
			case <-r.Context().Done():
			case msgs <- sseMsg{event: "logerr", data: err.Error()}:
			}
		}
	}()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-msgs:
			if !ok {
				return
			}
			writeSSE(w, flusher, msg.event, msg.data)
		case <-ticker.C:
			_, _ = w.Write([]byte(": ping\n\n"))
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event, data string) {
	if event != "" {
		_, _ = w.Write([]byte("event: " + event + "\n"))
	}
	// SSE: one data: line; escape newlines into multiple data fields.
	for _, line := range strings.Split(data, "\n") {
		_, _ = w.Write([]byte("data: " + line + "\n"))
	}
	_, _ = w.Write([]byte("\n"))
	flusher.Flush()
}
