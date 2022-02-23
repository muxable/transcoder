package transcoder

import (
	"context"
	"log"

	"github.com/muxable/signal/pkg/signal"
	"github.com/muxable/transcoder/api"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Client struct {
	ctx  context.Context
	conn *grpc.ClientConn
}

func NewClient(ctx context.Context, conn *grpc.ClientConn) (*Client, error) {
	return &Client{
		ctx:  ctx,
		conn: conn,
	}, nil
}

type TranscodeOption func(*api.TranscodeRequest)

func (c *Client) Transcode(tl *webrtc.TrackLocalStaticRTP, options ...TranscodeOption) (*webrtc.TrackRemote, error) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	m := &webrtc.MediaEngine{}

	if err := m.RegisterCodec(webrtc.RTPCodecParameters{RTPCodecCapability: tl.Codec(), PayloadType: webrtc.PayloadType(96)}, tl.Kind()); err != nil {
		return nil, err
	}

	i := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		return nil, err
	}

	send, err := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i)).NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	sendSignaller := signal.Negotiate(send)

	sendClient := api.NewTranscoderClient(c.conn)

	sendSignal, err := sendClient.Publish(c.ctx)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			signal, err := sendSignaller.ReadSignal()
			if err != nil {
				zap.L().Error("failed to read signal", zap.Error(err))
				return
			}

			log.Printf("signal %v", signal)

			if err := sendSignal.Send(signal); err != nil {
				zap.L().Error("failed to send signal", zap.Error(err))
				return
			}
		}
	}()

	go func() {
		defer send.Close()
		for {
			in, err := sendSignal.Recv()
			if err != nil {
				zap.L().Error("failed to receive", zap.Error(err))
				return
			}
			log.Printf("signal %v", in)

			if err := sendSignaller.WriteSignal(in); err != nil {
				zap.L().Error("failed to write signal", zap.Error(err))
				return
			}
		}
	}()

	rtpSender, err := send.AddTrack(tl)
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

	recv, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	recvSignaller := signal.Negotiate(recv)

	recvClient := api.NewTranscoderClient(c.conn)

	recvSignal, err := recvClient.Subscribe(c.ctx)
	if err != nil {
		return nil, err
	}

	promise := make(chan *webrtc.TrackRemote)

	recv.OnTrack(func(tr *webrtc.TrackRemote, r *webrtc.RTPReceiver) {
		go func() {
			buf := make([]byte, 1500)
			for {
				if _, _, err := r.Read(buf); err != nil {
					return
				}
			}
		}()

		log.Printf("got track %v", tr)

		promise <- tr
	})

	go func() {
		for {
			signal, err := recvSignaller.ReadSignal()
			if err != nil {
				zap.L().Error("failed to read signal", zap.Error(err))
				return
			}

			if err := recvSignal.Send(&api.SubscribeRequest{Operation: &api.SubscribeRequest_Signal{Signal: signal}}); err != nil {
				zap.L().Error("failed to send signal", zap.Error(err))
				return
			}
		}
	}()

	go func() {
		defer recv.Close()
		for {
			in, err := recvSignal.Recv()
			if err != nil {
				zap.L().Error("failed to receive", zap.Error(err))
				return
			}

			if err := recvSignaller.WriteSignal(in); err != nil {
				zap.L().Error("failed to write signal", zap.Error(err))
				return
			}
		}
	}()

	request := &api.TranscodeRequest{
		StreamId:    tl.StreamID(),
		TrackId:     tl.ID(),
		RtpStreamId: tl.RID(),
	}

	for _, option := range options {
		option(request)
	}

	if err := recvSignal.Send(&api.SubscribeRequest{Operation: &api.SubscribeRequest_Request{Request: request}}); err != nil {
		return nil, err
	}

	return <-promise, nil
}

func ToMimeType(mimeType string) TranscodeOption {
	return func(request *api.TranscodeRequest) {
		request.MimeType = mimeType
	}
}
