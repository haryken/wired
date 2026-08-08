package mods

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/digital-dream-labs/vector-go-sdk/pkg/vectorpb"
	"github.com/os-vector/wired/vars"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

// FreeTime is a watch-only tab: camera + live vision overlays while Vector freeplays.
type FreeTime struct {
	vars.Modification
}

func NewFreeTime() *FreeTime {
	return &FreeTime{}
}

func (m *FreeTime) Name() string { return "FreeTime" }

func (m *FreeTime) Description() string {
	return "Vector Freetime: camera with face/cube/expression/motion overlays (freeplay, no assume)"
}

func (m *FreeTime) Load() error { return nil }

type ftRect struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	W float32 `json:"w"`
	H float32 `json:"h"`
}

type ftFace struct {
	ID         int32  `json:"id"`
	Name       string `json:"name"`
	Expression string `json:"expression"`
	Rect       ftRect `json:"rect"`
	SeenAt     int64  `json:"seenAt"`
}

type ftObject struct {
	ID     int32  `json:"id"`
	Type   string `json:"type"`
	Rect   ftRect `json:"rect"`
	SeenAt int64  `json:"seenAt"`
}

// ftMotion is optical-flow motion centroid (not a classified object).
type ftMotion struct {
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	Area   float32 `json:"area"`
	SeenAt int64   `json:"seenAt"`
}

type ftSnapshot struct {
	Active  bool       `json:"active"`
	Stim    float64    `json:"stim"`
	Faces   []ftFace   `json:"faces"`
	Objects []ftObject `json:"objects"`
	Motion  *ftMotion  `json:"motion"`
	ImgW    int        `json:"imgW"`
	ImgH    int        `json:"imgH"`
}

var (
	ftMu       sync.Mutex
	ftActive   int32
	ftStop     chan struct{}
	ftCancel   context.CancelFunc
	ftFaces    = map[int32]ftFace{}
	ftObjects  = map[int32]ftObject{}
	ftMotionPt *ftMotion
	ftStim     float64
	ftImgW     = 640
	ftImgH     = 360
	ftStaleMs  = int64(1500)
)

func (m *FreeTime) HTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/mods/"+m.Name()) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/mods/"+m.Name()+"/")
	switch path {
	case "start":
		if err := ftStart(); err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		vars.HTTPSuccess(w, r)
	case "stop":
		ftStopVision()
		stopCamFlag()
		vars.HTTPSuccess(w, r)
	case "snapshot":
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ftBuildSnapshot())
	case "status":
		fmt.Fprintf(w, `{"active":%v}`, atomic.LoadInt32(&ftActive) == 1)
	default:
		vars.HTTPError(w, r, "404 not found")
	}
}

func ftBuildSnapshot() ftSnapshot {
	now := time.Now().UnixMilli()
	ftMu.Lock()
	defer ftMu.Unlock()

	faces := make([]ftFace, 0, len(ftFaces))
	for id, f := range ftFaces {
		if now-f.SeenAt > ftStaleMs {
			delete(ftFaces, id)
			continue
		}
		faces = append(faces, f)
	}
	objs := make([]ftObject, 0, len(ftObjects))
	for id, o := range ftObjects {
		if now-o.SeenAt > ftStaleMs {
			delete(ftObjects, id)
			continue
		}
		objs = append(objs, o)
	}
	var motion *ftMotion
	if ftMotionPt != nil && now-ftMotionPt.SeenAt <= ftStaleMs {
		cp := *ftMotionPt
		motion = &cp
	} else {
		ftMotionPt = nil
	}
	return ftSnapshot{
		Active:  atomic.LoadInt32(&ftActive) == 1,
		Stim:    ftStim,
		Faces:   faces,
		Objects: objs,
		Motion:  motion,
		ImgW:    ftImgW,
		ImgH:    ftImgH,
	}
}

func ftExprLabel(e vectorpb.FacialExpression) string {
	s := e.String()
	s = strings.TrimPrefix(s, "FacialExpression_")
	s = strings.TrimPrefix(s, "EXPRESSION_")
	if s == "" || s == "UNKNOWN" {
		return ""
	}
	return s
}

func ftIsTimeout(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "DeadlineExceeded") || strings.Contains(s, "context deadline")
}

// extractMotionFromEvent pulls robot_observed_motion (proto field 20) from Event
// unknown fields — stock vector-go-sdk Event schema omits that oneof arm.
func extractMotionFromEvent(ev *vectorpb.Event) (x, y int32, area float32, ok bool) {
	if ev == nil {
		return
	}
	raw, err := proto.Marshal(ev)
	if err != nil {
		return
	}
	for len(raw) > 0 {
		num, typ, n := protowire.ConsumeTag(raw)
		if n < 0 {
			return
		}
		raw = raw[n:]
		if num == 20 && typ == protowire.BytesType {
			val, n := protowire.ConsumeBytes(raw)
			if n < 0 {
				return
			}
			return parseMotionPayload(val)
		}
		n = protowire.ConsumeFieldValue(num, typ, raw)
		if n < 0 {
			return
		}
		raw = raw[n:]
	}
	return
}

func parseMotionPayload(b []byte) (x, y int32, area float32, ok bool) {
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return
		}
		b = b[n:]
		switch {
		case num == 2 && typ == protowire.Fixed32Type:
			v, n := protowire.ConsumeFixed32(b)
			if n < 0 {
				return
			}
			area = math.Float32frombits(v)
			b = b[n:]
		case num == 3 && (typ == protowire.VarintType):
			v, n := protowire.ConsumeVarint(b)
			if n < 0 {
				return
			}
			x = int32(v)
			b = b[n:]
		case num == 4 && (typ == protowire.VarintType):
			v, n := protowire.ConsumeVarint(b)
			if n < 0 {
				return
			}
			y = int32(v)
			ok = true
			b = b[n:]
		default:
			n = protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return
			}
			b = b[n:]
		}
	}
	return
}

