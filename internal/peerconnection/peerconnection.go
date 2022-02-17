// peerconnection is a package that configures a peerconnection with additional video types supported.
package peerconnection

import (
	"strings"

	"github.com/muxable/transcoder/internal/codecs"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
)

// NewTranscoderPeerConnection creates a new PeerConnection with additional video types supported.
func NewTranscoderPeerConnection(configuration webrtc.Configuration) (*webrtc.PeerConnection, error) {
	m := &webrtc.MediaEngine{}

	for _, codec := range codecs.DefaultOutputCodecs {
		if strings.HasPrefix(codec.MimeType, "video/") {
			if err := m.RegisterCodec(codec, webrtc.RTPCodecTypeVideo); err != nil {
				return nil, err
			}
		} else if strings.HasPrefix(codec.MimeType, "audio/") {
			if err := m.RegisterCodec(codec, webrtc.RTPCodecTypeAudio); err != nil {
				return nil, err
			}
		}
	}

	i := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		return nil, err
	}

	return webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i)).NewPeerConnection(configuration)
}
