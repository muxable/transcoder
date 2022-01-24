package transcode

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/muxable/transcoder/internal/gst"
	"github.com/muxable/transcoder/internal/server"
	"github.com/pion/rtp"
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
)

type Transcoder struct {
	sink rtpio.RTPWriteCloser
	source rtpio.RTPReader

	synchronizer     *Synchronizer
	encodingPipeline string

	bin *gst.Bin

	outputCodec *webrtc.RTPCodecParameters
}

func NewTranscoder(from webrtc.RTPCodecParameters, options ...TranscoderOption) (*Transcoder, error) {
	t := &Transcoder{}

	for _, option := range options {
		option(t)
	}

	if t.outputCodec == nil {
		if strings.HasPrefix(from.MimeType, "video") {
			codec := server.DefaultOutputCodecs[webrtc.MimeTypeH264]
			t.outputCodec = &codec
		} else if strings.HasPrefix(from.MimeType, "audio") {
			codec := server.DefaultOutputCodecs[webrtc.MimeTypeOpus]
			t.outputCodec = &codec
		} else {
			return nil, fmt.Errorf("unsupported codec: %s", from.MimeType)
		}
	}

	transcodingPipelineStr, err := server.PipelineString(from, *t.outputCodec, t.encodingPipeline)
	if err != nil {
		return nil, err
	}

	log.Printf("%v", transcodingPipelineStr)

	bin, err := gst.ParseBinFromDescription(transcodingPipelineStr)
	if err != nil {
		return nil, err
	}

	source := bin.GetByName("source")
	sink := bin.GetByName("sink")
	if source != nil {
		t.sink = source
	}
	if sink != nil {
		t.source = sink
	}
	if t.synchronizer == nil {
		pipeline, err := gst.NewPipeline()
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

func (t *Transcoder) OutputCodec() webrtc.RTPCodecParameters {
	return *t.outputCodec
}

func (t *Transcoder) ReadRTP() (*rtp.Packet, error) {
	if t.source == nil {
		return nil, io.EOF
	}
	p, err := t.source.ReadRTP()
	if err != nil {
		return nil, err
	}
	return p, nil
}

// WriteRTP writes the RTP packet to the writer if it's present.
func (t *Transcoder) WriteRTP(p *rtp.Packet) error {
	if t.sink == nil {
		return nil
	}
	return t.sink.WriteRTP(p)
}

func (t *Transcoder) Close() error {
	t.bin.SetState(gst.StateNull)

	if t.synchronizer != nil {
		t.synchronizer.element.Remove(&t.bin.Element)
	}

	if t.sink == nil {
		return nil
	}

	if err := t.sink.Close(); err != nil {
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