func ftStart() error {
	ftMu.Lock()
	defer ftMu.Unlock()
	if atomic.LoadInt32(&ftActive) == 1 {
		return nil
	}

	v, err := vars.GetVec()
	if err != nil {
		return err
	}

	// Short deadlines: gateway waits for engine vision confirm; don't block the web UI forever.
	// WireOS engine allows Markers/Faces/Motion without BehaviorControl (sdkComponent).
	enFace, cancelFace := context.WithTimeout(context.Background(), 3*time.Second)
	_, err = v.Conn.EnableFaceDetection(enFace, &vectorpb.EnableFaceDetectionRequest{
		Enable:                     true,
		EnableExpressionEstimation: true,
	})
	cancelFace()
	if err != nil && !ftIsTimeout(err) {
		return fmt.Errorf("EnableFaceDetection: %w", err)
	}
	enMark, cancelMark := context.WithTimeout(context.Background(), 3*time.Second)
	_, err = v.Conn.EnableMarkerDetection(enMark, &vectorpb.EnableMarkerDetectionRequest{Enable: true})
	cancelMark()
	if err != nil && !ftIsTimeout(err) {
		return fmt.Errorf("EnableMarkerDetection: %w", err)
	}
	enMot, cancelMot := context.WithTimeout(context.Background(), 3*time.Second)
	_, err = v.Conn.EnableMotionDetection(enMot, &vectorpb.EnableMotionDetectionRequest{Enable: true})
	cancelMot()
	if err != nil && !ftIsTimeout(err) {
		return fmt.Errorf("EnableMotionDetection: %w", err)
	}

	streamCtx, cancel := context.WithCancel(context.Background())
	client, err := v.Conn.EventStream(streamCtx, &vectorpb.EventRequest{
		ListType: &vectorpb.EventRequest_WhiteList{
			WhiteList: &vectorpb.FilterList{
				List: []string{"robot_observed_face", "object_event", "stimulation_info", "robot_observed_motion"},
			},
		},
		ConnectionId: "wired-freetime",
	})
	if err != nil {
		cancel()
		return fmt.Errorf("EventStream: %w", err)
	}

	stop := make(chan struct{})
	ftStop = stop
	ftCancel = cancel
	ftFaces = map[int32]ftFace{}
	ftObjects = map[int32]ftObject{}
	ftMotionPt = nil
	ftStim = 0
	atomic.StoreInt32(&ftActive, 1)

	go func() {
		defer cancel()
		defer atomic.StoreInt32(&ftActive, 0)
		for {
			resp, err := client.Recv()
			if err != nil {
				log.Println("[FreeTime] EventStream recv:", err)
				return
			}
			select {
			case <-stop:
				return
			default:
			}
			ev := resp.GetEvent()
			if ev == nil {
				continue
			}
			now := time.Now().UnixMilli()

			if face := ev.GetRobotObservedFace(); face != nil {
				r := face.GetImgRect()
				if r == nil {
					continue
				}
				ftMu.Lock()
				ftFaces[face.GetFaceId()] = ftFace{
					ID:         face.GetFaceId(),
					Name:       face.GetName(),
					Expression: ftExprLabel(face.GetExpression()),
					Rect:       ftRect{X: r.GetXTopLeft(), Y: r.GetYTopLeft(), W: r.GetWidth(), H: r.GetHeight()},
					SeenAt:     now,
				}
				ftMu.Unlock()
				continue
			}

			if oe := ev.GetObjectEvent(); oe != nil {
				if obj := oe.GetRobotObservedObject(); obj != nil {
					r := obj.GetImgRect()
					if r == nil {
						continue
					}
					ftMu.Lock()
					ftObjects[obj.GetObjectId()] = ftObject{
						ID:     obj.GetObjectId(),
						Type:   strings.TrimPrefix(obj.GetObjectType().String(), "ObjectType_"),
						Rect:   ftRect{X: r.GetXTopLeft(), Y: r.GetYTopLeft(), W: r.GetWidth(), H: r.GetHeight()},
						SeenAt: now,
					}
					ftMu.Unlock()
				}
				continue
			}

			if mx, my, area, mok := extractMotionFromEvent(ev); mok {
				ftMu.Lock()
				ftMotionPt = &ftMotion{X: float32(mx), Y: float32(my), Area: area, SeenAt: now}
				ftMu.Unlock()
				continue
			}

			if info := ev.GetStimulationInfo(); info != nil {
				s := fmt.Sprint(info)
				if strings.Contains(s, "velocity") {
					ftMu.Lock()
					ftStim = float64(info.Value)
					ftMu.Unlock()
				}
			}
		}
	}()
	return nil
}

func ftStopVision() {
	ftMu.Lock()
	if ftCancel != nil {
		ftCancel()
		ftCancel = nil
	}
	if ftStop != nil {
		select {
		case <-ftStop:
		default:
			close(ftStop)
		}
		ftStop = nil
	}
	atomic.StoreInt32(&ftActive, 0)
	ftFaces = map[int32]ftFace{}
	ftObjects = map[int32]ftObject{}
	ftMotionPt = nil
	ftStim = 0
	ftMu.Unlock()

	if v, err := vars.GetVec(); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, _ = v.Conn.EnableFaceDetection(ctx, &vectorpb.EnableFaceDetectionRequest{Enable: false})
		_, _ = v.Conn.EnableMarkerDetection(ctx, &vectorpb.EnableMarkerDetectionRequest{Enable: false})
		_, _ = v.Conn.EnableMotionDetection(ctx, &vectorpb.EnableMotionDetectionRequest{Enable: false})
		cancel()
	}
}
