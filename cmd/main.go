package main

import (
	"flag"
	"net"
	"os"

	"github.com/blendle/zapdriver"
	"github.com/muxable/transcoder/api"
	"github.com/muxable/transcoder/pkg/server"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func logger() (*zap.Logger, error) {
	if os.Getenv("APP_ENV") == "production" {
		return zapdriver.NewProduction()
	} else {
		return zap.NewDevelopment()
	}
}

func main() {
	addr := flag.String("addr", ":50051", "The address to listen on")
	flag.Parse()

	logger, err := logger()
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

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		panic(err)
	}

	s := grpc.NewServer()

	api.RegisterTranscoderServer(s, server.NewTranscoderServer(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}))
	grpc_health_v1.RegisterHealthServer(s, health.NewServer())

	zap.L().Info("starting transcoder server", zap.String("addr", *addr))

	if err := s.Serve(lis); err != nil {
		panic(err)
	}
}
