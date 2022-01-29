package server

import (
	"github.com/muxable/transcoder/pkg/transcode"
	"github.com/pion/webrtc/v3"
)

type Source struct {
	*webrtc.PeerConnection
	*webrtc.TrackRemote
	*transcode.Synchronizer
}
