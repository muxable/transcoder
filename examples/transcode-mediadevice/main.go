package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/muxable/transcoder/examples/internal/signal"
	"github.com/muxable/transcoder/pkg/transcoder"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// The transcoder API typically requires a bidi streaming gRPC connection which is difficult
// to replicate on the browser (at least, not without a decent amount of infrastructure). Since
// transcoding directly on from a browser isn't a common use case, we'll use a simple SDP exchange
// to broker the connection. In practice, your code will look much simpler if a track is originating
// from a server.

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	zap.ReplaceGlobals(logger)

	addr := flag.String("addr", "localhost:50051", "transcoder grpc addr")

	flag.Parse()

	// Create a new transcoding client.
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	tc, err := transcoder.NewClient(context.Background(), conn)
	if err != nil {
		panic(err)
	}

	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new PeerConnection to the browser.
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	defer peerConnection.Close()

	// Read incoming RTCP packets
	// Before these packets are returned they are processed by interceptors. For things
	// like NACK this needs to be called.
	processRTCP := func(rtpSender *webrtc.RTPSender) {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}
	for _, rtpSender := range peerConnection.GetSenders() {
		go processRTCP(rtpSender)
	}

	// Wait for the offer to be pasted
	offer := webrtc.SessionDescription{}
	signal.Decode(signal.MustReadStdin(), &offer)

	if err = peerConnection.SetRemoteDescription(offer); err != nil {
		panic(err)
	}

	transcodedLocal, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "transcoded-track", "transcoded-h264-stream")
	if err != nil {
		panic(err)
	}

	// Add the track to the PeerConnection
	rtpSender, err := peerConnection.AddTrack(transcodedLocal)
	if err != nil {
		panic(err)
	}
	
	go processRTCP(rtpSender)

	// Set a handler for when a new remote track starts
	peerConnection.OnTrack(func(trackRemote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		fmt.Printf("on track %s (%s)\n", trackRemote.ID(), trackRemote.Codec().MimeType)
		
		// Create a new TrackLocal
		trackLocal, err := webrtc.NewTrackLocalStaticRTP(trackRemote.Codec().RTPCodecCapability, trackRemote.ID(), trackRemote.StreamID())
		if err != nil {
			panic(err)
		}

		// Read RTP packets being sent to Pion
		go func() {
			for {
				packet, _, readErr := trackRemote.ReadRTP()
				if readErr != nil {
					panic(readErr)
				}

				if writeErr := trackLocal.WriteRTP(packet); writeErr != nil && !errors.Is(writeErr, io.ErrClosedPipe) {
					panic(writeErr)
				}
			}
		}()

		transcodedRemote, err := tc.Transcode(trackLocal)
		if err != nil {
			panic(err)
		}

		// Write RTP packets back to the client.
		go func() {
			for {
				packet, _, readErr := transcodedRemote.ReadRTP()
				if readErr != nil {
					panic(readErr)
				}

				if writeErr := transcodedLocal.WriteRTP(packet); writeErr != nil && !errors.Is(writeErr, io.ErrClosedPipe) {
					panic(writeErr)
				}
			}
		}()
	})

	// Set the handler for Peer connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", s.String())

		if s == webrtc.PeerConnectionStateFailed {
			// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
			// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
			// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
			fmt.Println("Peer Connection has gone to failed exiting")
			os.Exit(0)
		}
	})

	// Create an answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	// Sets the LocalDescription, and starts our UDP listeners
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		panic(err)
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	<-gatherComplete

	// Output the answer in base64 so we can paste it in browser
	fmt.Println(signal.Encode(*peerConnection.LocalDescription()))

	// Block forever
	select {}
}
