package main

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/muxable/transcoder/internal/gst"
	"github.com/muxable/transcoder/internal/server"
	"github.com/pion/webrtc/v3"
)

func main() {
	loop := gst.MainLoopNew()

	pipeline1, err := gst.ParseLaunch("videotestsrc ! autovideosink")
	if err != nil {
		log.Fatalf("failed to create pipeline1: %v", err)
	}
	pipeline1.SetState(gst.StatePlaying)

	pipeline2, err := gst.ParseLaunch("udpsrc address=0.0.0.0 caps=application/x-rtp,encoding-name=VP8 ! rtpvp8depay ! queue ! decodebin ! queue ! autovideosink")
	if err != nil {
		log.Fatalf("failed to create pipeline2: %v", err)
	}
	pipeline2.SetState(gst.StatePlaying)

	loop.Wait()

	pipeline1.SetState(gst.StateNull)
	pipeline2.SetState(gst.StateNull)

	pipeline1.Close()
	pipeline2.Close()
	loop.Close()

	select{}
	for mime := range server.SupportedCodecs {
		if mime != webrtc.MimeTypeVP8 {
			continue
		}
		if !strings.HasPrefix(mime, "video") {
			continue
		}

		log.Printf("playing %s", mime)

		// outputCodec := server.DefaultOutputCodecs[mime]

		// tc, err := transcode.NewTranscoder(
		// 	server.DefaultOutputCodecs[webrtc.MimeTypeVP8],
		// 	transcode.ToOutputCodec(outputCodec))
		// if err != nil {
		// 	fmt.Printf("failed to create transcoder: %v", err)
		// 	continue
		// }

		r, err := gst.ParseLaunch("videotestsrc ! queue ! vp8enc deadline=1 ! queue ! rtpvp8pay ! udpsink host=127.0.0.1")
		if err != nil {
			fmt.Printf("failed to create pipeline: %v", err)
			return
		}
		w, err := gst.ParseLaunch("udpsrc address=0.0.0.0 caps=application/x-rtp,encoding-name=VP8 ! rtpvp8depay ! queue ! decodebin ! queue ! autovideosink")
		if err != nil {
			fmt.Printf("failed to create pipeline: %v", err)
			return
		}
		r.SetState(gst.StatePlaying)
		w.SetState(gst.StatePlaying)
		// r, err := reader("fmt.Sprintf("videotestsrc ! vp8enc ! rtpvp8pay pt=%d mtu=1200 ! appsink name=sink", server.DefaultOutputCodecs[webrtc.MimeTypeVP8].PayloadType))
		// if err != nil {
		// 	fmt.Printf("failed to create bin: %v", err)
		// }
		// w, err := writer("appsrc format=time name=source ! video/x-raw,format=RGB,width=640,height=480,framerate=30/1 ! videoconvert ! vp8enc ! avmux_ivf ! filesink location=output.ivf") // fmt.Sprintf("appsrc format=time name=source ! application/x-rtp,%s ! %s ! queue ! decodebin ! autovideosink", codec.ToCaps(outputCodec), codec.Depayloader))
		// if err != nil {
		// 	fmt.Printf("failed to create bin: %v", err)
		// }

		// log.Printf("PLAYING")
		var wg sync.WaitGroup
		wg.Add(1)
		// go func() {
		// 	sink :=  r.GetByName("sink")
		// 	source := w.GetByName("source")
		// 	defer source.Close()
		// 	for {
		// 		s, err := sink.ReadSample()
		// 		if err != nil {
		// 			fmt.Printf("failed to pull sample: %v", err)
		// 			break
		// 		}
		// 		if err := source.WriteSample(s); err != nil {
		// 			fmt.Printf("failed to push buffer: %v", err)
		// 			break
		// 		}
		// 	}
		// 	// tc.Close()
		// 	// wg.Done()
		// }()
		// // go func() {
		// // 	rtpio.CopyRTP(w, tc)
		// // 	w.Close()
		// // 	wg.Done()
		// // }()
		wg.Wait()
	}
}

// func TestTranscodingAudio(t *testing.T) {
// 	for mime, codec := range server.SupportedCodecs {
// 		if !strings.HasPrefix(mime, "audio") {
// 			continue
// 		}

// 		log.Printf("playing %s", mime)

// 		outputCodec := server.DefaultOutputCodecs[mime]

// 		tc, err := transcode.NewTranscoder(
// 			server.DefaultOutputCodecs[webrtc.MimeTypeOpus],
// 			transcode.ToOutputCodec(outputCodec))
// 		if err != nil {
// 			fmt.Printf("failed to create transcoder: %v", err)
// 			continue
// 		}

// 		reader, err := reader(fmt.Sprintf("filesrc location=input.ogg ! oggdemux ! rtpopuspay pt=%d mtu=1200 ! appsink name=sink", server.DefaultOutputCodecs[webrtc.MimeTypeOpus].PayloadType))
// 		if err != nil {
// 			fmt.Printf("failed to create bin: %v", err)
// 		}
// 		writer, err := writer(fmt.Sprintf("appsrc format=time is-live=true name=source ! application/x-rtp,%s ! %s ! queue ! decodebin ! audioconvert ! autoaudiosink", codec.ToCaps(outputCodec), codec.Depayloader))
// 		if err != nil {
// 			fmt.Printf("failed to create bin: %v", err)
// 		}

// 		var wg sync.WaitGroup
// 		wg.Add(2)
// 		go func() {
// 			rtpio.CopyRTP(tc, reader)
// 			tc.Close()
// 			wg.Done()
// 		}()
// 		go func() {
// 			rtpio.CopyRTP(writer, tc)
// 			writer.Close()
// 			wg.Done()
// 		}()
// 		wg.Wait()
// 	}
// }
