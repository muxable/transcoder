package internal

import (
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
)

// NewPeerConnection creates a new PeerConnection with additional video types supported.
func NewPeerConnection(configuration webrtc.Configuration) (*webrtc.PeerConnection, error) {
	m := &webrtc.MediaEngine{}
	if err := m.RegisterDefaultCodecs(); err != nil {
		return nil, err
	}

	// signal support for h265 until pion supports it.
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{"video/h265", 90000, 0, "", []webrtc.RTCPFeedback{{"goog-remb", ""}, {"ccm", "fir"}, {"nack", ""}, {"nack", "pli"}}},
		PayloadType:        103,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		return nil,  err
	}

	i := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		return nil,  err
	}

	return webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i)).NewPeerConnection(configuration)
}