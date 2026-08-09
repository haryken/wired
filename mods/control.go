package mods

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/digital-dream-labs/vector-go-sdk/pkg/vector"
	"github.com/digital-dream-labs/vector-go-sdk/pkg/vectorpb"
	"github.com/os-vector/wired/vars"
)

// Control is WirePod-style beta robot drive / camera / mirror on :8080.
type Control struct {
	vars.Modification
}

func NewControl() *Control {
	return &Control{}
}

func (modu *Control) Name() string {
	return "Control"
}

func (modu *Control) Description() string {
	return "Beta: assume control, drive pad, camera stream, mirror mode"
}

func (modu *Control) Load() error {
	return nil
}

var (
	ctrlMu       sync.Mutex
	bcAssuming   int32
	bcStop       chan bool
	bcCancel     context.CancelFunc
	camStreaming int32
	camCancel    context.CancelFunc
)

func (m *Control) HTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/mods/"+m.Name()) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/mods/"+m.Name()+"/")
	switch path {
	case "assume":
		if err := assumeControl(); err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		vars.HTTPSuccess(w, r)
	case "release":
		stopMicStream()
		releaseControl()
		_ = setMirror(false)
		_ = driveWheels(0, 0)
		stopCamFlag()
		vars.HTTPSuccess(w, r)
	case "status":
		fmt.Fprintf(w, `{"assuming":%v,"cam":%v,"mic":%v}`,
			atomic.LoadInt32(&bcAssuming) == 1,
			atomic.LoadInt32(&camStreaming) == 1,
			atomic.LoadInt32(&micStreaming) == 1)
	case "wheels":
		lw, _ := strconv.ParseFloat(r.FormValue("lw"), 32)
		rw, _ := strconv.ParseFloat(r.FormValue("rw"), 32)
		if err := driveWheels(float32(lw), float32(rw)); err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		vars.HTTPSuccess(w, r)
	case "lift":
		sp, _ := strconv.ParseFloat(r.FormValue("speed"), 32)
		if err := moveLift(float32(sp)); err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		vars.HTTPSuccess(w, r)
	case "head":
		sp, _ := strconv.ParseFloat(r.FormValue("speed"), 32)
		if err := moveHead(float32(sp)); err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		vars.HTTPSuccess(w, r)
	case "say_text":
		text := r.FormValue("text")
		if text == "" {
			vars.HTTPError(w, r, "empty text")
			return
		}
		if err := sayText(text); err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		vars.HTTPSuccess(w, r)
	case "play_sound":
		if err := playUploadedSound(r); err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		vars.HTTPSuccess(w, r)
	case "mic-stream":
		serveMicStream(w, r)
	case "mic-stop":
		stopMicStream()
		vars.HTTPSuccess(w, r)
	case "mirror":
		en := r.FormValue("enable") == "true" || r.FormValue("enable") == "1"
		if err := setMirror(en); err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		vars.HTTPSuccess(w, r)
	case "stop_cam":
		stopCamFlag()
		vars.HTTPSuccess(w, r)
	case "cam-stream":
		serveCamStream(w, r)
	default:
		vars.HTTPError(w, r, "404 not found")
	}
}

func getRobot() (*vector.Vector, error) {
	return vars.GetVec()
}

func rpcCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 8*time.Second)
}

func assumeControl() error {
	ctrlMu.Lock()
	defer ctrlMu.Unlock()
	if atomic.LoadInt32(&bcAssuming) == 1 {
		return nil
	}
	v, err := getRobot()
	if err != nil {
		return err
	}
	start := make(chan bool, 1)
	stop := make(chan bool, 1)
	ctx, cancel := context.WithCancel(context.Background())
	bcStop = stop
	bcCancel = cancel
	atomic.StoreInt32(&bcAssuming, 1)

	go func() {
		defer cancel()
		r, err := v.Conn.BehaviorControl(ctx)
		if err != nil {
			log.Println("[Control] BehaviorControl:", err)
			atomic.StoreInt32(&bcAssuming, 0)
			return
		}
		req := &vectorpb.BehaviorControlRequest{
			RequestType: &vectorpb.BehaviorControlRequest_ControlRequest{
				ControlRequest: &vectorpb.ControlRequest{
					Priority: vectorpb.ControlRequest_OVERRIDE_BEHAVIORS,
				},
			},
		}
		if err := r.Send(req); err != nil {
			log.Println("[Control] send control:", err)
			atomic.StoreInt32(&bcAssuming, 0)
			return
		}
		for {
			resp, err := r.Recv()
			if err != nil {
				log.Println("[Control] recv:", err)
				atomic.StoreInt32(&bcAssuming, 0)
				return
			}
			if resp.GetControlGrantedResponse() != nil {
				start <- true
				break
			}
		}
		select {
		case <-stop:
		case <-ctx.Done():
		}
		_ = r.Send(&vectorpb.BehaviorControlRequest{
			RequestType: &vectorpb.BehaviorControlRequest_ControlRelease{
				ControlRelease: &vectorpb.ControlRelease{},
			},
		})
		atomic.StoreInt32(&bcAssuming, 0)
	}()

	select {
	case <-start:
		return nil
	case <-time.After(8 * time.Second):
		atomic.StoreInt32(&bcAssuming, 0)
		cancel()
		select {
		case stop <- true:
		default:
		}
		return fmt.Errorf("timeout waiting for behavior control")
	}
}

