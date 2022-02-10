package server

import (
	"context"
	"sync"
	"time"

	"github.com/muxable/transcoder/api"
	"github.com/muxable/transcoder/internal/codecs"
	"github.com/muxable/transcoder/internal/peerconnection"
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

type Source struct {
	*webrtc.PeerConnection
	*webrtc.TrackRemote
}

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

	signaller := peerconnection.Negotiate(peerConnection)

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
		})

		s.onTrack.Broadcast()
		s.onTrack.L.Unlock()
	})

	go func() {
		for {
			signal, err := signaller.ReadSignal()
			if err != nil {
				zap.L().Error("failed to read signal", zap.Error(err))
				return
			}
			if err := conn.Send(signal); err != nil {
				zap.L().Error("failed to send signal", zap.Error(err))
				return
			}
		}
	}()

	for {
		in, err := conn.Recv()
		if err != nil {
			zap.L().Error("failed to receive", zap.Error(err))
			return nil
		}

		if err := signaller.WriteSignal(in); err != nil {
			zap.L().Error("failed to write signal", zap.Error(err))
			return nil
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

	options := []TranscoderOption{}
	if request.MimeType != "" {
		options = append(options, ToOutputCodec(codecs.DefaultOutputCodecs[request.MimeType]))
	}

	// tr is the remote track that matches the request.
	transcoder, err := NewTranscoder(matched.TrackRemote.Codec(), options...)
	if err != nil {
		return nil, err
	}
	
	time.Sleep(1 * time.Second)

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
