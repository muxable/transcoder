package server

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/muxable/transcoder/api"
	"github.com/muxable/transcoder/internal/peerconnection"
	"github.com/muxable/transcoder/internal/server"
	"github.com/muxable/transcoder/pkg/transcode"
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

type TranscoderServer struct {
	api.UnimplementedTranscoderServer
	config webrtc.Configuration

	// the transcoding server likely cannot process a huge number of remote tracks
	// so there's no need to optimize this.
	sources []*Source

	// this is like the poor man's rx behavior subject.
	onTrack *sync.Cond
}

func NewTranscoderServer(config webrtc.Configuration) *TranscoderServer {
	return &TranscoderServer{
		config:  config,
		onTrack: sync.NewCond(&sync.Mutex{}),
	}
}

func (s *TranscoderServer) Signal(conn api.Transcoder_SignalServer) error {
	peerConnection, err := peerconnection.NewTranscoderPeerConnection(s.config)
	if err != nil {
		return err
	}

	synchronizer, err := transcode.NewSynchronizer()
	if err != nil {
		return err
	}

	peerConnection.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
		if pcs == webrtc.PeerConnectionStateClosed {
			synchronizer.Close()
		}
	})

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

		if err := conn.Send(&api.SignalMessage{
			Payload: &api.SignalMessage_OfferSdp{OfferSdp: offer.SDP},
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

		if err := conn.Send(&api.SignalMessage{
			Payload: &api.SignalMessage_Trickle{Trickle: string(trickle)},
		}); err != nil {
			zap.L().Error("failed to send candidate", zap.Error(err))
		}
	})

	peerConnection.OnTrack(func(tr *webrtc.TrackRemote, r *webrtc.RTPReceiver) {
		go func() {
			buf := make([]byte, 1500)
			for {
				if _, _, err := r.Read(buf); err != nil {
					return
				}
			}
		}()

		s.onTrack.L.Lock()
		s.sources = append(s.sources, &Source{
			PeerConnection: peerConnection,
			TrackRemote:    tr,
			Synchronizer:   synchronizer,
		})

		s.onTrack.Broadcast()
		s.onTrack.L.Unlock()
	})

	for {
		in, err := conn.Recv()
		if err != nil {
			zap.L().Error("failed to receive", zap.Error(err))
			return nil
		}

		switch payload := in.Payload.(type) {
		case *api.SignalMessage_OfferSdp:
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

			if err := conn.Send(&api.SignalMessage{
				Payload: &api.SignalMessage_AnswerSdp{AnswerSdp: answer.SDP},
			}); err != nil {
				return err
			}

		case *api.SignalMessage_AnswerSdp:
			if err := peerConnection.SetRemoteDescription(webrtc.SessionDescription{
				SDP:  payload.AnswerSdp,
				Type: webrtc.SDPTypeAnswer,
			}); err != nil {
				return err
			}

		case *api.SignalMessage_Trickle:
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

func (s *TranscoderServer) Transcode(ctx context.Context, request *api.TranscodeRequest) (*api.TranscodeResponse, error) {
	var matched *Source
	for matched == nil {
		s.onTrack.L.Lock()
		// find the track that matches the request.
		for i, source := range s.sources {
			tr := source.TrackRemote
			if tr.StreamID() == request.StreamId && tr.ID() == request.TrackId && tr.RID() == request.RtpStreamId {
				matched = source
				s.sources = append(s.sources[:i], s.sources[i+1:]...)
				break
			}
		}

		if matched == nil {
			s.onTrack.Wait()
		}
		s.onTrack.L.Unlock()
	}

	// tr is the remote track that matches the request.
	transcoder, err := transcode.NewTranscoder(matched.TrackRemote.Codec(),
		transcode.WithSynchronizer(matched.Synchronizer),
		transcode.ToOutputCodec(server.DefaultOutputCodecs[request.MimeType]),
		transcode.ViaGStreamerEncoder(request.GstreamerPipeline))
	if err != nil {
		return nil, err
	}

	tl, err := webrtc.NewTrackLocalStaticRTP(transcoder.OutputCodec().RTPCodecCapability, matched.TrackRemote.ID(), matched.TrackRemote.StreamID())
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			p, _, err := matched.TrackRemote.ReadRTP()
			if err != nil {
				return
			}
			if err := transcoder.WriteRTP(p); err != nil {
				return
			}
		}
	}()
	go rtpio.CopyRTP(tl, transcoder)

	rtpSender, err := matched.PeerConnection.AddTrack(tl)
	if err != nil {
		return nil, err
	}

	go func() {
		buf := make([]byte, 1500)
		for {
			if _, _, err := rtpSender.Read(buf); err != nil {
				return
			}
		}
	}()

	// respond with the RTP stream ID.
	return &api.TranscodeResponse{
		StreamId:    tl.StreamID(),
		TrackId:     tl.ID(),
		RtpStreamId: tl.RID(),
	}, nil
}
