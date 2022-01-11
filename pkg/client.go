package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	transcoder "github.com/muxable/transcoder/api"
	"github.com/muxable/transcoder/internal"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type TranscoderClient struct {
	sync.Mutex

	peerConnection *webrtc.PeerConnection
	grpcConn       *grpc.ClientConn
	promises       map[string]chan *webrtc.TrackRemote
}

func NewTranscoderAPIClient(conn *grpc.ClientConn) (*TranscoderClient, error) {
	peerConnection, err := internal.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		return nil, err
	}

	client, err := transcoder.NewTranscoderClient(conn).Signal(context.Background())
	if err != nil {
		return nil, err
	}

	c := &TranscoderClient{
		peerConnection: peerConnection,
		grpcConn:       conn,
		promises:       make(map[string]chan *webrtc.TrackRemote),
	}

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

		if err := client.Send(&transcoder.Request{
			Payload: &transcoder.Request_OfferSdp{OfferSdp: offer.SDP},
		}); err != nil {
			zap.L().Error("failed to send offer", zap.Error(err))
			return
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

		client.Send(&transcoder.Request{
			Payload: &transcoder.Request_Trickle{Trickle: string(trickle)},
		})
	})

	peerConnection.OnTrack(func(tr *webrtc.TrackRemote, r *webrtc.RTPReceiver) {
		c.Lock()
		defer c.Unlock()

		if promise, ok := c.promises[tr.ID()]; ok {
			promise <- tr
			delete(c.promises, tr.ID())
		} else {
			zap.L().Error("received track without promise", zap.String("track", tr.ID()), zap.String("promises", fmt.Sprintf("%v", c.promises)))
		}
	})

	go func() {
		defer peerConnection.Close()
		for {
			in, err := client.Recv()
			if err != nil {
				zap.L().Error("failed to receive", zap.Error(err))
				return
			}

			switch payload := in.Payload.(type) {
			case *transcoder.Reply_OfferSdp:
				if err := peerConnection.SetRemoteDescription(webrtc.SessionDescription{
					SDP:  payload.OfferSdp,
					Type: webrtc.SDPTypeOffer,
				}); err != nil {
					zap.L().Error("failed to set remote description", zap.Error(err))
					break
				}
				answer, err := peerConnection.CreateAnswer(nil)
				if err != nil {
					zap.L().Error("failed to create answer", zap.Error(err))
					break
				}

				if err := peerConnection.SetLocalDescription(answer); err != nil {
					zap.L().Error("failed to set local description", zap.Error(err))
					break
				}

				if err := client.Send(&transcoder.Request{
					Payload: &transcoder.Request_AnswerSdp{AnswerSdp: answer.SDP},
				}); err != nil {
					zap.L().Error("failed to send answer", zap.Error(err))
					break
				}

			case *transcoder.Reply_AnswerSdp:
				if err := peerConnection.SetRemoteDescription(webrtc.SessionDescription{
					SDP:  payload.AnswerSdp,
					Type: webrtc.SDPTypeAnswer,
				}); err != nil {
					zap.L().Error("failed to set remote description", zap.Error(err))
					break
				}

			case *transcoder.Reply_Trickle:
				candidate := webrtc.ICECandidateInit{}
				if err := json.Unmarshal([]byte(payload.Trickle), &candidate); err != nil {
					zap.L().Error("failed to unmarshal candidate", zap.Error(err))
					break
				}

				if err := peerConnection.AddICECandidate(candidate); err != nil {
					zap.L().Error("failed to add candidate", zap.Error(err))
					break
				}
			}
		}
	}()

	return c, nil
}

func (c *TranscoderClient) Transcode(tl webrtc.TrackLocal) (*webrtc.TrackRemote, error) {
	if _, err := c.peerConnection.AddTrack(tl); err != nil {
		return nil, err
	}

	c.Lock()
	promise := make(chan *webrtc.TrackRemote)
	c.promises[tl.ID()] = promise
	c.Unlock()

	return <-promise, nil
}

// Close closes the underlying peer connection.
func (c *TranscoderClient) Close() error {
	if err := c.grpcConn.Close(); err != nil {
		return err
	}
	return c.peerConnection.Close()
}
