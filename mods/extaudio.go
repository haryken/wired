package mods

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"sync/atomic"
	"time"

	"github.com/digital-dream-labs/vector-go-sdk/pkg/vectorpb"
)

const (
	extAudioDefaultVol = 100
	extAudioMaxChunk   = 1024 // engine hard max
	extAudioMinRate    = 8000
	extAudioMaxRate    = 16025
	extAudioMaxUpload  = 5 << 20 // 5 MiB WAV
)

var (
	extAudioActive int32
	micStreaming   int32
	micCancel      context.CancelFunc
)

func extAudioBusy() bool {
	return atomic.LoadInt32(&extAudioActive) == 1
}

// playPCM16LE streams raw 16-bit LE mono PCM to Vector's speaker via ExternalAudio.
func playPCM16LE(ctx context.Context, pcm []byte, frameRate, volume uint32) error {
	if len(pcm) == 0 {
		return fmt.Errorf("empty pcm")
	}
	if frameRate < extAudioMinRate || frameRate > extAudioMaxRate {
		return fmt.Errorf("sample rate %d out of range (%d-%d)", frameRate, extAudioMinRate, extAudioMaxRate)
	}
	if volume > 100 {
		volume = 100
	}
	if !atomic.CompareAndSwapInt32(&extAudioActive, 0, 1) {
		return fmt.Errorf("audio already playing")
	}
	defer atomic.StoreInt32(&extAudioActive, 0)

	return streamPCMReader(ctx, bytes.NewReader(pcm), frameRate, volume, true)
}

// streamPCMReader pushes PCM from r. If pace is true, throttles so we stay ~0.5s ahead of realtime
// (file playback). Live mic should pass pace=false.
func streamPCMReader(ctx context.Context, r io.Reader, frameRate, volume uint32, pace bool) error {
	v, err := getRobot()
	if err != nil {
		return err
	}
	stream, err := v.Conn.ExternalAudioStreamPlayback(ctx)
	if err != nil {
		return fmt.Errorf("ExternalAudioStreamPlayback: %w", err)
	}

	done := make(chan struct{})
	var playErr error
	go func() {
		defer close(done)
		for {
			resp, err := stream.Recv()
			if err != nil {
				if err != io.EOF && ctx.Err() == nil {
					playErr = err
				}
				return
			}
			if ov := resp.GetAudioStreamBufferOverrun(); ov != nil {
				log.Printf("[Control] ExternalAudio overrun sent=%d played=%d", ov.GetAudioSamplesSent(), ov.GetAudioSamplesPlayed())
			}
			if resp.GetAudioStreamPlaybackFailyer() != nil {
				playErr = fmt.Errorf("ExternalAudio playback failure")
				return
			}
			if resp.GetAudioStreamPlaybackComplete() != nil {
				return
			}
		}
	}()

	prep := &vectorpb.ExternalAudioStreamRequest{
		AudioRequestType: &vectorpb.ExternalAudioStreamRequest_AudioStreamPrepare{
			AudioStreamPrepare: &vectorpb.ExternalAudioStreamPrepare{
				AudioFrameRate: frameRate,
				AudioVolume:    volume,
			},
		},
	}
	if err := stream.Send(prep); err != nil {
		_ = stream.CloseSend()
		<-done
		return fmt.Errorf("prepare: %w", err)
	}

	buf := make([]byte, extAudioMaxChunk)
	start := time.Now()
	var framesSent int64 // 16-bit samples

	for {
		if ctx.Err() != nil {
			_ = stream.Send(&vectorpb.ExternalAudioStreamRequest{
				AudioRequestType: &vectorpb.ExternalAudioStreamRequest_AudioStreamCancel{
					AudioStreamCancel: &vectorpb.ExternalAudioStreamCancel{},
				},
			})
			_ = stream.CloseSend()
			<-done
			return ctx.Err()
		}
		n, readErr := io.ReadFull(r, buf)
		if n > 0 {
			// Ensure even byte count (16-bit samples).
			if n%2 != 0 {
				n--
			}
			chunk := append([]byte(nil), buf[:n]...)
			msg := &vectorpb.ExternalAudioStreamRequest{
				AudioRequestType: &vectorpb.ExternalAudioStreamRequest_AudioStreamChunk{
					AudioStreamChunk: &vectorpb.ExternalAudioStreamChunk{
						AudioChunkSizeBytes: uint32(len(chunk)),
						AudioChunkSamples:   chunk,
					},
				},
			}
			if err := stream.Send(msg); err != nil {
				_ = stream.CloseSend()
				<-done
				return fmt.Errorf("chunk: %w", err)
			}
			framesSent += int64(len(chunk) / 2)
			if pace && frameRate > 0 {
				elapsed := time.Since(start).Seconds()
				expected := elapsed * float64(frameRate)
				ahead := (float64(framesSent) - expected) / float64(frameRate)
				if ahead > 1.0 {
					time.Sleep(time.Duration((ahead - 0.5) * float64(time.Second)))
				}
			}
		}
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			_ = stream.Send(&vectorpb.ExternalAudioStreamRequest{
				AudioRequestType: &vectorpb.ExternalAudioStreamRequest_AudioStreamCancel{
					AudioStreamCancel: &vectorpb.ExternalAudioStreamCancel{},
				},
			})
			_ = stream.CloseSend()
			<-done
			return readErr
		}
	}

	if err := stream.Send(&vectorpb.ExternalAudioStreamRequest{
		AudioRequestType: &vectorpb.ExternalAudioStreamRequest_AudioStreamComplete{
			AudioStreamComplete: &vectorpb.ExternalAudioStreamComplete{},
		},
	}); err != nil {
		_ = stream.CloseSend()
		<-done
		return fmt.Errorf("complete: %w", err)
	}
	_ = stream.CloseSend()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for playback complete")
	case <-ctx.Done():
		return ctx.Err()
	}
	return playErr
}

