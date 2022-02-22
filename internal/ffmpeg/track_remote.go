package ffmpeg

import (
	"io"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

// TrackRemote is a remote track.
type TrackRemote struct {
	readCh chan []byte
	sessionName string
	payloadType webrtc.PayloadType
	kind        webrtc.RTPCodecType
	codec       webrtc.RTPCodecParameters
}

// read-only accessors.

func (t *TrackRemote) SessionName() string {
	return t.sessionName
}

func (t *TrackRemote) PayloadType() webrtc.PayloadType {
	return t.payloadType
}

func (t *TrackRemote) Kind() webrtc.RTPCodecType {
	return t.kind
}

func (t *TrackRemote) Codec() webrtc.RTPCodecParameters {
	return t.codec
}

func (t *TrackRemote) ReadRTP() (*rtp.Packet, error) {
	buf, ok := <-t.readCh
	if !ok {
		return nil, io.EOF
	}
	p := &rtp.Packet{}
	if err := p.Unmarshal(buf); err != nil {
		return nil, err
	}
	return p, nil
}