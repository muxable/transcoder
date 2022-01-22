package test

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"

	"github.com/muxable/transcoder/internal/gst"
	"github.com/muxable/transcoder/internal/server"
	"github.com/muxable/transcoder/pkg/transcode"
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
)

func writer(s string) (*gst.Element, error) {
	pipeline, err := gst.PipelineNew()
	if err != nil {
		return nil, err
	}
	bin, err := gst.ParseBinFromDescription(s)
	if err != nil {
		return nil, err
	}

	pipeline.SetState(gst.StatePlaying)
	pipeline.Add(&bin.Element)
	bin.SetState(gst.StatePlaying)

	return bin.GetByName("source"), nil
}

func reader(s string) (*gst.Element, error) {
	pipeline, err := gst.PipelineNew()
	if err != nil {
		return nil, err
	}
	bin, err := gst.ParseBinFromDescription(s)
	if err != nil {
		return nil, err
	}

	pipeline.SetState(gst.StatePlaying)
	pipeline.Add(&bin.Element)
	bin.SetState(gst.StatePlaying)

	return bin.GetByName("sink"), nil
}

func TestTranscodingVideo(t *testing.T) {
	for mime, codec := range server.SupportedCodecs {
		if mime != webrtc.MimeTypeVP8 {
			continue
		}
		if !strings.HasPrefix(mime, "video") {
			continue
		}

		log.Printf("playing %s", mime)

		outputCodec := server.DefaultOutputCodecs[mime]

		tc, err := transcode.NewTranscoder(
			server.DefaultOutputCodecs[webrtc.MimeTypeVP8],
			transcode.ToOutputCodec(outputCodec))
		if err != nil {
			t.Errorf("failed to create transcoder: %v", err)
			continue
		}

		reader, err := reader(fmt.Sprintf("filesrc location=input.ivf ! decodebin ! vp8enc ! rtpvp8pay pt=%d mtu=1200 ! appsink name=sink", server.DefaultOutputCodecs[webrtc.MimeTypeVP8].PayloadType))
		if err != nil {
			t.Errorf("failed to create bin: %v", err)
		}
		writer, err := writer(fmt.Sprintf("appsrc format=time is-live=true name=source ! application/x-rtp,%s ! %s ! queue ! decodebin ! autovideosink", codec.ToCaps(outputCodec), codec.Depayloader))
		if err != nil {
			t.Errorf("failed to create bin: %v", err)
		}

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			rtpio.CopyRTP(tc, reader)
			tc.Close()
			wg.Done()
		}()
		go func() {
			rtpio.CopyRTP(writer, tc)
			writer.Close()
			wg.Done()
		}()
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
// 		writer, err := writer(fmt.Sprintf("appsrc format=time is-live=true name=source ! application/x-rtp,%s ! %s ! queue ! decodebin ! audioconvert ! pulsesink provide-clock=false", codec.ToCaps(outputCodec), codec.Depayloader))
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
