package transcode

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/muxable/transcoder/internal/server"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/tinyzimmer/go-gst/gst"
	"github.com/tinyzimmer/go-gst/gst/app"
)

type Transcoder struct {
	sink   *app.Source
	source *app.Sink

	synchronizer     *Synchronizer
	encodingPipeline string

	bin *gst.Bin

	inputCodec  *webrtc.RTPCodecParameters
	outputCodec *webrtc.RTPCodecParameters

	t0                uint32
	previousTimestamp uint32
	rollovers         uint32
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

	transcodingPipelineStr, err := server.PipelineString(from, *t.outputCodec, t.encodingPipeline)
	if err != nil {
		return nil, err
	}

	bin, err := gst.NewBinFromString(transcodingPipelineStr, false)
	if err != nil {
		return nil, err
	}

	src, err := bin.GetElementByName("source")
	if err != nil {
		return nil, err
	}
	sink, err := bin.GetElementByName("sink")
	if err != nil {
		return nil, err
	}
	t.sink = app.SrcFromElement(src)
	t.source = app.SinkFromElement(sink)
	if t.synchronizer == nil {
		pipeline, err := gst.NewPipeline("")
		if err != nil {
			return nil, err
		}
		pipeline.Add(bin.Element)
		pipeline.SetState(gst.StatePlaying)

		t.bin = pipeline.Bin
	} else {
		t.synchronizer.element.Add(bin.Element)
		t.bin = bin
	}

	bin.SetState(gst.StatePlaying)

	return t, nil
}

func (t *Transcoder) OutputCodec() webrtc.RTPCodecParameters {
	return *t.outputCodec
}

func (t *Transcoder) ReadRTP() (*rtp.Packet, error) {
	sample := t.source.PullSample()
	if sample == nil {
		return nil, io.EOF
	}
	buffer := sample.GetBuffer()
	if buffer == nil {
		return nil, fmt.Errorf("no buffer in sample")
	}
	buf := buffer.Map(gst.MapRead).Bytes()
	defer buffer.Unmap()
	p := &rtp.Packet{}
	if err := p.Unmarshal(buf); err != nil {
		return nil, err
	}
	return p, nil
}

// WriteRTP writes the RTP packet to the writer if it's present.
func (t *Transcoder) WriteRTP(p *rtp.Packet) error {
	if t.sink == nil {
		return nil
	}
	buf, err := p.Marshal()
	if err != nil {
		return err
	}

	buffer := gst.NewBufferWithSize(int64(len(buf)))

	buffer.Map(gst.MapWrite).WriteData(buf)
	buffer.Unmap()

	if t.t0 == 0 {
		t.t0 = p.Timestamp
	}
	if t.previousTimestamp > 1<<31 && p.Timestamp < 1<<31 {
		t.rollovers++
	}
	t.previousTimestamp = p.Timestamp
	trueTs := int64(p.Timestamp) - int64(t.t0) + int64(t.rollovers) * (1<<32)
	pts := (time.Duration(trueTs) * time.Second) / time.Duration(t.inputCodec.ClockRate)
	buffer.SetPresentationTimestamp(pts)

	if r := t.sink.PushBuffer(buffer); r != gst.FlowOK {
		return fmt.Errorf("failed to push buffer: %v", r)
	}
	return nil
}

func (t *Transcoder) Close() error {
	t.bin.SetState(gst.StateNull)

	if t.synchronizer != nil {
		t.synchronizer.element.Remove(t.bin.Element)
	}

	if err := t.sink.EndStream(); err != gst.FlowEOS {
		return errors.New("failed to end stream")
	}

	t.source.Unref()
	t.sink.Unref()

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
