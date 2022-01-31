package server

import (
	"fmt"
	"strings"

	"github.com/muxable/transcoder/internal/pipeline"
	"github.com/muxable/transcoder/internal/server"
	"github.com/pion/webrtc/v3"
)

type Transcoder struct {
	pipeline.ReadWritePipeline

	synchronizer     *pipeline.Synchronizer
	encodingPipeline string

	inputCodec  *webrtc.RTPCodecParameters
	outputCodec *webrtc.RTPCodecParameters
}

func NewTranscoder(from webrtc.RTPCodecParameters, options ...TranscoderOption) (*Transcoder, error) {
	t := &Transcoder{inputCodec: &from}

	for _, option := range options {
		option(t)
	}

	if t.outputCodec == nil {
		if strings.HasPrefix(from.MimeType, "video") {
			codec := server.DefaultOutputCodecs[webrtc.MimeTypeVP8]
			t.outputCodec = &codec
		} else if strings.HasPrefix(from.MimeType, "audio") {
			codec := server.DefaultOutputCodecs[webrtc.MimeTypeOpus]
			t.outputCodec = &codec
		} else {
			return nil, fmt.Errorf("unsupported codec: %s", from.MimeType)
		}
	}

	if t.synchronizer == nil {
		s, err := pipeline.NewSynchronizer()
		if err != nil {
			return nil, err
		}
		t.synchronizer = s
	}

	transcodingPipelineStr, err := server.PipelineString(from, *t.outputCodec, t.encodingPipeline)
	if err != nil {
		return nil, err
	}

	p, err := t.synchronizer.NewReadWritePipeline(transcodingPipelineStr)
	if err != nil {
		return nil, err
	}

	t.ReadWritePipeline = p

	return t, nil
}

func (t *Transcoder) OutputCodec() webrtc.RTPCodecParameters {
	return *t.outputCodec
}

type TranscoderOption func(*Transcoder)

func WithSynchronizer(s *pipeline.Synchronizer) TranscoderOption {
	return func(t *Transcoder) {
		t.synchronizer = s
	}
}

func ToOutputCodec(codec webrtc.RTPCodecParameters) TranscoderOption {
	return func(t *Transcoder) {
		t.outputCodec = &codec
	}
}

func ViaGStreamerEncoder(pipeline string) TranscoderOption {
	return func(t *Transcoder) {
		t.encodingPipeline = pipeline
	}
}
