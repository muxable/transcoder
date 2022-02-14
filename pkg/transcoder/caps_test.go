package transcoder

import (
	"reflect"
	"testing"

	"github.com/pion/webrtc/v3"
	"go.uber.org/goleak"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

var flagtests = []struct {
	caps  string
	codec *webrtc.RTPCodecParameters
}{
	{
		"application/x-rtp, media=(string)video, payload=(int)100, clock-rate=(int)90000, encoding-name=(string)VP8",
		&webrtc.RTPCodecParameters{
			PayloadType: 100,
			RTPCodecCapability: webrtc.RTPCodecCapability{
				MimeType:  webrtc.MimeTypeVP8,
				ClockRate: 90000,
			},
		},
	},
	{
		"application/x-rtp, media=(string)audio, payload=(int)111, clock-rate=(int)48000, encoding-name=(string)PCMA, encoding-params=(string)2",
		&webrtc.RTPCodecParameters{
			PayloadType: 111,
			RTPCodecCapability: webrtc.RTPCodecCapability{
				MimeType:  webrtc.MimeTypePCMA,
				ClockRate: 48000,
				Channels:  2,
			},
		},
	},
	{
		"application/x-rtp, media=(string)video, payload=(int)111, clock-rate=(int)90000, encoding-name=(string)H264, packetization-mode=(string)1, profile-level-id=(string)42e01f",
		&webrtc.RTPCodecParameters{
			PayloadType: 111,
			RTPCodecCapability: webrtc.RTPCodecCapability{
				MimeType:    webrtc.MimeTypeH264,
				ClockRate:   90000,
				SDPFmtpLine: "packetization-mode=1;profile-level-id=42e01f",
			},
		},
	},
}

func TestCaps_ToCaps(t *testing.T) {
	logger := zaptest.NewLogger(t)
	defer logger.Sync()
	undo := zap.ReplaceGlobals(logger)
	defer undo()

	defer goleak.VerifyNone(t)
	
	for _, tt := range flagtests {
		t.Run(tt.caps, func(t *testing.T) {
			caps, err := CapsFromRTPCodecParameters(tt.codec)
			if err != nil {
				t.Error(err)
				return
			}
			got := caps.String()
			if got != tt.caps {
				t.Errorf("got %s, want %s", got, tt.caps)
			}
		})
	}
}

func TestCaps_ToCodec(t *testing.T) {
	logger := zaptest.NewLogger(t)
	defer logger.Sync()
	undo := zap.ReplaceGlobals(logger)
	defer undo()

	defer goleak.VerifyNone(t)

	for _, tt := range flagtests {
		t.Run(tt.caps, func(t *testing.T) {
			caps := CapsFromString(tt.caps)
			codec, err := caps.RTPCodecParameters()
			if err != nil {
				t.Error(err)
				return
			}
			if !reflect.DeepEqual(codec, tt.codec) {
				t.Errorf("got %#v, want %#v", codec, tt.codec)
			}
		})
	}
}
