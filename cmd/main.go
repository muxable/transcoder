package main

import (
	"fmt"
	"net"
	"os"

	transcoder "github.com/muxable/transcoder/api"
	"github.com/muxable/transcoder/internal"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	undo := zap.ReplaceGlobals(logger)
	defer undo()

	port := os.Getenv("PORT")
	if port == "" {
			port = "50051"
	}

	addr := fmt.Sprintf(":%s", port)
	
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	s := grpc.NewServer()

	transcoder.RegisterTranscoderServer(s, &internal.TranscoderServer{
		Configuration: webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{URLs: []string{"stun:stun.l.google.com:19302"}},
			},
		},
		OnPeerConnection: func(pc *webrtc.PeerConnection) {
			if err := internal.TranscodePeerConnection(pc); err != nil {
				zap.L().Error("failed to transcode", zap.Error(err))
			}
		},
	})
	grpc_health_v1.RegisterHealthServer(s, health.NewServer())

	zap.L().Info("starting transcoder server", zap.String("addr", addr))

	if err := s.Serve(lis); err != nil {
		panic(err)
	}
}
