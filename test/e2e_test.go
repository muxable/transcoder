package test

import (
	"log"
	"strings"
	"sync"
	"testing"

	"github.com/muxable/transcoder/internal/gst"
	"github.com/muxable/transcoder/internal/server"
	"github.com/pion/webrtc/v3"
)

func TestTranscodingVideo(t *testing.T) {
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
		// 	t.Errorf("failed to create transcoder: %v", err)
		// 	continue
		// }

		r, err := gst.ParseLaunch("filesrc location=input.ivf ! avdemux_ivf ! rtpvp8pay ! appsink name=sink")
		if err != nil {
			t.Errorf("failed to create pipeline: %v", err)
			continue
		}
		w, err := gst.ParseLaunch("appsrc do-timestamp=true format=time name=source ! application/x-rtp,encoding-name=VP8 ! rtpvp8depay ! vp8dec ! fakesink dump=true")
		if err != nil {
			t.Errorf("failed to create pipeline: %v", err)
			continue
		}
		r.SetState(gst.StatePlaying)
		w.SetState(gst.StatePlaying)
		// r, err := reader("videotestsrc num-buffers=1000 ! video/x-raw,format=RGB,width=640,height=480,framerate=30/1 ! appsink name=sink") // fmt.Sprintf("videotestsrc ! vp8enc ! rtpvp8pay pt=%d mtu=1200 ! appsink name=sink", server.DefaultOutputCodecs[webrtc.MimeTypeVP8].PayloadType))
		// if err != nil {
		// 	t.Errorf("failed to create bin: %v", err)
		// }
		// w, err := writer("appsrc format=time name=source ! video/x-raw,format=RGB,width=640,height=480,framerate=30/1 ! videoconvert ! vp8enc ! avmux_ivf ! filesink location=output.ivf") // fmt.Sprintf("appsrc format=time name=source ! application/x-rtp,%s ! %s ! queue ! decodebin ! autovideosink", codec.ToCaps(outputCodec), codec.Depayloader))
		// if err != nil {
		// 	t.Errorf("failed to create bin: %v", err)
		// }

		// log.Printf("PLAYING")
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			for {
				s, err := r.GetByName("sink").PullSample()
				if err != nil {
					t.Errorf("failed to pull sample: %v", err)
					break
				}
				if err := w.GetByName("source").PushBuffer(s.Data); err != nil {
					t.Errorf("failed to push buffer: %v", err)
					break
				}
			}
			// tc.Close()
			wg.Done()
		}()
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
// 			t.Errorf("failed to create transcoder: %v", err)
// 			continue
// 		}

// 		reader, err := reader(fmt.Sprintf("filesrc location=input.ogg ! oggdemux ! rtpopuspay pt=%d mtu=1200 ! appsink name=sink", server.DefaultOutputCodecs[webrtc.MimeTypeOpus].PayloadType))
// 		if err != nil {
// 			t.Errorf("failed to create bin: %v", err)
// 		}
// 		writer, err := writer(fmt.Sprintf("appsrc format=time is-live=true name=source ! application/x-rtp,%s ! %s ! queue ! decodebin ! audioconvert ! autoaudiosink", codec.ToCaps(outputCodec), codec.Depayloader))
// 		if err != nil {
// 			t.Errorf("failed to create bin: %v", err)
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
