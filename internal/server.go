package internal

import (
	"encoding/json"

	transcoder "github.com/muxable/transcoder/api"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

type TranscoderServer struct {
	transcoder.UnimplementedTranscoderServer
	webrtc.Configuration

	OnPeerConnection func(pc *webrtc.PeerConnection)
}

func (s *TranscoderServer) Signal(conn transcoder.Transcoder_SignalServer) error {
	peerConnection, err := NewPeerConnection(s.Configuration)
	if err != nil {
		return err
	}

	defer peerConnection.Close()

	peerConnection.OnNegotiationNeeded(func() {
		offer, err := peerConnection.CreateOffer(nil)
		if err != nil {
			zap.L().Error("failed to create offer", zap.Error(err))
			return
		}

		if err := peerConnection.SetLocalDescription(offer); err != nil {
			zap.L().Error("failed to set local description", zap.Error(err))
			return
		}

		if err := conn.Send(&transcoder.Reply{
			Payload: &transcoder.Reply_OfferSdp{OfferSdp: offer.SDP},
		}); err != nil {
			zap.L().Error("failed to send offer", zap.Error(err))
		}
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		trickle, err := json.Marshal(candidate.ToJSON())
		if err != nil {
			zap.L().Error("failed to marshal candidate", zap.Error(err))
			return
		}

		conn.Send(&transcoder.Reply{
			Payload: &transcoder.Reply_Trickle{Trickle: string(trickle)},
		})
	})

	s.OnPeerConnection(peerConnection)

	for {
		in, err := conn.Recv()
		if err != nil {
			zap.L().Error("failed to receive", zap.Error(err))
			return nil
		}

		switch payload := in.Payload.(type) {
		case *transcoder.Request_OfferSdp:
			if err := peerConnection.SetRemoteDescription(webrtc.SessionDescription{
				SDP:  payload.OfferSdp,
				Type: webrtc.SDPTypeOffer,
			}); err != nil {
				return err
			}
			answer, err := peerConnection.CreateAnswer(nil)
			if err != nil {
				return err
			}

			if err := peerConnection.SetLocalDescription(answer); err != nil {
				return err
			}

			if err := conn.Send(&transcoder.Reply{
				Payload: &transcoder.Reply_AnswerSdp{AnswerSdp: answer.SDP},
			}); err != nil {
				return err
			}

		case *transcoder.Request_AnswerSdp:
			if err := peerConnection.SetRemoteDescription(webrtc.SessionDescription{
				SDP:  payload.AnswerSdp,
				Type: webrtc.SDPTypeAnswer,
			}); err != nil {
				return err
			}

		case *transcoder.Request_Trickle:
			candidate := webrtc.ICECandidateInit{}
			if err := json.Unmarshal([]byte(payload.Trickle), &candidate); err != nil {
				return err
			}

			if err := peerConnection.AddICECandidate(candidate); err != nil {
				return err
			}
		}
	}
}
