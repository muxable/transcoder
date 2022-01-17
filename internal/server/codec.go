package server

import (
	"fmt"

	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
)

type OutputCodec struct {
	webrtc.RTPCodecCapability
	rtp.Payloader
	GStreamerEncoder string
}

func ResolveOutputCodec(tr *webrtc.TrackRemote, mimeType, pipelineStr string) (*OutputCodec, error) {
	if mimeType == "" {
		// use the default output codec for the track remote kind.
		switch tr.Kind() {
		case webrtc.RTPCodecTypeVideo:
			return &OutputCodec{
				RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000},
				Payloader:          &codecs.H264Payloader{},
				GStreamerEncoder:   "x264enc speed-preset=ultrafast tune=zerolatency key-int-max=20",
			}, nil
		case webrtc.RTPCodecTypeAudio:
			return &OutputCodec{
				RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000},
				Payloader:          &codecs.OpusPayloader{},
				GStreamerEncoder:   "opusenc",
			}, nil
		}
		return nil, fmt.Errorf("unsupported track remote kind %s", tr.Kind())
	}
	output := &OutputCodec{}
	switch mimeType {
	case webrtc.MimeTypeH264:
		output.RTPCodecCapability = webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000}
		output.Payloader = &codecs.H264Payloader{}
		output.GStreamerEncoder = "x264enc speed-preset=ultrafast tune=zerolatency key-int-max=20"
	case webrtc.MimeTypeVP8:
		output.RTPCodecCapability = webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}
		output.Payloader = &codecs.VP8Payloader{}
		output.GStreamerEncoder = "vp8enc deadline=1"
	case webrtc.MimeTypeOpus:
		output.RTPCodecCapability = webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000}
		output.Payloader = &codecs.OpusPayloader{}
		output.GStreamerEncoder = "opusenc"
	default:
		return nil, fmt.Errorf("unsupported codec %s", mimeType)
	}
	if pipelineStr != "" {
		output.GStreamerEncoder = pipelineStr
	}
	return output, nil
}
