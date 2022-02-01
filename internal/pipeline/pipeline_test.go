package pipeline

import (
	"io"
	"runtime"
	"testing"
	"time"

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

	s, err := NewSynchronizer()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	p, err := s.NewReadWritePipeline("identity")
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

	s, err := NewSynchronizer()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	p, err := s.NewReadOnlyPipeline("videotestsrc num-buffers=100")
	if err != nil {
		t.Fatal(err)
	}

	// read some data.
	for i := 0; i < 100; i++ {
		sample, err := p.ReadBuffer()
		if err != nil {
			t.Fatal(err)
		}
		if len(sample.Data)%3 != 0 {
			t.Fatalf("got %d, want multiple of 3", len(sample.Data))
		}
		if sample.Offset != i {
			t.Fatalf("got %d, want %d", sample.Offset, i)
		}
	}
	// verify the next read returns io.EOF
	sample, err := p.ReadBuffer()
	if err != io.EOF {
		t.Fatalf("got %v, want io.EOF", err)
	}
	if sample != nil {
		t.Fatalf("got %v, want nil", sample)
	}
}

func TestPipeline_WriteOnly(t *testing.T) {
	logger := zaptest.NewLogger(t)
	defer logger.Sync()
	undo := zap.ReplaceGlobals(logger)
	defer undo()

	defer goleak.VerifyNone(t)

	s, err := NewSynchronizer()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	p, err := s.NewWriteOnlyPipeline("testsink name=sink")
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// write some data.
	buf := make([]byte, 1400)
	for i := 0; i < 100; i++ {
		n, err := p.Write(buf)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(buf) {
			t.Fatalf("got %d, want %d", n, len(buf))
		}
	}
	// check that the sink received the data
	time.Sleep(100 * time.Millisecond)
	sink, err := p.GetElement("sink")
	if err != nil {
		t.Fatal(err)
	}
	n := sink.GetInt("buffer-count")
	// unfortunately due to the way the sink is implemented, we can't
	// guarantee that the sink has received all the data.
	if n == 0 {
		t.Fatalf("got %d, want %d", n, 100)
	}
}

// this test requires an appsrc to work correctly.

// func TestPipeline_EOS(t *testing.T) {
// 	logger := zaptest.NewLogger(t)
// 	defer logger.Sync()
// 	undo := zap.ReplaceGlobals(logger)
// 	defer undo()

// 	defer goleak.VerifyNone(t)

// 	s, err := NewSynchronizer()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer s.Close()

// 	p, err := s.NewReadWritePipeline("identity sleep-time=10000") // 10ms
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer p.Close()

// 	if _, err := p.Write([]byte("test")); err != nil {
// 		t.Fatal(err)
// 	}
// 	buf := make([]byte, 4)
// 	if _, err := p.Read(buf); err != nil {
// 		t.Fatal(err)
// 	}

// 	if err := p.Close(); err != nil {
// 		t.Fatal(err)
// 	}

// 	a := time.Now()
// 	if _, err := p.Read(buf); err != io.EOF {
// 		t.Errorf("got %v, want io.EOF", err)
// 	}
// 	b := time.Now()
// 	if b.Sub(a) < time.Millisecond*10 {
// 		t.Error("expected pipeline to wait for EOS")
// 	}
// }

func TestPipeline_RTPPiping(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	logger := zaptest.NewLogger(t)
	defer logger.Sync()
	undo := zap.ReplaceGlobals(logger)
	defer undo()

	defer goleak.VerifyNone(t)

	a, err := NewSynchronizer()
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	p1, err := a.NewReadOnlyPipeline("videotestsrc is-live=true num-buffers=20 ! x264enc speed-preset=veryfast tune=zerolatency ! rtph264pay mtu=1200")
	if err != nil {
		t.Fatal(err)
	}

	b, err := NewSynchronizer()
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	p2, err := b.NewWriteOnlyPipeline("application/x-rtp,encoding-name=H264 ! rtph264depay ! queue ! decodebin ! testsink")
	if err != nil {
		t.Fatal(err)
	}
	defer p2.Close()

	for {
		p, err := p1.ReadRTP()
		if err != nil {
			if err != io.EOF {
				t.Errorf("error reading from UDP socket: %v", err)
			}
			return
		}
		if err := p2.WriteRTP(p); err != nil {
			t.Errorf("error writing to UDP socket: %v", err)
			return
		}
	}
}
