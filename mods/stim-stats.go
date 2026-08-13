package mods

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/digital-dream-labs/vector-go-sdk/pkg/vectorpb"
	"github.com/os-vector/wired/vars"
)

// StimStats exposes WirePod-style stimulation graph + lifetime statistics.
type StimStats struct {
	vars.Modification
}

func NewStimStats() *StimStats {
	return &StimStats{}
}

func (m *StimStats) Name() string { return "StimStats" }

func (m *StimStats) Description() string {
	return "Stimulation stream and robot lifetime statistics"
}

func (m *StimStats) Load() error { return nil }

var (
	stimMu        sync.Mutex
	stimStreaming int32
	stimValue     float64
	stimStop      chan struct{}
	stimCancel    context.CancelFunc
)

func (m *StimStats) HTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/mods/"+m.Name()) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/mods/"+m.Name()+"/")
	switch path {
	case "begin_stim":
		if err := beginStimStream(); err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		vars.HTTPSuccess(w, r)
	case "stop_stim":
		stopStimStream()
		vars.HTTPSuccess(w, r)
	case "get_stim":
		if atomic.LoadInt32(&stimStreaming) != 1 {
			vars.HTTPError(w, r, "stim stream not running")
			return
		}
		stimMu.Lock()
		v := stimValue
		stimMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	case "get_stats":
		doc, err := getLifetimeStats()
		if err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(doc))
	case "get_hold":
		hold, val := readStimHold()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"hold":  hold,
			"value": val,
		})
	case "set_stim":
		if err := setStimHold(r); err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		vars.HTTPSuccess(w, r)
	default:
		vars.HTTPError(w, r, "404 not found")
	}
}

func beginStimStream() error {
	stimMu.Lock()
	defer stimMu.Unlock()
	if atomic.LoadInt32(&stimStreaming) == 1 {
		return nil
	}
	v, err := vars.GetVec()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	client, err := v.Conn.EventStream(ctx, &vectorpb.EventRequest{
		ListType: &vectorpb.EventRequest_WhiteList{
			WhiteList: &vectorpb.FilterList{
				List: []string{"stimulation_info"},
			},
		},
		ConnectionId: "wired",
	})
	if err != nil {
		cancel()
		return err
	}
	stop := make(chan struct{})
	stimStop = stop
	stimCancel = cancel
	atomic.StoreInt32(&stimStreaming, 1)
	stimValue = 0

	go func() {
		defer cancel()
		defer atomic.StoreInt32(&stimStreaming, 0)
		for {
			resp, err := client.Recv()
			if err != nil {
				return
			}
			select {
			case <-stop:
				return
			default:
			}
			info := resp.GetEvent().GetStimulationInfo()
			if info == nil {
				continue
			}
			// Proto3 omits zero floats from String(), so a held stim (velocity=0)
			// never contains "velocity". Always take the value field.
			stimMu.Lock()
			stimValue = float64(info.GetValue())
			stimMu.Unlock()
		}
	}()
	return nil
}

func stopStimStream() {
	stimMu.Lock()
	defer stimMu.Unlock()
	if stimCancel != nil {
		stimCancel()
		stimCancel = nil
	}
	if stimStop != nil {
		select {
		case <-stimStop:
		default:
			close(stimStop)
		}
		stimStop = nil
	}
	atomic.StoreInt32(&stimStreaming, 0)
	stimValue = 0
}

func getLifetimeStats() (string, error) {
	v, err := vars.GetVec()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := v.Conn.PullJdocs(ctx, &vectorpb.PullJdocsRequest{
		JdocTypes: []vectorpb.JdocType{vectorpb.JdocType_ROBOT_LIFETIME_STATS},
	})
	if err != nil {
		return "", err
	}
	if len(resp.NamedJdocs) == 0 || resp.NamedJdocs[0].Doc == nil {
		return "", fmt.Errorf("empty lifetime stats")
	}
	return resp.NamedJdocs[0].Doc.JsonDoc, nil
}

func stimHoldPath() string {
	return filepath.Join(vars.GetModDir("StimStats"), "hold")
}

func readStimHold() (hold bool, value float64) {
	raw, err := vars.ReadFile(stimHoldPath())
	if err != nil {
		return false, 0
	}
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" || s == "auto" || s == "off" || s == "false" || s == "none" {
		return false, 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return false, 0
	}
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	return true, v
}

func setStimHold(r *http.Request) error {
	_ = r.ParseForm()
	holdStr := strings.TrimSpace(strings.ToLower(r.FormValue("hold")))
	if holdStr == "" {
		holdStr = "true"
	}
	hold := holdStr != "false" && holdStr != "0" && holdStr != "off" && holdStr != "auto"
	path := stimHoldPath()
	if !hold {
		_ = os.Remove(path)
		// Keep an explicit auto marker so get_hold is unambiguous after a save.
		return vars.SaveFile("auto\n", path)
	}
	valStr := strings.TrimSpace(r.FormValue("value"))
	if valStr == "" {
		valStr = "0.9"
	}
	v, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return fmt.Errorf("invalid stim value")
	}
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	return vars.SaveFile(strconv.FormatFloat(v, 'f', 3, 64)+"\n", path)
}
