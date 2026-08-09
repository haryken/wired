package mods

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var micUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // LAN robot UI
}

// serveMicStream accepts a browser WebSocket of raw Int16 LE mono PCM @ 8 kHz
// and forwards it to Vector ExternalAudio while Assume Control is held.
func serveMicStream(w http.ResponseWriter, r *http.Request) {
	if atomic.LoadInt32(&bcAssuming) != 1 {
		http.Error(w, "assume control first", http.StatusForbidden)
		return
	}
	if extAudioBusy() || atomic.LoadInt32(&micStreaming) == 1 {
		http.Error(w, "audio already in use", http.StatusConflict)
		return
	}

	conn, err := micUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[Control] mic ws upgrade:", err)
		return
	}
	defer conn.Close()

	if !atomic.CompareAndSwapInt32(&micStreaming, 0, 1) {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"mic busy"}`))
		return
	}
	if !atomic.CompareAndSwapInt32(&extAudioActive, 0, 1) {
		atomic.StoreInt32(&micStreaming, 0)
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"audio busy"}`))
		return
	}
	defer func() {
		atomic.StoreInt32(&micStreaming, 0)
		atomic.StoreInt32(&extAudioActive, 0)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	ctrlMu.Lock()
	if micCancel != nil {
		micCancel()
	}
	micCancel = cancel
	ctrlMu.Unlock()
	defer func() {
		cancel()
		ctrlMu.Lock()
		if micCancel != nil {
			micCancel = nil
		}
		ctrlMu.Unlock()
	}()

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"status":"ready","rate":8000}`))

	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		errCh <- streamPCMReader(ctx, pr, 8000, extAudioDefaultVol, false)
	}()

	for {
		mt, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if mt != websocket.BinaryMessage || len(data) == 0 {
			continue
		}
		if len(data)%2 != 0 {
			data = data[:len(data)-1]
		}
		if _, err := pw.Write(data); err != nil {
			break
		}
	}
	_ = pw.Close()

	select {
	case err := <-errCh:
		if err != nil && err != io.EOF && err != context.Canceled {
			log.Println("[Control] mic ExternalAudio:", err)
		}
	case <-time.After(8 * time.Second):
		cancel()
		<-errCh
	}
}

func playUploadedSound(r *http.Request) error {
	if atomic.LoadInt32(&bcAssuming) != 1 {
		return fmt.Errorf("assume control first")
	}
	if extAudioBusy() {
		return fmt.Errorf("audio already playing")
	}
	file, _, err := r.FormFile("sound")
	if err != nil {
		return err
	}
	defer file.Close()
	limited := io.LimitReader(file, extAudioMaxUpload+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return err
	}
	if len(data) > extAudioMaxUpload {
		return fmt.Errorf("file too large (max %d bytes)", extAudioMaxUpload)
	}
	pcm, rate, err := extractWAVPCM(data)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	return playPCM16LE(ctx, pcm, rate, extAudioDefaultVol)
}
