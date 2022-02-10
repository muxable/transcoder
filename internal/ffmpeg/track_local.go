package ffmpeg

import (
	"io"
	"strings"

	"github.com/pion/rtp"
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
)

// TrackLocal is a remote track.
type TrackLocal struct {
	sessionName string
	codec       webrtc.RTPCodecParameters

	Writers []io.Writer
}

func NewTrackLocal(codec webrtc.RTPCodecParameters, sessionName string) (*TrackLocal, error) {
	return &TrackLocal{
		sessionName: sessionName,
		codec:       codec,
	}, nil
}

// read-only accessors.

func (t *TrackLocal) SessionName() string {
	return t.sessionName
}

func (t *TrackLocal) PayloadType() webrtc.PayloadType {
	return t.codec.PayloadType
}

func (s *TrackLocal) Kind() webrtc.RTPCodecType {
	switch {
	case strings.HasPrefix(s.codec.MimeType, "audio/"):
		return webrtc.RTPCodecTypeAudio
	case strings.HasPrefix(s.codec.MimeType, "video/"):
		return webrtc.RTPCodecTypeVideo
	default:
		return webrtc.RTPCodecType(0)
	}
}

func (t *TrackLocal) Codec() webrtc.RTPCodecParameters {
	return t.codec
}

func (t *TrackLocal) Write(buf []byte) (int, error) {
	for _, w := range t.Writers {
		if _, err := w.Write(buf); err != nil {
			panic(err)
		}
	}
	return len(buf), nil
}

func (t *TrackLocal) WriteRTP(p *rtp.Packet) error {
	buf, err := p.Marshal()
	if err != nil {
		return err
	}
	n, err := t.Write(buf)
	if n < len(buf) {
		return io.ErrShortWrite
	}
	return err
}

var _ io.Writer = (*TrackLocal)(nil)
var _ rtpio.RTPWriter = (*TrackLocal)(nil)
