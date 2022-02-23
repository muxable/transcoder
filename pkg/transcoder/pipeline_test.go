package transcoder

import (
	"testing"

	"github.com/pion/rtp"
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
	"go.uber.org/goleak"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestPipeline_Empty(t *testing.T) {
	logger := zaptest.NewLogger(t)
	defer logger.Sync()
	undo := zap.ReplaceGlobals(logger)
	defer undo()

	defer goleak.VerifyNone(t)

	s, err := NewTranscoder()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	p, err := s.NewReadWritePipeline(&webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeH264,
			ClockRate: 90000,
		},
		PayloadType: 96,
	}, "identity")
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	buf1 := &rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 1}}
	buf2 := &rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 2}}
	buf3 := &rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 3}}

	// write some data and read some data.
	if err := p.WriteRTP(buf1); err != nil {
		t.Fatal(err)
	}
	if err := p.WriteRTP(buf2); err != nil {
		t.Fatal(err)
	}
	if err := p.WriteRTP(buf3); err != nil {
		t.Fatal(err)
	}

	// read the data back
	got1, err := p.ReadRTP()
	if err != nil {
		t.Fatal(err)
	}
	if got1.SequenceNumber != 1 {
		t.Fatalf("got %d, want 1", got1.SequenceNumber)
	}

	got2, err := p.ReadRTP()
	if err != nil {
		t.Fatal(err)
	}
	if got2.SequenceNumber != 2 {
		t.Fatalf("got %d, want 2", got2.SequenceNumber)
	}

	got3, err := p.ReadRTP()
	if err != nil {
		t.Fatal(err)
	}
	if got3.SequenceNumber != 3 {
		t.Fatalf("got %d, want 3", got3.SequenceNumber)
	}
}

// func TestPipeline_ReadOnly(t *testing.T) {
// 	logger := zaptest.NewLogger(t)
// 	defer logger.Sync()
// 	undo := zap.ReplaceGlobals(logger)
// 	defer undo()

// 	defer goleak.VerifyNone(t)

// 	s, err := NewTranscoder()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer s.Close()

// 	p, err := s.NewReadOnlyPipeline("videotestsrc num-buffers=100 ! x264enc ! rtph264pay")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// read some data.
// 	for {
// 		pkt, err := p.ReadRTP()
// 		if err != nil {
// 			if err == io.EOF {
// 				break
// 			}
// 			t.Fatal(err)
// 		}
// 		if pkt.PayloadType != 96 {
// 			t.Fatalf("got %d, want 96", pkt.PayloadType)
// 		}
// 		if len(pkt.Payload) == 0 {
// 			t.Fatal("got empty payload")
// 		}
// 	}
// }

func TestPipeline_Transcode(t *testing.T) {
	logger := zaptest.NewLogger(t)
	defer logger.Sync()
	undo := zap.ReplaceGlobals(logger)
	defer undo()

	defer goleak.VerifyNone(t)

	s, err := NewTranscoder()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	q, err := s.NewReadOnlyPipeline("videotestsrc num-buffers=100 ! x264enc ! rtph264pay")
	if err != nil {
		t.Fatal(err)
	}

	qcodec, err := q.Codec()
	if err != nil {
		t.Fatal(err)
	}

	p, err := s.NewReadWritePipeline(qcodec, "rtph264depay ! decodebin ! videoconvert ! video/x-raw,format=I420 ! x265enc ! rtph265pay pt=100")
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	go rtpio.CopyRTP(p, q)

	ssrc, err := p.SSRC()
	if err != nil {
		t.Fatal(err)
	}
	if ssrc == 0 {
		t.Fatal("got 0 ssrc")
	}
	codec, err := p.Codec()
	if err != nil {
		t.Fatal(err)
	}
	if codec.PayloadType != 100 {
		t.Fatalf("got %d, want 100", codec.PayloadType)
	}
	if codec.MimeType != webrtc.MimeTypeH265 {
		t.Fatalf("got %s, want %s", codec.MimeType, webrtc.MimeTypeH265)
	}
	if codec.ClockRate != 90000 {
		t.Fatalf("got %d, want 90000", codec.ClockRate)
	}
}
