package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"time"

	"github.com/muxable/transcoder/pkg/transcoder"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"github.com/pion/webrtc/v3/pkg/media/ivfwriter"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func fileToVideoTrack(filename string, tl *webrtc.TrackLocalStaticSample) {
	file, ivfErr := os.Open(filename)
	if ivfErr != nil {
		panic(ivfErr)
	}

	ivf, header, ivfErr := ivfreader.NewWith(file)
	if ivfErr != nil {
		panic(ivfErr)
	}

	dt := time.Millisecond * time.Duration((float32(header.TimebaseNumerator)/float32(header.TimebaseDenominator))*1000)
	ticker := time.NewTicker(dt)
	for ; true; <-ticker.C {
		frame, _, ivfErr := ivf.ParseNextFrame()
		if ivfErr == io.EOF {
			os.Exit(0)
		}

		if ivfErr != nil {
			panic(ivfErr)
		}

		if ivfErr = tl.WriteSample(media.Sample{Data: frame, Duration: dt}); ivfErr != nil {
			panic(ivfErr)
		}
	}
}

func videoTrackToFile(tr *webrtc.TrackRemote, filename string) {
	ivfFile, err := ivfwriter.New(filename)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := ivfFile.Close(); err != nil {
			panic(err)
		}
	}()

	for {
		rtpPacket, _, err := tr.ReadRTP()
		if err != nil {
			panic(err)
		}
		if err := ivfFile.WriteRTP(rtpPacket); err != nil {
			panic(err)
		}
	}
}

func main() {
	addr := flag.String("addr", "localhost:50051", "the address to connect to")
	input := flag.String("i", "input.ivf", "the input file")
	output := flag.String("o", "output.ivf", "the output file")
	flag.Parse()

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	zap.ReplaceGlobals(logger)

	// create tracks
	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video", "pion")
	if err != nil {
		panic(err)
	}

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	client, err := transcoder.NewClient(context.Background(), conn)
	if err != nil {
		panic(err)
	}

	go fileToVideoTrack(*input, videoTrack)

	transcodedVideoTrack, err := client.Transcode(videoTrack, transcoder.ToMimeType(webrtc.MimeTypeVP8))
	if err != nil {
		panic(err)
	}

	go videoTrackToFile(transcodedVideoTrack, *output)

	select {}
}
