package main

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/muxable/transcoder/internal/codecs"
	"github.com/muxable/transcoder/internal/pipeline"
	pkg_server "github.com/muxable/transcoder/pkg/server"
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestTranscoding(t *testing.T) {
	t.Skip("test must be manually run")
	for mime, codec := range codecs.SupportedCodecs {
		t.Run(mime, func(t *testing.T) {
			runTranscoder(t, mime, codec)
		})
	}
}

func construct(t *testing.T, mime string, codec codecs.GStreamerParameters) (webrtc.RTPCodecParameters, webrtc.RTPCodecParameters, pipeline.ReadOnlyPipeline, pipeline.WriteOnlyPipeline) {
	rs, err := pipeline.NewSynchronizer()
	if err != nil {
		t.Fatal(err)
	}
	ws, err := pipeline.NewSynchronizer()
	if err != nil {
		t.Fatal(err)
	}
	oc := codecs.DefaultOutputCodecs[mime]
	if strings.HasPrefix(mime, "audio") {
		ic := codecs.DefaultOutputCodecs[webrtc.MimeTypeOpus]
		p, err := rs.NewReadOnlyPipeline(fmt.Sprintf("audiotestsrc is-live=true num-buffers=100 ! opusenc ! rtpopuspay pt=%d mtu=1200", ic.PayloadType))
		if err != nil {
			t.Fatal(err)
		}
		q, err := ws.NewWriteOnlyPipeline(fmt.Sprintf("application/x-rtp,%s,clock-rate=%d,payload=%d ! queue ! decodebin ! audioconvert ! autoaudiosink", codec.Caps, ic.ClockRate, ic.PayloadType))
		if err != nil {
			t.Fatal(err)
		}
		return ic, oc, p, q
	} else {
		ic := codecs.DefaultOutputCodecs[webrtc.MimeTypeH264]
		p, err := rs.NewReadOnlyPipeline(fmt.Sprintf("videotestsrc is-live=true num-buffers=100 ! x264enc ! rtph264pay pt=%d mtu=1200", ic.PayloadType))
		if err != nil {
			t.Fatal(err)
		}
		q, err := ws.NewWriteOnlyPipeline(fmt.Sprintf("application/x-rtp,%s,clock-rate=%d,payload=%d ! queue ! decodebin ! videoconvert ! autovideosink", codec.Caps, ic.ClockRate, ic.PayloadType))
		if err != nil {
			t.Fatal(err)
		}
		return ic, oc, p, q
	}
}

func runTranscoder(t *testing.T, mime string, codec codecs.GStreamerParameters) {
	logger := zaptest.NewLogger(t)
	defer logger.Sync()
	undo := zap.ReplaceGlobals(logger)
	defer undo()

	ic, oc, p, q := construct(t, mime, codec)

	tc, err := pkg_server.NewTranscoder(ic, pkg_server.ToOutputCodec(oc))
	if err != nil {
		t.Errorf("failed to create transcoder: %v", err)
		return
	}

	go func() {
		if err := rtpio.CopyRTP(tc, p); err != nil && err != io.EOF {
			t.Errorf("failed to copy rtp: %v", err)
		}
		tc.Close()
	}()

	if err := rtpio.CopyRTP(q, tc); err != nil && err != io.EOF {
		t.Errorf("failed to copy rtp: %v", err)
	}
	q.Close()
}
