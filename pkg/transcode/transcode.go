package transcode

import (
	"fmt"
	"log"
	"strings"

	"github.com/muxable/rtpio/pkg/rtpio"
	"github.com/muxable/transcoder/internal/gst"
	"github.com/muxable/transcoder/internal/server"
	"github.com/pion/webrtc/v3"
)

type Transcoder struct {
	rtpio.RTPWriteCloser
	rtpio.RTPReader

	synchronizer     *Synchronizer
	outputMimeType   string
	encodingPipeline string

	bin *gst.Bin

	outputCodec webrtc.RTPCodecCapability
}

func NewTranscoder(from webrtc.RTPCodecCapability, options ...TranscoderOption) (*Transcoder, error) {
	t := &Transcoder{}

	if strings.HasPrefix(from.MimeType, "video") {
		t.outputMimeType = "video"
	} else if strings.HasPrefix(from.MimeType, "audio") {
		t.outputMimeType = "audio"
	}

	for _, option := range options {
		option(t)
	}

	outputCodec, ok := server.DefaultOutputCodecs[t.outputMimeType]
	if !ok {
		return nil, fmt.Errorf("unsupported output codec: %s", t.outputMimeType)
	}

	transcodingPipelineStr, err := server.PipelineString(from, outputCodec, t.encodingPipeline)
	if err != nil {
		return nil, err
	}

	log.Printf("creating pipeline: %s", transcodingPipelineStr)

	bin, err := gst.ParseBinFromDescription(transcodingPipelineStr)
	if err != nil {
		return nil, err
	}

	t.RTPWriteCloser = NewSource(bin.GetByName("source"))
	t.RTPReader = NewSink(bin.GetByName("sink"))
	t.outputCodec = outputCodec
	if t.synchronizer == nil {
		pipeline, err := gst.PipelineNew()
		if err != nil {
			return nil, err
		}
		pipeline.Add(&bin.Element)
		pipeline.SetState(gst.StatePlaying)

		t.bin = &pipeline.Bin
	} else {
		t.synchronizer.element.Add(&bin.Element)
		t.bin = bin
	}

	bin.SetState(gst.StatePlaying)

	return t, nil
}

func (t *Transcoder) OutputCodec() webrtc.RTPCodecCapability {
	return t.outputCodec
}

func (t *Transcoder) Close() error {
	t.bin.SetState(gst.StateNull)

	if t.synchronizer != nil {
		t.synchronizer.element.Remove(&t.bin.Element)
	}

	if err := t.RTPWriteCloser.Close(); err != nil {
		return err
	}

	return nil
}

type TranscoderOption func(*Transcoder)

func WithSynchronizer(s *Synchronizer) TranscoderOption {
	return func(t *Transcoder) {
		t.synchronizer = s
	}
}

func ToMimeType(mimeType string) TranscoderOption {
	return func(t *Transcoder) {
		t.outputMimeType = mimeType
	}
}

func ViaGStreamerEncoder(pipeline string) TranscoderOption {
	return func(t *Transcoder) {
		t.encodingPipeline = pipeline
	}
}
