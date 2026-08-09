package mods

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wiredListenFlag = "/run/wired/listen_mic"
	wiredMicSock    = "/run/wired/mic.sock"
)

var (
	robotMicStreaming int32
	robotMicCancel    context.CancelFunc
)

func stopRobotMicStream() {
	ctrlMu.Lock()
	c := robotMicCancel
	robotMicCancel = nil
	ctrlMu.Unlock()
	if c != nil {
		c()
	}
	_ = os.Remove(wiredListenFlag)
	atomic.StoreInt32(&robotMicStreaming, 0)
}

// serveRobotMicStream relays processed mono mic PCM from vic-anim
// (/run/wired/mic.sock, Int16 LE @ 16 kHz) to the browser WebSocket.
func serveRobotMicStream(w http.ResponseWriter, r *http.Request) {
	if atomic.LoadInt32(&bcAssuming) != 1 {
		http.Error(w, "assume control first", http.StatusForbidden)
		return
	}
	if atomic.LoadInt32(&robotMicStreaming) == 1 {
		http.Error(w, "robot mic already streaming", http.StatusConflict)
		return
	}

	if err := os.MkdirAll("/run/wired", 0o755); err != nil {
		http.Error(w, "mkdir /run/wired: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_ = os.Remove(wiredMicSock)
	pc, err := net.ListenPacket("unixgram", wiredMicSock)
	if err != nil {
		http.Error(w, "listen mic sock: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_ = os.Chmod(wiredMicSock, 0o666)

	conn, err := micUpgrader.Upgrade(w, r, nil)
	if err != nil {
		_ = pc.Close()
		_ = os.Remove(wiredMicSock)
		log.Println("[Control] robot-mic ws upgrade:", err)
		return
	}

	if !atomic.CompareAndSwapInt32(&robotMicStreaming, 0, 1) {
		_ = conn.Close()
		_ = pc.Close()
		_ = os.Remove(wiredMicSock)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	ctrlMu.Lock()
	if robotMicCancel != nil {
		robotMicCancel()
	}
	robotMicCancel = cancel
	ctrlMu.Unlock()

	// Enable anim tap
	if err := os.WriteFile(wiredListenFlag, []byte("1\n"), 0o644); err != nil {
		log.Println("[Control] listen_mic flag:", err)
	}

	defer func() {
		cancel()
		_ = os.Remove(wiredListenFlag)
		_ = pc.Close()
		_ = os.Remove(wiredMicSock)
		_ = conn.Close()
		atomic.StoreInt32(&robotMicStreaming, 0)
		ctrlMu.Lock()
		if robotMicCancel != nil {
			robotMicCancel = nil
		}
		ctrlMu.Unlock()
	}()

	go func() {
		<-ctx.Done()
		_ = conn.Close()
		_ = pc.Close()
	}()
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cancel()
				return
			}
		}
	}()

	_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"status":"ready","rate":16000}`))

	buf := make([]byte, 4096)
	_ = pc.SetReadDeadline(time.Now().Add(15 * time.Second))
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_ = pc.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, _, err := pc.ReadFrom(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				// keep waiting while flag is up
				if atomic.LoadInt32(&robotMicStreaming) == 1 {
					continue
				}
			}
			return
		}
		if n < 2 {
			continue
		}
		if n%2 != 0 {
			n--
		}
		chunk := make([]byte, n)
		copy(chunk, buf[:n])
		if err := conn.WriteMessage(websocket.BinaryMessage, chunk); err != nil {
			return
		}
	}
}
