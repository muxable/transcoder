package transcoder

import (
	"errors"
	"log"
	"strings"
	"sync"

	"github.com/muxable/signal/pkg/signal"
	"github.com/muxable/transcoder/api"
	"github.com/muxable/transcoder/internal/codecs"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

type Source struct {
	*webrtc.PeerConnection
	*webrtc.TrackRemote

	sinks []rtpio.RTPWriteCloser
}

func (s *Source) addSink(sink rtpio.RTPWriteCloser) {
	s.sinks = append(s.sinks, sink)
	if len(s.sinks) == 1 {
		go func() {
			for {
				p, _, err := s.TrackRemote.ReadRTP()
				if err != nil {
					for _, sink := range s.sinks {
						sink.Close()
					}
					return
				}
				for _, sink := range s.sinks {
					if err := sink.WriteRTP(p); err != nil {

					}
				}
			}
		}()
	}
}

type TranscoderServer struct {
	api.UnimplementedTranscoderServer
	config webrtc.Configuration

	// the transcoding server likely cannot process a huge number of remote tracks
	// so there's no need to optimize this.
	sources []*Source

	// this is like the poor man's rx behavior subject.
	onTrack *sync.Cond

	transcoders map[string]*Transcoder
}

func NewTranscoderServer(config webrtc.Configuration) *TranscoderServer {
	return &TranscoderServer{
		config:      config,
		onTrack:     sync.NewCond(&sync.Mutex{}),
		transcoders: make(map[string]*Transcoder),
	}
}

func (s *TranscoderServer) transcoderFor(streamID string) (*Transcoder, error) {
	t, ok := s.transcoders[streamID]
	if ok {
		return t, nil
	}

	t, err := NewTranscoder()
	if err != nil {
		return nil, err
	}
	s.transcoders[streamID] = t
	return t, nil
}

func (s *TranscoderServer) Publish(conn api.Transcoder_PublishServer) error {
	m := &webrtc.MediaEngine{}

	// signal that we accept all the codecs.
	for _, codec := range codecs.DefaultOutputCodecs {
		if strings.HasPrefix(codec.MimeType, "video/") {
			if err := m.RegisterCodec(codec, webrtc.RTPCodecTypeVideo); err != nil {
				return err
			}
		} else if strings.HasPrefix(codec.MimeType, "audio/") {
			if err := m.RegisterCodec(codec, webrtc.RTPCodecTypeAudio); err != nil {
				return err
			}
		}
	}

	i := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		return err
	}

	peerConnection, err := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i)).NewPeerConnection(s.config)
	if err != nil {
		return err
	}

	signaller := signal.Negotiate(peerConnection)

	peerConnection.OnTrack(func(tr *webrtc.TrackRemote, r *webrtc.RTPReceiver) {
		go func() {
			buf := make([]byte, 1500)
			for {
				if _, _, err := r.Read(buf); err != nil {
					return
				}
			}
		}()

		source := &Source{
			PeerConnection: peerConnection,
			TrackRemote:    tr,
		}

		s.onTrack.L.Lock()
		s.sources = append(s.sources, source)
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
		signal, err := conn.Recv()
		if err != nil {
			zap.L().Error("failed to receive", zap.Error(err))
			return nil
		}

		if err := signaller.WriteSignal(signal); err != nil {
			zap.L().Error("failed to write signal", zap.Error(err), zap.String("signal", signal.String()))
			return nil
		}
	}
}

func (s *TranscoderServer) Subscribe(conn api.Transcoder_SubscribeServer) error {
	request, err := conn.Recv()
	if err != nil {
		return err
	}

	op, ok := request.Operation.(*api.SubscribeRequest_Request)
	if !ok {
		return errors.New("unexpected signal")
	}

	var matched *Source
	for matched == nil {
		s.onTrack.L.Lock()
		// find the track that matches the request.
		for i, source := range s.sources {
			tr := source.TrackRemote
			if tr.StreamID() == op.Request.StreamId && tr.ID() == op.Request.TrackId && tr.RID() == op.Request.RtpStreamId {
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

	builder, err := NewPipelineBuilder(matched.TrackRemote.Kind(), op.Request.MimeType, op.Request.GstreamerPipeline)
	if err != nil {
		return err
	}

	inCodec := matched.TrackRemote.Codec()
	tc, err := s.transcoderFor(matched.TrackRemote.StreamID())
	if err != nil {
		return err
	}
	pipeline, err := tc.NewReadWritePipeline(&inCodec, builder)
	if err != nil {
		return err
	}

	pipeline.OnUpstreamForceKeyUnit(func() {
		if err := matched.PeerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(matched.TrackRemote.SSRC())}}); err != nil {
			zap.L().Warn("failed to write rtcp", zap.Error(err))
		}
	})

	matched.addSink(pipeline)

	buf := make(chan *rtp.Packet, 1000) // this is to resolve the cyclic dependency.

	go func() {
		for {
			p, err := pipeline.ReadRTP()
			if err != nil {
				close(buf)
				return
			}
			buf <- p
		}
	}()

	outCodec, err := pipeline.Codec()
	if err != nil {
		return err
	}

	log.Printf("negotiated %v", outCodec)

	tl, err := webrtc.NewTrackLocalStaticRTP(outCodec.RTPCodecCapability, matched.TrackRemote.ID(), matched.TrackRemote.StreamID())
	if err != nil {
		return err
	}
	
	go func() {
		for {
			p, ok := <-buf
			if !ok {
				return
			}
			if err := tl.WriteRTP(p); err != nil {
				zap.L().Warn("failed to write rtp", zap.Error(err))
				return
			}
		}
	}()

	m := &webrtc.MediaEngine{}

	if err := m.RegisterCodec(*outCodec, matched.TrackRemote.Kind()); err != nil {
		return err
	}

	i := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		return err
	}

	peerConnection, err := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i)).NewPeerConnection(s.config)
	if err != nil {
		return err
	}

	rtpSender, err := peerConnection.AddTrack(tl)
	if err != nil {
		return err
	}

	go func() {
		buf := make([]byte, 1500)
		for {
			if _, _, err := rtpSender.Read(buf); err != nil {
				return
			}
		}
	}()

	signaller := signal.Negotiate(peerConnection)

	go func() {
		for {
			signal, err := signaller.ReadSignal()
			if err != nil {
				zap.L().Error("failed to read signal", zap.Error(err))
				return
			}

			log.Printf("received %v", signal)

			if err := conn.Send(signal); err != nil {
				zap.L().Error("failed to send signal", zap.Error(err))
				return
			}
		}
	}()

	for {
		signal, err := conn.Recv()
		if err != nil {
			return err
		}

		log.Printf("received %v", signal)

		switch signal := signal.Operation.(type) {
		case *api.SubscribeRequest_Signal:
			if err := signaller.WriteSignal(signal.Signal); err != nil {
				return err
			}
		case *api.SubscribeRequest_Request:
			return errors.New("unexpected request")
		}
	}
}
