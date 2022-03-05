package main

import (
	"net"
	"os"

	"github.com/blendle/zapdriver"
	"github.com/muxable/transcoder/pkg/av"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

/*
import (
	"flag"
	"net"
	"os"

	"github.com/blendle/zapdriver"
	"github.com/muxable/transcoder/api"
	"github.com/muxable/transcoder/pkg/transcoder"
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

	api.RegisterTranscoderServer(s, transcoder.NewTranscoderServer(webrtc.Configuration{
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
*/

func logger() (*zap.Logger, error) {
	if os.Getenv("APP_ENV") == "production" {
		return zapdriver.NewProduction()
	} else {
		return zap.NewDevelopment()
	}
}

func main() {
	logger, err := logger()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	undo := zap.ReplaceGlobals(logger)
	defer undo()

	tc, err := av.NewTranscoder(webrtc.RTPCodecParameters{
		PayloadType: 96,
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: "video/H265",
			ClockRate: 90000,
		},
	}, 
	webrtc.RTPCodecCapability{
		MimeType: "video/H264",
		ClockRate: 90000,
	})

	if err != nil {
		panic(err)
	}

	zap.L().Info("starting transcoder server")

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 5000})
	if err != nil {
		panic(err)
	}

	dial, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5001})
	if err != nil {
		panic(err)
	}

	go func() {
		buf := make([]byte, 1500)
		defer tc.Close()
		for {
			n, err := conn.Read(buf)
			if err != nil {
				return
			}
			p := &rtp.Packet{}
			if err := p.Unmarshal(buf[:n]); err != nil {
				panic(err)
			}
			if err := tc.WriteRTP(p); err != nil {
				panic(err)
			}
		}
	}()

	for {
		p, err := tc.ReadRTP()
		if err != nil {
			return
		}
		buf, err := p.Marshal()
		if err != nil {
			panic(err)
		}
		if _, err := dial.Write(buf); err != nil {
			panic(err)
		}
	}
}