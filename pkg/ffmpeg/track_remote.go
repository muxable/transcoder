package ffmpeg

import (
	"io"

	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
)

// TrackRemote is a remote track.
type TrackRemote struct {
	io.Reader
	rtpio.RTPReader
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
