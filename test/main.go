package main

import (
	"log"
	"strings"

	"github.com/muxable/transcoder/pkg"
	gst_sink "github.com/pion/ion-sdk-go/pkg/gstreamer-sink"
	gst_src "github.com/pion/ion-sdk-go/pkg/gstreamer-src"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	zap.ReplaceGlobals(logger)

	// create tracks
	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video", "pion")
	if err != nil {
		panic(err)
	}

	audioTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "pion")
	if err != nil {
		panic(err)
	}

	conn, err := grpc.Dial("127.0.0.1:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	client, err := pkg.NewTranscoderAPIClient(conn)
	if err != nil {
		panic(err)
	}

	gst_src.CreatePipeline("opus", []*webrtc.TrackLocalStaticSample{audioTrack}, "uridecodebin uri=file:///home/kevin/transcoder/test/input.ogg").Start()
	gst_src.CreatePipeline("h264", []*webrtc.TrackLocalStaticSample{videoTrack}, "videotestsrc").Start()

	go func() {
		transcodedVideoTrack, err := client.Transcode(videoTrack)
		if err != nil {
			panic(err)
		}

		codecName := strings.Split(transcodedVideoTrack.Codec().RTPCodecCapability.MimeType, "/")[1]
		p := gst_sink.CreatePipeline(strings.ToLower(codecName), "autovideosink")
		p.Start()

		buf := make([]byte, 1400)
		for {
			i, _, err := transcodedVideoTrack.Read(buf)
			log.Printf("read %d bytes", i)
			if err != nil {
				panic(err)
			}
			p.Push(buf[:i])
		}
	}()

	go func() {
		transcodedAudioTrack, err := client.Transcode(audioTrack)
		if err != nil {
			panic(err)
		}

		codecName := strings.Split(transcodedAudioTrack.Codec().RTPCodecCapability.MimeType, "/")[1]
		p := gst_sink.CreatePipeline(strings.ToLower(codecName), "autoaudiosink")
		p.Start()

		buf := make([]byte, 1400)
		log.Printf("listening")
		for {
			i, _, err := transcodedAudioTrack.Read(buf)
			log.Printf("read %d bytes", i)
			if err != nil {
				panic(err)
			}
			log.Printf("%v audio bytes", i)
			p.Push(buf[:i])
		}
	}()

	select {}
}