func releaseControl() {
	ctrlMu.Lock()
	defer ctrlMu.Unlock()
	if atomic.LoadInt32(&bcAssuming) == 0 {
		return
	}
	if bcCancel != nil {
		bcCancel()
		bcCancel = nil
	}
	if bcStop != nil {
		select {
		case bcStop <- true:
		default:
		}
	}
	// assume goroutine clears bcAssuming after release send
	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&bcAssuming) == 1 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	atomic.StoreInt32(&bcAssuming, 0)
}

func driveWheels(lw, rw float32) error {
	v, err := getRobot()
	if err != nil {
		return err
	}
	ctx, cancel := rpcCtx()
	defer cancel()
	_, err = v.Conn.DriveWheels(ctx, &vectorpb.DriveWheelsRequest{
		LeftWheelMmps:   lw,
		RightWheelMmps:  rw,
		LeftWheelMmps2:  lw,
		RightWheelMmps2: rw,
	})
	return err
}

func moveLift(speed float32) error {
	v, err := getRobot()
	if err != nil {
		return err
	}
	ctx, cancel := rpcCtx()
	defer cancel()
	_, err = v.Conn.MoveLift(ctx, &vectorpb.MoveLiftRequest{SpeedRadPerSec: speed})
	return err
}

func moveHead(speed float32) error {
	v, err := getRobot()
	if err != nil {
		return err
	}
	ctx, cancel := rpcCtx()
	defer cancel()
	_, err = v.Conn.MoveHead(ctx, &vectorpb.MoveHeadRequest{SpeedRadPerSec: speed})
	return err
}

func sayText(text string) error {
	v, err := getRobot()
	if err != nil {
		return err
	}
	ctx, cancel := rpcCtx()
	defer cancel()
	_, err = v.Conn.SayText(ctx, &vectorpb.SayTextRequest{
		Text:           text,
		UseVectorVoice: true,
		DurationScalar: 1.0,
	})
	return err
}

func setMirror(enable bool) error {
	v, err := getRobot()
	if err != nil {
		return err
	}
	ctx, cancel := rpcCtx()
	defer cancel()
	_, err = v.Conn.EnableMirrorMode(ctx, &vectorpb.EnableMirrorModeRequest{Enable: enable})
	return err
}

func stopCamFlag() {
	ctrlMu.Lock()
	if camCancel != nil {
		camCancel()
		camCancel = nil
	}
	ctrlMu.Unlock()
	atomic.StoreInt32(&camStreaming, 0)
}

func serveCamStream(w http.ResponseWriter, r *http.Request) {
	// Stop any previous stream quickly so a new one can attach.
	if atomic.LoadInt32(&camStreaming) == 1 {
		stopCamFlag()
		time.Sleep(150 * time.Millisecond)
	}
	v, err := getRobot()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	enCtx, enCancel := context.WithTimeout(context.Background(), 3*time.Second)
	_, _ = v.Conn.EnableImageStreaming(enCtx, &vectorpb.EnableImageStreamingRequest{Enable: true})
	enCancel()

	ctx, cancel := context.WithCancel(r.Context())
	ctrlMu.Lock()
	camCancel = cancel
	ctrlMu.Unlock()

	client, err := v.Conn.CameraFeed(ctx, &vectorpb.CameraFeedRequest{})
	if err != nil {
		cancel()
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, okFlush := w.(http.Flusher)
	if !okFlush {
		cancel()
		http.Error(w, "streaming unsupported", 500)
		return
	}
	atomic.StoreInt32(&camStreaming, 1)
	defer func() {
		cancel()
		ctrlMu.Lock()
		if camCancel != nil {
			camCancel = nil
		}
		ctrlMu.Unlock()
		atomic.StoreInt32(&camStreaming, 0)
		offCtx, offCancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, _ = v.Conn.EnableImageStreaming(offCtx, &vectorpb.EnableImageStreamingRequest{Enable: false})
		offCancel()
	}()

	// Keep only the newest JPEG so a slow browser/network never builds backlog.
	latest := make(chan []byte, 1)
	recvDone := make(chan struct{})
	go func() {
		defer close(recvDone)
		for {
			resp, err := client.Recv()
			if err != nil {
				return
			}
			data := resp.GetData()
			if len(data) == 0 {
				continue
			}
			// Drop any unread older frame, then publish this one.
			select {
			case <-latest:
			default:
			}
			select {
			case latest <- data:
			default:
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-recvDone:
			return
		case data, ok := <-latest:
			if !ok {
				return
			}
			if atomic.LoadInt32(&camStreaming) == 0 {
				return
			}
			// Forward robot JPEG as-is (no decode/re-encode) for lower latency.
			if _, err := fmt.Fprintf(w, "--frame\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", len(data)); err != nil {
				return
			}
			if _, err := w.Write(data); err != nil {
				return
			}
			if _, err := fmt.Fprintf(w, "\r\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
