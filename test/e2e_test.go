package main

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/muxable/transcoder/internal/pipeline"
	"github.com/muxable/transcoder/internal/server"
	pkg_server "github.com/muxable/transcoder/pkg/server"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestTranscoding(t *testing.T) {
	for mime, codec := range server.SupportedCodecs {
		if mime == "video/VP8" {
		// t.Run(mime, func(t *testing.T) {
			runTranscoder(t, mime, codec)
		// })
		}
	}
}

func construct(t *testing.T, mime string, codec server.GStreamerParameters) (webrtc.RTPCodecParameters, webrtc.RTPCodecParameters, pipeline.ReadOnlyPipeline, pipeline.WriteOnlyPipeline) {
	rs, err := pipeline.NewSynchronizer()
	if err != nil {
		t.Fatal(err)
	}
	ws, err := pipeline.NewSynchronizer()
	if err != nil {
		t.Fatal(err)
	}
	oc := server.DefaultOutputCodecs[mime]
	if strings.HasPrefix(mime, "audio") {
		ic := server.DefaultOutputCodecs[webrtc.MimeTypeOpus]
		p, err := rs.NewReadOnlyPipeline(fmt.Sprintf("audiotestsrc is-live=true num-buffers=10000 ! opusenc ! rtpopuspay pt=%d mtu=1200", ic.PayloadType))
		if err != nil {
			t.Fatal(err)
		}
		q, err := ws.NewWriteOnlyPipeline(fmt.Sprintf("application/x-rtp,%s ! %s ! queue ! decodebin ! audioconvert ! autoaudiosink", codec.ToCaps(oc), codec.Depayloader))
		if err != nil {
			t.Fatal(err)
		}
		return ic, oc, p, q
	} else {
		ic := server.DefaultOutputCodecs[webrtc.MimeTypeH264]
		p, err := rs.NewReadOnlyPipeline(fmt.Sprintf("videotestsrc is-live=true num-buffers=1000 ! x264enc ! rtph264pay pt=%d mtu=1200", ic.PayloadType))
		if err != nil {
			t.Fatal(err)
		}
		q, err := ws.NewWriteOnlyPipeline(fmt.Sprintf("application/x-rtp,%s ! %s ! queue ! decodebin ! videoconvert ! autovideosink", codec.ToCaps(oc), codec.Depayloader))
		if err != nil {
			t.Fatal(err)
		}
		return ic, oc, p, q
	}
}

func runTranscoder(t *testing.T, mime string, codec server.GStreamerParameters) {
	logger := zaptest.NewLogger(t)
	defer logger.Sync()
	undo := zap.ReplaceGlobals(logger)
	defer undo()

	ic, oc, p, q := construct(t, mime, codec)
	defer p.Close()
	defer q.Close()

	tc, err := pkg_server.NewTranscoder(ic, pkg_server.ToOutputCodec(oc))
	if err != nil {
		t.Errorf("failed to create transcoder: %v", err)
		return
	}

	go func() {
		for {
			sample, err := p.ReadSample()
			if err != nil {
				if err == io.EOF {
					if err := tc.SendEndOfStream(); err != nil {
						t.Errorf("failed to send end of stream: %v", err)
					}
					return
				}
				t.Errorf("failed to read sample: %v", err)
				return
			}
			if err := tc.WriteSample(sample); err != nil {
				t.Errorf("failed to write sample: %v", err)
				return
			}
		}
	}()

	go func() {
		for {
			sample, err := tc.ReadSample()
			if err != nil {
				if err == io.EOF {
					if err := q.SendEndOfStream(); err != nil {
						t.Errorf("failed to send end of stream: %v", err)
					}
					return
				}
				t.Errorf("failed to read sample: %v", err)
				return
			}
			if err := q.WriteSample(sample); err != nil {
				t.Errorf("failed to write sample: %v", err)
				return
			}
		}
	}()

	tc.WaitForEndOfStream()
}
