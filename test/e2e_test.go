package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/muxable/transcoder/internal/codecs"
	"github.com/muxable/transcoder/pkg/transcoder"
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestTranscoding(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
	for mime, codec := range codecs.SupportedCodecs {
		t.Run(mime, func(t *testing.T) {
			if strings.HasPrefix(mime, "video") {
				runVideoTranscoder(t, mime, codec)
			}
		})
	}
}

func runVideoTranscoder(t *testing.T, mime string, codec codecs.GStreamerParameters) {
	logger := zaptest.NewLogger(t)
	defer logger.Sync()
	undo := zap.ReplaceGlobals(logger)
	defer undo()

	ptc, err := transcoder.NewTranscoder()
	if err != nil {
		t.Errorf("failed to create transcoder: %v", err)
		return
	}

	qtc, err := transcoder.NewTranscoder()
	if err != nil {
		t.Errorf("failed to create transcoder: %v", err)
		return
	}

	rtc, err := transcoder.NewTranscoder()
	if err != nil {
		t.Errorf("failed to create transcoder: %v", err)
		return
	}

	ic := codecs.DefaultOutputCodecs[webrtc.MimeTypeH264]
	p, err := ptc.NewReadOnlyPipeline(fmt.Sprintf("videotestsrc is-live=true num-buffers=100 ! video/x-raw,format=I420 ! x264enc ! rtph264pay pt=%d mtu=1200", ic.PayloadType))
	if err != nil {
		t.Fatal(err)
	}
	pcodec, err := p.Codec()
	if err != nil {
		t.Fatal(err)
	}

	qs, err := transcoder.NewPipelineBuilder(webrtc.RTPCodecTypeVideo, mime, "")
	if err != nil {
		t.Fatal(err)
	}

	q, err := qtc.NewReadWritePipeline(pcodec, qs)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := rtpio.CopyRTP(q, p); err != nil && err != io.EOF {
			t.Errorf("failed to copy rtp: %v", err)
		}
		q.Close()
	}()

	qcodec, err := q.Codec()
	if err != nil {
		t.Fatal(err)
	}

	r, err := rtc.NewWriteOnlyPipeline(qcodec, "decodebin ! autovideosink")
	if err != nil {
		t.Fatal(err)
	}

	if err := rtpio.CopyRTP(r, q); err != nil && err != io.EOF {
		t.Errorf("failed to copy rtp: %v", err)
	}
	r.Close()
}
