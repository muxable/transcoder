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
)

func TestTranscoding(t *testing.T) {
	for mime, codec := range server.SupportedCodecs {
		t.Run(mime, func(t *testing.T) {
			runTranscoder(t, mime, codec)
		})
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
		p, err := rs.NewReadOnlyPipeline(fmt.Sprintf("audiotestsrc num-buffers=100 ! opusenc ! rtpopuspay pt=%d mtu=1200", ic.PayloadType))
		if err != nil {
			t.Fatal(err)
		}
		q, err := ws.NewWriteOnlyPipeline(fmt.Sprintf("application/x-rtp,%s ! %s ! queue ! decodebin ! audioconvert ! testsink name=test", codec.ToCaps(oc), codec.Depayloader))
		if err != nil {
			t.Fatal(err)
		}
		return ic, oc, p, q
	} else {
		ic := server.DefaultOutputCodecs[webrtc.MimeTypeVP8]
		p, err := rs.NewReadOnlyPipeline(fmt.Sprintf("videotestsrc num-buffers=100 ! vp8enc ! rtpvp8pay pt=%d mtu=1200", ic.PayloadType))
		if err != nil {
			t.Fatal(err)
		}
		q, err := ws.NewWriteOnlyPipeline(fmt.Sprintf("application/x-rtp,%s ! %s ! queue ! decodebin ! videoconvert ! testsink name=test", codec.ToCaps(oc), codec.Depayloader))
		if err != nil {
			t.Fatal(err)
		}
		return ic, oc, p, q
	}
}

func runTranscoder(t *testing.T, mime string, codec server.GStreamerParameters) {
	ic, oc, p, q := construct(t, mime, codec)

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
