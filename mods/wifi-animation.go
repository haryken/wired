package mods

import (
	"context"
	"time"

	"github.com/digital-dream-labs/vector-go-sdk/pkg/vectorpb"
	"github.com/os-vector/wired/vars"
)

var wifiStatusAnimations = map[string]string{
	"trying": "anim_pairing_icon_wifi",
	"ok":     "anim_pounce_success_02",
	"fail":   "anim_pounce_fail_02",
}

// PlayWifiStatusAnimation uses Vector's existing Gateway animation API. It
// does not modify vic-anim or add work to its render/update loop.
func PlayWifiStatusAnimation(state string) {
	name := wifiStatusAnimations[state]
	if name == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		vec, err := vars.GetVec()
		if err != nil {
			wifiLog("status animation %s: gateway unavailable: %v", state, err)
			return
		}
		control, err := vec.Conn.AssumeBehaviorControl(ctx, &vectorpb.BehaviorControlRequest{
			RequestType: &vectorpb.BehaviorControlRequest_ControlRequest{
				ControlRequest: &vectorpb.ControlRequest{
					Priority: vectorpb.ControlRequest_DEFAULT,
				},
			},
		})
		if err != nil {
			wifiLog("status animation %s: control request failed: %v", state, err)
			return
		}
		granted, err := control.Recv()
		if err != nil || granted.GetControlGrantedResponse() == nil {
			wifiLog("status animation %s: control not granted: response=%v err=%v", state, granted, err)
			return
		}
		resp, err := vec.Conn.PlayAnimation(ctx, &vectorpb.PlayAnimationRequest{
			Animation: &vectorpb.Animation{Name: name},
			Loops:     1,
			// Show only the face track; never move the robot during WiFi setup.
			IgnoreBodyTrack: true,
			IgnoreHeadTrack: true,
			IgnoreLiftTrack: true,
		})
		if err != nil {
			wifiLog("status animation %s (%s) failed: %v", state, name, err)
			return
		}
		wifiLog("status animation %s (%s): %v", state, name, resp.GetStatus())
	}()
}
