package av

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/muxable/rtptools/pkg/h265"
	h265reader "github.com/muxable/rtptools/pkg/h265/reader"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/h264writer"
)

func TestTranscode_Video(t *testing.T) {
	f, err := os.Open("../../test/video.h265")
	if err != nil {
		t.Fatalf("failed to open input file: %v", err)
	}

	r, err := h265reader.NewReader(f)
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}

	w, err := h264writer.New("../../test/output.h264")
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}
	defer w.Close()

	h265packetizer := rtp.NewPacketizer(
		1200,
		96,
		0,
		&h265.H265Payloader{},
		rtp.NewRandomSequencer(),
		90000,
	)

	tc, err := NewTranscoder(webrtc.RTPCodecParameters{
		PayloadType: 96,
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeH265,
			ClockRate: 90000,
			SDPFmtpLine: "sprop-vps=QAEMA///AWAAAAMAAAMAAAMAAAMAHgAAtSOBIA==; sprop-sps=QgEDAWAAAAMAAAMAAAMAAAMAHgAAoAPAgBDn+JtSPJI2R3JSU29OFIbmgIAAAfQAAHUSBA==; sprop-pps=RAHBVPBwxCQA",
		},
	}, webrtc.RTPCodecCapability{
		MimeType:  webrtc.MimeTypeH264,
		ClockRate: 90000,
	})
	if err != nil {
		t.Fatalf("failed to create transcoder: %v", err)
	}

	time.Sleep(1 * time.Second)

	go func() {
		defer tc.Close()
		for {
			time.Sleep(33 * time.Millisecond)
			nal, err := r.NextNAL()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("failed to read next NAL: %v", err)
			}

			samples := uint32(1750)
			for _, p := range h265packetizer.Packetize(nal.Data, samples) {
				if err := tc.WriteRTP(p); err != nil {
					t.Errorf("failed to write RTP packet: %v", err)
				}
			}
		}
	}()

	// read tc until eof
	for {
		p, err := tc.ReadRTP()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("failed to read next packet: %v", err)
		}
		if err := w.WriteRTP(p); err != nil {
			t.Fatalf("failed to write RTP packet: %v", err)
		}
	}
	select{}
}

// func TestTranscode_Audio(t *testing.T) {
// 	f, err := os.Open("../../test/input.ogg")
// 	if err != nil {
// 		t.Fatalf("failed to open input file: %v", err)
// 	}

// 	r, _, err := oggreader.NewWith(f)
// 	if err != nil {
// 		t.Fatalf("failed to create reader: %v", err)
// 	}

// 	w, err := oggwriter.New("../../test/output.ogg", 48000, 2)
// 	if err != nil {
// 		t.Fatalf("failed to create writer: %v", err)
// 	}
// 	defer w.Close()

// 	packetizer := rtp.NewPacketizer(
// 		1200,
// 		96,
// 		0,
// 		&codecs.OpusPayloader{},
// 		rtp.NewRandomSequencer(),
// 		90000,
// 	)

// 	tc := NewTranscoder(webrtc.RTPCodecCapability{
// 		MimeType:  webrtc.MimeTypeOpus,
// 		ClockRate: 48000,
// 		Channels:  2,
// 	}, webrtc.RTPCodecCapability{
// 		MimeType:  webrtc.MimeTypeOpus,
// 		ClockRate: 48000,
// 		Channels:  2,
// 	})

// 	go func() {
// 		defer tc.Close()
// 		var lastGranule uint64
// 		for {
// 			pageData, pageHeader, err := r.ParseNextPage()
// 			if err == io.EOF {
// 				break
// 			}
// 			if err != nil {
// 				t.Errorf("failed to read next page: %v", err)
// 			}

// 			// The amount of samples is the difference between the last and current timestamp
// 			sampleCount := pageHeader.GranulePosition - lastGranule
// 			lastGranule = pageHeader.GranulePosition
// 			for _, p := range packetizer.Packetize(pageData, uint32(sampleCount)) {
// 				log.Printf("%x", p.Payload)
// 				if err := tc.WriteRTP(p); err != nil {
// 					t.Errorf("failed to write RTP packet: %v", err)
// 				}
// 			}
// 		}
// 	}()

// 	// read tc until eof
// 	for {
// 		p, err := tc.ReadRTP()
// 		if err == io.EOF {
// 			break
// 		}
// 		if err != nil {
// 			t.Fatalf("failed to read next packet: %v", err)
// 		}

// 		if err := w.WriteRTP(p); err != nil {
// 			t.Fatalf("failed to write RTP packet: %v", err)
// 		}
// 	}
// }
