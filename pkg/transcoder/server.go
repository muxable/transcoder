package transcoder

import (
	"context"
	"log"
	"sync"

	"github.com/muxable/transcoder/api"
	"github.com/muxable/transcoder/internal/peerconnection"
	"github.com/pion/rtcp"
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

type Source struct {
	*webrtc.PeerConnection
	*webrtc.TrackRemote
	*Transcoder
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

	transcoder, err := NewTranscoder()
	if err != nil {
		return err
	}

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
			Transcoder:     transcoder,
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

	inCodec := matched.TrackRemote.Codec()

	log.Printf("input %v", inCodec)

	builder, err := NewPipelineBuilder(inCodec.MimeType, request.MimeType, request.GstreamerPipeline)
	if err != nil {
		return nil, err
	}

	if inCodec.MimeType == webrtc.MimeTypeH265 {
		inCodec.SDPFmtpLine = "sprop-vps=QAEMAf//AWAAAAMAkAAAAwAAAwA/ugJA,sprop-sps=QgEBAWAAAAMAkAAAAwAAAwA/oAoIDxZbpKTC//AAEAAQEAAAAwAQAAADAeCA,sprop-pps=RAHAcYES"
	}

	pipeline, err := matched.Transcoder.NewReadWritePipeline(&inCodec, builder)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			p, _, err := matched.TrackRemote.ReadRTP()
			if err != nil {
				zap.L().Error("failed to read rtp", zap.Error(err))
				return
			}
			if err := pipeline.WriteRTP(p); err != nil {
				zap.L().Warn("failed to write rtp", zap.Error(err))
				return
			}
		}
	}()

	pipeline.OnUpstreamForceKeyUnit(func() {
		if err := matched.PeerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(matched.TrackRemote.SSRC())}}); err != nil {
			zap.L().Warn("failed to write rtcp", zap.Error(err))
		}
	})

	outCodec, err := pipeline.Codec()
	if err != nil {
		return nil, err
	}

	log.Printf("negotiated %v", outCodec)

	tl, err := webrtc.NewTrackLocalStaticRTP(outCodec.RTPCodecCapability, matched.TrackRemote.ID(), matched.TrackRemote.StreamID())
	if err != nil {
		return nil, err
	}

	go rtpio.CopyRTP(tl, pipeline)

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
