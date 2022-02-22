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

	writers []io.Writer
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
	for _, w := range t.writers {
		w.Write(buf)
	}
	return len(buf), nil
}

func (t *TrackLocal) WriteRTP(p *rtp.Packet) error {
	buf := make([]byte, 1500)
	if err := p.Unmarshal(buf); err != nil {
		return err
	}
	n, err := t.Write(buf)
	if n < p.MarshalSize() {
		return io.ErrShortWrite
	}
	return err
}

var _ io.Writer = (*TrackLocal)(nil)
var _ rtpio.RTPWriter = (*TrackLocal)(nil)
