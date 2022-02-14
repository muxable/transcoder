package transcoder

import (
	"io"
	"testing"

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

	p, err := s.NewReadWritePipeline(&webrtc.RTPCodecParameters{}, "identity")
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	buf1 := []byte("test")
	buf2 := []byte("moo")
	buf3 := []byte("cows")

	// write some data and read some data.
	if _, err := p.Write(buf1); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Write(buf2); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Write(buf3); err != nil {
		t.Fatal(err)
	}

	// read the data back
	got1 := make([]byte, len(buf1))
	if _, err := p.Read(got1); err != nil {
		t.Fatal(err)
	}
	if string(got1) != string(buf1) {
		t.Fatalf("got %s, want %s", got1, buf1)
	}

	got2 := make([]byte, len(buf2))
	if _, err := p.Read(got2); err != nil {
		t.Fatal(err)
	}
	if string(got2) != string(buf2) {
		t.Fatalf("got %s, want %s", got2, buf2)
	}

	got3 := make([]byte, len(buf3))
	if _, err := p.Read(got3); err != nil {
		t.Fatal(err)
	}
}

func TestPipeline_ReadOnly(t *testing.T) {
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

	p, err := s.NewReadOnlyPipeline("videotestsrc num-buffers=100 ! x264enc ! rtph264pay")
	if err != nil {
		t.Fatal(err)
	}

	// read some data.
	for {
		pkt, err := p.ReadRTP()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		if pkt.PayloadType != 96 {
			t.Fatalf("got %d, want 96", pkt.PayloadType)
		}
		if len(pkt.Payload) == 0 {
			t.Fatal("got empty payload")
		}
	}
}

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
	if codec.SDPFmtpLine == "" {
		t.Fatal("got empty SDPFmtpLine")
	}
	if codec.ClockRate != 90000 {
		t.Fatalf("got %d, want 90000", codec.ClockRate)
	}
}
