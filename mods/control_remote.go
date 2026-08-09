package mods

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	remoteShareCookie   = "wired_share"
	remoteProxyAddr     = "127.0.0.1:18765"
	remoteDefaultTTL    = 2 * time.Hour
	remoteCloudflaredVer = "2025.2.1"
)

var tryCloudflareURL = regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)

type remoteShareStatus struct {
	Enabled   bool   `json:"enabled"`
	URL       string `json:"url,omitempty"`
	Phase     string `json:"phase"` // idle | downloading | starting | ready | error
	Error     string `json:"error,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type remoteShare struct {
	mu        sync.Mutex
	enabled   bool
	phase     string
	errMsg    string
	token     string
	publicURL string // https://xxx.trycloudflare.com
	shareURL  string // public + /r/token/#control
	expires   time.Time
	proxySrv  *http.Server
	cfCmd     *exec.Cmd
	cfCancel  context.CancelFunc
	expireT   *time.Timer
}

var ctrlRemote = &remoteShare{phase: "idle"}

func remoteJSON(w http.ResponseWriter, code int, st remoteShareStatus) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(st)
}

func (rs *remoteShare) snapshot() remoteShareStatus {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	st := remoteShareStatus{
		Enabled: rs.enabled,
		URL:     rs.shareURL,
		Phase:   rs.phase,
		Error:   rs.errMsg,
	}
	if !rs.expires.IsZero() {
		st.ExpiresAt = rs.expires.UTC().Format(time.RFC3339)
	}
	return st
}

func (rs *remoteShare) setPhase(phase, errMsg string) {
	rs.mu.Lock()
	rs.phase = phase
	rs.errMsg = errMsg
	rs.mu.Unlock()
}

func enableRemoteShare() remoteShareStatus {
	rs := ctrlRemote
	rs.mu.Lock()
	if rs.enabled && rs.shareURL != "" && rs.phase == "ready" {
		st := remoteShareStatus{
			Enabled:   true,
			URL:       rs.shareURL,
			Phase:     "ready",
			ExpiresAt: rs.expires.UTC().Format(time.RFC3339),
		}
		rs.mu.Unlock()
		return st
	}
	if rs.phase == "downloading" || rs.phase == "starting" {
		st := remoteShareStatus{Enabled: true, Phase: rs.phase, Error: rs.errMsg}
		rs.mu.Unlock()
		return st
	}
	rs.enabled = true
	rs.phase = "starting"
	rs.errMsg = ""
	rs.shareURL = ""
	rs.publicURL = ""
	rs.mu.Unlock()

	go rs.start()
	return rs.snapshot()
}

func disableRemoteShare() remoteShareStatus {
	ctrlRemote.stop("idle")
	return ctrlRemote.snapshot()
}

func (rs *remoteShare) stop(finalPhase string) {
	rs.mu.Lock()
	if rs.expireT != nil {
		rs.expireT.Stop()
		rs.expireT = nil
	}
	cancel := rs.cfCancel
	cmd := rs.cfCmd
	srv := rs.proxySrv
	rs.cfCancel = nil
	rs.cfCmd = nil
	rs.proxySrv = nil
	rs.enabled = false
	rs.phase = finalPhase
	if finalPhase != "error" {
		rs.errMsg = ""
	}
	rs.shareURL = ""
	rs.publicURL = ""
	rs.token = ""
	rs.expires = time.Time{}
	rs.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	if srv != nil {
		ctx, c := context.WithTimeout(context.Background(), 2*time.Second)
		_ = srv.Shutdown(ctx)
		c()
	}
}

func (rs *remoteShare) start() {
	token, err := randomHex(16)
	if err != nil {
		rs.setPhase("error", "token: "+err.Error())
		rs.mu.Lock()
		rs.enabled = false
		rs.mu.Unlock()
		return
	}

	bin, err := ensureCloudflared(func(msg string) {
		rs.setPhase("downloading", msg)
	})
	if err != nil {
		rs.setPhase("error", err.Error())
		rs.mu.Lock()
		rs.enabled = false
		rs.mu.Unlock()
		return
	}
	rs.setPhase("starting", "")

	if err := rs.startShareProxy(token); err != nil {
		rs.setPhase("error", "proxy: "+err.Error())
		rs.mu.Lock()
		rs.enabled = false
		rs.mu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, bin, "tunnel", "--no-autoupdate", "--url", "http://"+remoteProxyAddr)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		rs.stop("error")
		rs.setPhase("error", err.Error())
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		rs.stop("error")
		rs.setPhase("error", err.Error())
		return
	}

	rs.mu.Lock()
	rs.token = token
	rs.cfCmd = cmd
	rs.cfCancel = cancel
	rs.mu.Unlock()

	if err := cmd.Start(); err != nil {
		cancel()
		rs.stop("error")
		rs.setPhase("error", "cloudflared start: "+err.Error())
		return
	}

	urlCh := make(chan string, 1)
	go scanTunnelURL(stdout, urlCh)
	go scanTunnelURL(stderr, urlCh)

	var public string
	select {
	case public = <-urlCh:
	case <-time.After(75 * time.Second):
		rs.stop("error")
		rs.setPhase("error", "timeout waiting for tunnel URL (robot needs outbound HTTPS)")
		return
	case <-ctx.Done():
		return
	}

	share := strings.TrimRight(public, "/") + "/r/" + token + "/#control"
	expires := time.Now().Add(remoteDefaultTTL)

	rs.mu.Lock()
	if rs.cfCmd != cmd {
		rs.mu.Unlock()
		return
	}
	rs.publicURL = public
	rs.shareURL = share
	rs.expires = expires
	rs.phase = "ready"
	rs.errMsg = ""
	rs.enabled = true
	if rs.expireT != nil {
		rs.expireT.Stop()
	}
	rs.expireT = time.AfterFunc(remoteDefaultTTL, func() {
		log.Println("control remote share: expired, disabling")
		rs.stop("idle")
	})
	rs.mu.Unlock()

	go func() {
		err := cmd.Wait()
		rs.mu.Lock()
		still := rs.cfCmd == cmd
		rs.mu.Unlock()
		if still {
			msg := "tunnel exited"
			if err != nil {
				msg = "tunnel exited: " + err.Error()
			}
			rs.stop("error")
			rs.setPhase("error", msg)
		}
	}()

	log.Println("control remote share ready:", share)
}

func scanTunnelURL(r io.Reader, out chan<- string) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if m := tryCloudflareURL.FindString(line); m != "" {
			select {
			case out <- m:
			default:
			}
			return
		}
	}
}

func (rs *remoteShare) startShareProxy(token string) error {
	target, err := url.Parse("http://127.0.0.1:8080")
	if err != nil {
		return err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		req.Host = target.Host
		req.Header.Set("X-Forwarded-Proto", "https")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/r/"+token) {
			http.SetCookie(w, &http.Cookie{
				Name:     remoteShareCookie,
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				Secure:   true,
				MaxAge:   int(remoteDefaultTTL.Seconds()),
			})
			http.Redirect(w, r, "/#control", http.StatusFound)
			return
		}
		c, err := r.Cookie(remoteShareCookie)
		if err != nil || c.Value != token {
			http.Error(w, "Remote control link invalid or expired. Enable sharing again on the robot LAN UI.", http.StatusUnauthorized)
			return
		}
		proxy.ServeHTTP(w, r)
	})

	ln, err := net.Listen("tcp", remoteProxyAddr)
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: mux}
	rs.mu.Lock()
	rs.proxySrv = srv
	rs.mu.Unlock()
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Println("control remote proxy:", err)
		}
	}()
	return nil
}

func randomHex(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func cloudflaredAssetName() string {
	switch runtime.GOARCH {
	case "arm":
		return "cloudflared-linux-arm"
	case "arm64":
		return "cloudflared-linux-arm64"
	default:
		return "cloudflared-linux-amd64"
	}
}

func cloudflaredInstallDir() string {
	if st, err := os.Stat("/data/wired"); err == nil && st.IsDir() {
		return "/data/wired/bin"
	}
	return filepath.Join(os.TempDir(), "wired-bin")
}

func ensureCloudflared(progress func(string)) (string, error) {
	if p, err := exec.LookPath("cloudflared"); err == nil {
		return p, nil
	}
	dir := cloudflaredInstallDir()
	dest := filepath.Join(dir, "cloudflared")
	if st, err := os.Stat(dest); err == nil && st.Mode().IsRegular() && st.Mode()&0o111 != 0 {
		return dest, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}
	asset := cloudflaredAssetName()
	dl := fmt.Sprintf("https://github.com/cloudflare/cloudflared/releases/download/%s/%s", remoteCloudflaredVer, asset)
	if progress != nil {
		progress("Downloading cloudflared…")
	}
	tmp := dest + ".tmp"
	if err := downloadFile(dl, tmp); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("download cloudflared: %w", err)
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return dest, nil
}

func downloadFile(urlStr, dest string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "wired-remote-share/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %s", resp.Status)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