// extractWAVPCM returns 16-bit LE mono PCM and sample rate from a WAV buffer.
func extractWAVPCM(data []byte) (pcm []byte, rate uint32, err error) {
	if len(data) < 44 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, 0, fmt.Errorf("not a WAV file")
	}
	off := 12
	var audioFormat, channels, bits uint16
	var sampleRate uint32
	var dataOff, dataLen int
	for off+8 <= len(data) {
		chunkID := string(data[off : off+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[off+4 : off+8]))
		off += 8
		if off+chunkSize > len(data) {
			break
		}
		switch chunkID {
		case "fmt ":
			if chunkSize < 16 {
				return nil, 0, fmt.Errorf("bad fmt chunk")
			}
			audioFormat = binary.LittleEndian.Uint16(data[off : off+2])
			channels = binary.LittleEndian.Uint16(data[off+2 : off+4])
			sampleRate = binary.LittleEndian.Uint32(data[off+4 : off+8])
			bits = binary.LittleEndian.Uint16(data[off+14 : off+16])
		case "data":
			dataOff = off
			dataLen = chunkSize
		}
		off += chunkSize
		if chunkSize%2 == 1 {
			off++ // pad byte
		}
	}
	if dataOff == 0 || dataLen == 0 {
		return nil, 0, fmt.Errorf("WAV missing data chunk")
	}
	if audioFormat != 1 {
		return nil, 0, fmt.Errorf("WAV must be PCM (format %d)", audioFormat)
	}
	if bits != 16 {
		return nil, 0, fmt.Errorf("WAV must be 16-bit (got %d)", bits)
	}
	if channels != 1 {
		return nil, 0, fmt.Errorf("WAV must be mono (got %d ch)", channels)
	}
	if sampleRate < extAudioMinRate || sampleRate > extAudioMaxRate {
		return nil, 0, fmt.Errorf("WAV rate %d out of range", sampleRate)
	}
	end := dataOff + dataLen
	if end > len(data) {
		end = len(data)
	}
	pcm = data[dataOff:end]
	if len(pcm)%2 != 0 {
		pcm = pcm[:len(pcm)-1]
	}
	return pcm, sampleRate, nil
}

func stopMicStream() {
	ctrlMu.Lock()
	if micCancel != nil {
		micCancel()
		micCancel = nil
	}
	ctrlMu.Unlock()
	atomic.StoreInt32(&micStreaming, 0)
}
