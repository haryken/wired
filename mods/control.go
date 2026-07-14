package mods

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/jpeg"
	"io"
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
	ctrlMu        sync.Mutex
	bcAssuming    int32
	bcStop        chan bool
	camStreaming  int32
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
		releaseControl()
		_ = setMirror(false)
		_ = driveWheels(0, 0)
		stopCamFlag()
		vars.HTTPSuccess(w, r)
	case "status":
		fmt.Fprintf(w, `{"assuming":%v,"cam":%v}`,
			atomic.LoadInt32(&bcAssuming) == 1,
			atomic.LoadInt32(&camStreaming) == 1)
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

func getRobot() (*vector.Vector, context.Context, error) {
	v, err := vars.GetVec()
	if err != nil {
		return nil, nil, err
	}
	return v, context.Background(), nil
}

func assumeControl() error {
	ctrlMu.Lock()
	defer ctrlMu.Unlock()
	if atomic.LoadInt32(&bcAssuming) == 1 {
		return nil
	}
	v, ctx, err := getRobot()
	if err != nil {
		return err
	}
	start := make(chan bool, 1)
	stop := make(chan bool, 1)
	bcStop = stop
	atomic.StoreInt32(&bcAssuming, 1)

	go func() {
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
		<-stop
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
	v, ctx, err := getRobot()
	if err != nil {
		return err
	}
	_, err = v.Conn.DriveWheels(ctx, &vectorpb.DriveWheelsRequest{
		LeftWheelMmps:   lw,
		RightWheelMmps:  rw,
		LeftWheelMmps2:  lw,
		RightWheelMmps2: rw,
	})
	return err
}

func moveLift(speed float32) error {
	v, ctx, err := getRobot()
	if err != nil {
		return err
	}
	_, err = v.Conn.MoveLift(ctx, &vectorpb.MoveLiftRequest{SpeedRadPerSec: speed})
	return err
}

func moveHead(speed float32) error {
	v, ctx, err := getRobot()
	if err != nil {
		return err
	}
	_, err = v.Conn.MoveHead(ctx, &vectorpb.MoveHeadRequest{SpeedRadPerSec: speed})
	return err
}

func sayText(text string) error {
	v, ctx, err := getRobot()
	if err != nil {
		return err
	}
	_, err = v.Conn.SayText(ctx, &vectorpb.SayTextRequest{
		Text:           text,
		UseVectorVoice: true,
		DurationScalar: 1.0,
	})
	return err
}

func playUploadedSound(r *http.Request) error {
	file, _, err := r.FormFile("sound")
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.Copy(io.Discard, file); err != nil {
		return err
	}
	return fmt.Errorf("play_sound upload accepted but ExternalAudio host path not wired yet")
}

func setMirror(enable bool) error {
	v, ctx, err := getRobot()
	if err != nil {
		return err
	}
	_, err = v.Conn.EnableMirrorMode(ctx, &vectorpb.EnableMirrorModeRequest{Enable: enable})
	return err
}

func stopCamFlag() {
	atomic.StoreInt32(&camStreaming, 0)
}

func serveCamStream(w http.ResponseWriter, r *http.Request) {
	if atomic.LoadInt32(&camStreaming) == 1 {
		atomic.StoreInt32(&camStreaming, 0)
		time.Sleep(400 * time.Millisecond)
	}
	v, ctx, err := getRobot()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_, _ = v.Conn.EnableImageStreaming(ctx, &vectorpb.EnableImageStreamingRequest{Enable: true})
	client, err := v.Conn.CameraFeed(ctx, &vectorpb.CameraFeedRequest{})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=--boundary")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, _ := w.(http.Flusher)
	atomic.StoreInt32(&camStreaming, 1)
	defer func() {
		atomic.StoreInt32(&camStreaming, 0)
		_, _ = v.Conn.EnableImageStreaming(ctx, &vectorpb.EnableImageStreamingRequest{Enable: false})
	}()

	for {
		select {
		case <-r.Context().Done():
			return
		default:
			if atomic.LoadInt32(&camStreaming) == 0 {
				return
			}
			resp, err := client.Recv()
			if err != nil {
				return
			}
			data := resp.GetData()
			img, _, err := image.Decode(bytes.NewReader(data))
			if err != nil {
				// already jpeg — write raw
				fmt.Fprintf(w, "--boundary\r\nContent-Type: image/jpeg\r\n\r\n")
				_, _ = w.Write(data)
				fmt.Fprintf(w, "\r\n")
			} else {
				fmt.Fprintf(w, "--boundary\r\nContent-Type: image/jpeg\r\n\r\n")
				_ = jpeg.Encode(io.MultiWriter(w), img, &jpeg.Options{Quality: 50})
				fmt.Fprintf(w, "\r\n")
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}
