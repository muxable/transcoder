package server

import (
	"fmt"
	"strings"

	"github.com/muxable/transcoder/internal/codecs"
	"github.com/muxable/transcoder/internal/pipeline"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type Transcoder struct {
	p pipeline.ReadWritePipeline

	synchronizer     *pipeline.Synchronizer
	encodingPipeline string

	inputCodec  *webrtc.RTPCodecParameters
	outputCodec *webrtc.RTPCodecParameters

	packetizerBuf []*rtp.Packet
	packetizer    rtp.Packetizer
}

func NewTranscoder(from webrtc.RTPCodecParameters, options ...TranscoderOption) (*Transcoder, error) {
	t := &Transcoder{inputCodec: &from}

	for _, option := range options {
		option(t)
	}

	if t.outputCodec == nil {
		if strings.HasPrefix(from.MimeType, "video") {
			codec := codecs.DefaultOutputCodecs[webrtc.MimeTypeVP8]
			t.outputCodec = &codec
		} else if strings.HasPrefix(from.MimeType, "audio") {
			codec := codecs.DefaultOutputCodecs[webrtc.MimeTypeOpus]
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

	// identify the depacketizer and packetizer
	inputParams, ok := codecs.SupportedCodecs[t.inputCodec.MimeType]
	if !ok {
		return nil, fmt.Errorf("unsupported codec %s", t.inputCodec.MimeType)
	}
	outputParams, ok := codecs.SupportedCodecs[t.outputCodec.MimeType]
	if !ok {
		return nil, fmt.Errorf("unsupported codec %s", t.outputCodec.MimeType)
	}

	if t.encodingPipeline == "" {
		t.encodingPipeline = outputParams.DefaultEncoder
	}

	elements := []string{}

	// construct the depacketizer
	// TODO: support native depacketizers
	elements = append(elements,
		fmt.Sprintf("application/x-rtp,%s,clock-rate=%d,payload=%d", inputParams.Caps, t.inputCodec.ClockRate, t.inputCodec.PayloadType),
		"rtpjitterbuffer",  // TODO: remove this if we can use appsrc maybe?
		inputParams.Depayloader)

	// construct the conversion pipeline

	if strings.HasPrefix(from.MimeType, "video") {
		elements = append(elements, "decodebin ! queue ! videoconvert ! videorate ! queue", t.encodingPipeline)
	} else if strings.HasPrefix(from.MimeType, "audio") {
		elements = append(elements, "decodebin ! queue ! audioconvert ! audioresample ! queue", t.encodingPipeline)
	}

	// construct the packetizer
	nativePayloader, ok := codecs.NativePayloader[t.outputCodec.MimeType]
	if ok {
		t.packetizer = codecs.NewTSPacketizer(1200, nativePayloader, rtp.NewRandomSequencer())
	} else {
		// use the gstreamer packetizer
		elements = append(elements, fmt.Sprintf("%s mtu=1200", outputParams.Payloader))
	}

	pstr := strings.Join(elements, " ! ")
	p, err := t.synchronizer.NewReadWritePipeline(pstr)
	if err != nil {
		return nil, err
	}

	t.p = p

	return t, nil
}

func (t *Transcoder) ReadRTP() (*rtp.Packet, error) {
	if t.packetizer == nil {
		return t.p.ReadRTP()
	}
	// pipe it through the native packetizer.
	if len(t.packetizerBuf) > 0 {
		p := t.packetizerBuf[0]
		t.packetizerBuf = t.packetizerBuf[1:]
		return p, nil
	}
	buf, err := t.p.ReadBuffer()
	if err != nil {
		return nil, err
	}
	rtpts := uint32(buf.PTS.Microseconds() * int64(t.outputCodec.ClockRate) / 1_000_000)
	t.packetizerBuf = t.packetizer.Packetize(buf.Data, rtpts)
	return t.ReadRTP()
}

func (t *Transcoder) WriteRTP(p *rtp.Packet) error {
	return t.p.WriteRTP(p)
}

func (t *Transcoder) Close() error {
	return t.p.Close()
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
