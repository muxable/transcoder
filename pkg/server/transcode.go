package server

import (
	"fmt"
	"strings"

	"github.com/muxable/transcoder/internal/av"
	"github.com/muxable/transcoder/internal/codecs"
	"github.com/pion/rtp"
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
)

type Transcoder struct {
	rtpio.RTPReadWriteCloser

	inputCodec  *webrtc.RTPCodecParameters
	outputCodec *webrtc.RTPCodecParameters

	depacketizer rtp.Depacketizer
	packetizer   rtp.Payloader
}

func NewTranscoder(from webrtc.RTPCodecParameters, options ...TranscoderOption) (*Transcoder, error) {
	t := &Transcoder{inputCodec: &from}

	for _, option := range options {
		option(t)
	}

	if t.outputCodec == nil {
		if strings.HasPrefix(from.MimeType, "video") {
			codec := codecs.DefaultOutputCodecs[webrtc.MimeTypeH264]
			t.outputCodec = &codec
		} else if strings.HasPrefix(from.MimeType, "audio") {
			codec := codecs.DefaultOutputCodecs[webrtc.MimeTypeOpus]
			t.outputCodec = &codec
		} else {
			return nil, fmt.Errorf("unsupported codec: %s", from.MimeType)
		}
	}

	// construct the ffmpeg pipeline.
	tc, err := av.NewTranscoder(from, t.outputCodec.RTPCodecCapability)
	if err != nil {
		return nil, err
	}

	t.RTPReadWriteCloser = tc

	return t, nil
}

func (t *Transcoder) OutputCodec() webrtc.RTPCodecParameters {
	return *t.outputCodec
}

type TranscoderOption func(*Transcoder)

func ToOutputCodec(codec webrtc.RTPCodecParameters) TranscoderOption {
	return func(t *Transcoder) {
		t.outputCodec = &codec
	}
}
