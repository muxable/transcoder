package transcode

import (
	"io"

	"github.com/muxable/transcoder/internal/gst"
	"github.com/pion/rtp"
)

type Source struct {
	element *gst.Element
}

func NewSource(element *gst.Element) *Source {
	return &Source{element: element}
}

func (s *Source) WriteRTP(p *rtp.Packet) error {
	buf, err := p.Marshal()
	if err != nil {
		return err
	}
	return s.element.PushBuffer(buf)
}

func (s *Source) Close() error {
	return s.element.EndOfStream()
}

type Sink struct {
	element    *gst.Element
}

func NewSink(element *gst.Element) *Sink {
	return &Sink{element:    element,}
}

func (s *Sink) ReadRTP() (*rtp.Packet, error) {
	sample, err := s.element.PullSample()
	if err != nil {
		if s.element.IsEOS() {
			return nil, io.EOF
		}
		return nil, err
	}

	p := &rtp.Packet{}
	if err := p.Unmarshal(sample.Data); err != nil {
		return nil, err
	}
	return p, nil
}
