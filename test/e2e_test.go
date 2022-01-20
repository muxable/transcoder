package test

import (
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/muxable/rtpio/pkg/rtpio"
	"github.com/muxable/transcoder/internal/gst"
	"github.com/muxable/transcoder/internal/server"
	"github.com/muxable/transcoder/pkg/transcode"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/oggreader"
)

const (
	oggPageDuration = time.Millisecond * 20
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

// func TestTranscodingVP8ToAny(t *testing.T) {
// 	var wg sync.WaitGroup
// 	for mime, codec := range server.SupportedCodecs {
// 		if !strings.HasPrefix(mime, "video") {
// 			continue
// 		}

// 		tc, err := transcode.NewTranscoder(webrtc.RTPCodecCapability{
// 			MimeType:  webrtc.MimeTypeVP8,
// 			ClockRate: 90000,
// 		}, transcode.ToMimeType(mime))
// 		if err != nil {
// 			t.Errorf("failed to create transcoder: %v", err)
// 			continue
// 		}

// 		writer, err := writer(fmt.Sprintf("appsrc format=time name=source ! application/x-rtp,encoding-name=(string)%s ! %s ! filesink location=output.%s", codec.EncodingName, codec.Depayloader, codec.EncodingName))
// 		if err != nil {
// 			t.Errorf("failed to create bin: %v", err)
// 		}

// 		file, ivfErr := os.Open("input.ivf")
// 		if ivfErr != nil {
// 			t.Errorf("failed to open input file: %v", ivfErr)
// 			continue
// 		}

// 		ivf, header, ivfErr := ivfreader.NewWith(file)
// 		if ivfErr != nil {
// 			t.Errorf("failed to read IVF file: %v", ivfErr)
// 			continue
// 		}

// 		packetizer := rtp.NewPacketizer(1200, 96, 0, &codecs.VP8Payloader{}, rtp.NewRandomSequencer(), 90000)
// 		duration := time.Millisecond * time.Duration((float32(header.TimebaseNumerator)/float32(header.TimebaseDenominator))*1000)

// 		wg.Add(2)

// 		go func() {
// 			defer wg.Done()
// 			defer tc.Close()
// 			ticker := time.NewTicker(duration)
// 			for range ticker.C {
// 				frame, _, err := ivf.ParseNextFrame()
// 				if err == io.EOF {
// 					return
// 				}

// 				if err != nil {
// 					t.Errorf("failed to read IVF frame: %v", err)
// 				}

// 				for _, p := range packetizer.Packetize(frame, uint32(duration.Seconds()*90000)) {
// 					if err := tc.WriteRTP(p); err != nil {
// 						t.Errorf("failed to write sample: %v", err)
// 					}
// 				}
// 			}
// 		}()

// 		src := transcode.NewSource(writer)
// 		go func() {
// 			rtpio.CopyRTP(src, tc)
// 			wg.Done()
// 		}()
// 	}
// 	wg.Wait()
// }

func TestTranscodingOggToAny(t *testing.T) {
	var wg sync.WaitGroup
	for mime, codec := range server.SupportedCodecs {
		if mime != webrtc.MimeTypeOpus {
			continue
		}
		if !strings.HasPrefix(mime, "audio") {
			continue
		}

		tc, err := transcode.NewTranscoder(webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeOpus,
			ClockRate: 48000,
		}, transcode.ToMimeType(mime))
		if err != nil {
			t.Errorf("failed to create transcoder: %v", err)
			continue
		}

		log.Printf("creating writer %s", codec)
		// writer, err := writer(fmt.Sprintf("appsrc format=time name=source ! application/x-rtp,encoding-name=(string)%s ! %s ! filesink location=output.%s", codec.EncodingName, codec.Depayloader, codec.EncodingName))
		writer, err := writer("appsrc format=time name=source ! fakesink dump=true")
		if err != nil {
			t.Errorf("failed to create bin: %v", err)
		}
		log.Printf("writing")

		file, oggErr := os.Open("input.ogg")
		if oggErr != nil {
			t.Errorf("failed to open input file: %v", oggErr)
			continue
		}

		// Open on oggfile in non-checksum mode.
		ogg, _, oggErr := oggreader.NewWith(file)
		if oggErr != nil {
			t.Errorf("failed to read OGG file: %v", oggErr)
			continue
		}

		// Keep track of last granule, the difference is the amount of samples in the buffer
		var lastGranule uint64

		packetizer := rtp.NewPacketizer(1200, 96, 0, &codecs.OpusPayloader{}, rtp.NewRandomSequencer(), 90000)

		wg.Add(2)

		go func() {
			defer wg.Done()
			defer tc.Close()
			ticker := time.NewTicker(oggPageDuration)
			for range ticker.C {
				pageData, pageHeader, oggErr := ogg.ParseNextPage()
				if oggErr == io.EOF {
					return
				}

				if oggErr != nil {
					t.Errorf("failed to read OGG page: %v", oggErr)
					continue
				}

				// The amount of samples is the difference between the last and current timestamp
				sampleCount := uint32(pageHeader.GranulePosition - lastGranule)
				lastGranule = pageHeader.GranulePosition

				for _, p := range packetizer.Packetize(pageData, sampleCount) {
					if err := tc.WriteRTP(p); err != nil {
						t.Errorf("failed to write sample: %v", err)
					}
				}
			}
		}()

		src := transcode.NewSource(writer)
		go func() {
			rtpio.CopyRTP(src, tc)
			wg.Done()
		}()
	}
	wg.Wait()
}
