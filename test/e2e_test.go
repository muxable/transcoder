package test

import (
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

		tc, err := transcode.NewTranscoder(webrtc.RTPCodecParameters{
			RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeOpus,
			ClockRate: 48000,
			},
			PayloadType: 96,
		}, transcode.ToMimeType(mime))
		if err != nil {
			t.Errorf("failed to create transcoder: %v", err)
			continue
		}

		reader, err := reader("filesrc location=input.ogg ! oggdemux ! rtpopuspay pt=96 ! appsink name=sink")
		if err != nil {
			t.Errorf("failed to create bin: %v", err)
		}
		writer, err := writer("appsrc format=time name=source ! application/x-rtp,encoding-name=(string)OPUS,clock-rate=(int)48000,payload=(int)96 ! rtpopusdepay ! queue ! decodebin ! autoaudiosink")
		if err != nil {
			t.Errorf("failed to create bin: %v", err)
		}
		
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
	}
	wg.Wait()
}
