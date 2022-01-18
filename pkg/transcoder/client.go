package transcoder

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/muxable/transcoder/api"
	"github.com/muxable/transcoder/internal/peerconnection"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Client struct {
	sync.Mutex

	ctx            context.Context
	peerConnection *webrtc.PeerConnection
	grpcClient     api.TranscoderClient
	promises       map[string]chan *webrtc.TrackRemote
}

func NewClient(ctx context.Context, conn *grpc.ClientConn) (*Client, error) {
	peerConnection, err := peerconnection.NewTranscoderPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		return nil, err
	}

	client := api.NewTranscoderClient(conn)

	signal, err := client.Signal(ctx)
	if err != nil {
		return nil, err
	}

	c := &Client{
		ctx:            ctx,
		peerConnection: peerConnection,
		grpcClient:     client,
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

		if err := signal.Send(&api.SignalMessage{
			Payload: &api.SignalMessage_OfferSdp{OfferSdp: offer.SDP},
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

		if err := signal.Send(&api.SignalMessage{
			Payload: &api.SignalMessage_Trickle{Trickle: string(trickle)},
		}); err != nil {
			zap.L().Error("failed to send candidate", zap.Error(err))
		}
	})

	peerConnection.OnTrack(func(tr *webrtc.TrackRemote, r *webrtc.RTPReceiver) {
		c.Lock()
		defer c.Unlock()

		go func() {
			buf := make([]byte, 1500)
			for {
				if _, _, err := r.Read(buf); err != nil {
					return
				}
			}
		}()

		// By contract, the transcoding server guarantees a globally unique RID for each track.
		if promise, ok := c.promises[fmt.Sprintf("%s:%s:%s", tr.StreamID(), tr.ID(), tr.RID())]; ok {
			promise <- tr
			delete(c.promises, tr.RID())
		} else {
			zap.L().Error("received track without promise", zap.String("track", tr.RID()), zap.String("promises", fmt.Sprintf("%v", c.promises)))
		}
	})

	go func() {
		defer peerConnection.Close()
		for {
			in, err := signal.Recv()
			if err != nil {
				zap.L().Error("failed to receive", zap.Error(err))
				return
			}

			switch payload := in.Payload.(type) {
			case *api.SignalMessage_OfferSdp:
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

				if err := signal.Send(&api.SignalMessage{
					Payload: &api.SignalMessage_AnswerSdp{AnswerSdp: answer.SDP},
				}); err != nil {
					zap.L().Error("failed to send answer", zap.Error(err))
					break
				}

			case *api.SignalMessage_AnswerSdp:
				if err := peerConnection.SetRemoteDescription(webrtc.SessionDescription{
					SDP:  payload.AnswerSdp,
					Type: webrtc.SDPTypeAnswer,
				}); err != nil {
					zap.L().Error("failed to set remote description", zap.Error(err))
					break
				}

			case *api.SignalMessage_Trickle:
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

type TranscodeOption func(*api.TranscodeRequest)

func (c *Client) Transcode(tl webrtc.TrackLocal, options ...TranscodeOption) (*webrtc.TrackRemote, error) {
	rtpSender, err := c.peerConnection.AddTrack(tl)
	if err != nil {
		return nil, err
	}
	request := &api.TranscodeRequest{
		StreamId: tl.StreamID(),
		TrackId:  tl.ID(),
	}

	for _, option := range options {
		option(request)
	}

	c.Lock()
	response, err := c.grpcClient.Transcode(c.ctx, request)
	if err != nil {
		return nil, err
	}
	promise := make(chan *webrtc.TrackRemote)
	c.promises[fmt.Sprintf("%s:%s:%s", response.StreamId, response.TrackId, response.RtpStreamId)] = promise
	c.Unlock()

	go func() {
		buf := make([]byte, 1500)
		for {
			if _, _, err := rtpSender.Read(buf); err != nil {
				return
			}
		}
	}()

	return <-promise, nil
}

func ToMimeType(mimeType string) TranscodeOption {
	return func(request *api.TranscodeRequest) {
		request.MimeType = mimeType
	}
}
