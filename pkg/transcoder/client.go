package transcoder

import (
	"context"
	"fmt"
	"sync"

	"github.com/muxable/signal/pkg/signal"
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

	signaller := signal.Negotiate(peerConnection)

	client := api.NewTranscoderClient(conn)

	signalClient, err := client.Signal(ctx)
	if err != nil {
		return nil, err
	}

	c := &Client{
		ctx:            ctx,
		peerConnection: peerConnection,
		grpcClient:     client,
		promises:       make(map[string]chan *webrtc.TrackRemote),
	}

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
		for {
			signal, err := signaller.ReadSignal()
			if err != nil {
				zap.L().Error("failed to read signal", zap.Error(err))
				return
			}
			if err := signalClient.Send(signal); err != nil {
				zap.L().Error("failed to send signal", zap.Error(err))
				return
			}
		}
	}()

	go func() {
		defer peerConnection.Close()
		for {
			in, err := signalClient.Recv()
			if err != nil {
				zap.L().Error("failed to receive", zap.Error(err))
				return
			}

			if err := signaller.WriteSignal(in); err != nil {
				zap.L().Error("failed to write signal", zap.Error(err))
				return
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
