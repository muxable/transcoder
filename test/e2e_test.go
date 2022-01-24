package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/muxable/transcoder/internal/server"
	"github.com/muxable/transcoder/pkg/transcode"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/tinyzimmer/go-glib/glib"
	"github.com/tinyzimmer/go-gst/gst"
	"github.com/tinyzimmer/go-gst/gst/app"
)

func MonitorPipeline(mainLoop *glib.MainLoop, pipeline *gst.Pipeline) func(msg *gst.Message) bool {
	return func(msg *gst.Message) bool {
		switch msg.Type() {
		case gst.MessageEOS:
			pipeline.BlockSetState(gst.StateNull)
			mainLoop.Quit()
		case gst.MessageError:
			err := msg.ParseError()
			fmt.Println("ERROR:", err.Error())
			if debug := err.DebugString(); debug != "" {
				fmt.Println("DEBUG:", debug)
			}
			mainLoop.Quit()
		default:
			// fmt.Println(msg)
		}
		return true
	}
}

func input(ps string) (*gst.Pipeline, *app.Sink) {
	r, err := gst.NewPipelineFromString(ps)
	if err != nil {
		panic(err)
	}

	sink, err := r.GetElementByName("sink")
	if err != nil {
		panic(err)
	}

	return r, app.SinkFromElement(sink)
}

func output(ps string) (*gst.Pipeline, *app.Source, *gst.Element) {
	w, err := gst.NewPipelineFromString(ps)
	if err != nil {
		panic(err)
	}

	src, err := w.GetElementByName("source")
	if err != nil {
		panic(err)
	}

	test, err := w.GetElementByName("test")
	if err != nil {
		panic(err)
	}

	return w, app.SrcFromElement(src), test
}

func TestTranscoding(t *testing.T) {
	gst.Init(&os.Args)

	for mime, codec := range server.SupportedCodecs {
		t.Run(mime, func(t *testing.T) {

			mainLoop := glib.NewMainLoop(glib.MainContextDefault(), false)

			var ic webrtc.RTPCodecParameters
			oc := server.DefaultOutputCodecs[mime]

			var rs, ws string
			if strings.HasPrefix(mime, "audio") {
				ic = server.DefaultOutputCodecs[webrtc.MimeTypeOpus]
				rs = fmt.Sprintf("audiotestsrc num-buffers=10 ! opusenc ! rtpopuspay pt=%d mtu=1200 ! appsink name=sink", server.DefaultOutputCodecs[webrtc.MimeTypeOpus].PayloadType)
				ws = fmt.Sprintf("appsrc format=time name=source ! application/x-rtp,%s ! %s ! queue ! decodebin ! audioconvert ! testsink name=test", codec.ToCaps(oc), codec.Depayloader)
			} else {
				ic = server.DefaultOutputCodecs[webrtc.MimeTypeVP8]
				rs = fmt.Sprintf("videotestsrc num-buffers=10 ! vp8enc ! rtpvp8pay pt=%d mtu=1200 ! appsink name=sink", server.DefaultOutputCodecs[webrtc.MimeTypeVP8].PayloadType)
				ws = fmt.Sprintf("appsrc format=time name=source ! application/x-rtp,%s ! %s ! queue ! decodebin ! videoconvert ! testsink name=test", codec.ToCaps(oc), codec.Depayloader)
			}

			r, sink := input(rs)
			w, src, test := output(ws)

			tc, err := transcode.NewTranscoder(
				ic,
				transcode.ToOutputCodec(oc))
			if err != nil {
				t.Errorf("failed to create transcoder: %v", err)
				return
			}

			r.GetPipelineBus().AddWatch(MonitorPipeline(mainLoop, r))
			w.GetPipelineBus().AddWatch(MonitorPipeline(mainLoop, w))

			sink.SetCallbacks(&app.SinkCallbacks{
				NewSampleFunc: func(sink *app.Sink) gst.FlowReturn {
					sample := sink.PullSample()
					if sample == nil {
						return gst.FlowEOS
					}
					buffer := sample.GetBuffer()
					if buffer == nil {
						return gst.FlowError
					}
					buf := buffer.Map(gst.MapRead).Bytes()
					defer buffer.Unmap()

					p := &rtp.Packet{}
					if err := p.Unmarshal(buf); err != nil {
						t.Errorf("failed to unmarshal packet: %v", err)
						return gst.FlowError
					}
					if err := tc.WriteRTP(p); err != nil {
						t.Errorf("failed to write packet: %v", err)
						return gst.FlowError
					}
					return gst.FlowOK
				},
			})

			src.SetCallbacks(&app.SourceCallbacks{
				NeedDataFunc: func(src *app.Source, length uint) {
					p, err := tc.ReadRTP()
					if err != nil {
						t.Errorf("failed to read packet: %v", err)
						return
					}
					buf, err := p.Marshal()
					if err != nil {
						t.Errorf("failed to marshal packet: %v", err)
						return
					}

					buffer := gst.NewBufferWithSize(int64(len(buf)))

					buffer.Map(gst.MapWrite).WriteData(buf)
					buffer.Unmap()

					src.PushBuffer(buffer)
				},
			})

			r.SetState(gst.StatePlaying)
			w.SetState(gst.StatePlaying)
			defer r.SetState(gst.StateNull)
			defer w.SetState(gst.StateNull)

			mainLoop.Run()

			bc, err := test.GetProperty("buffer-count")
			if err != nil {
				t.Errorf("failed to get buffer count: %v", err)
				return
			}
			if bc.(int64) == 0 {
				t.Errorf("buffer count is %d, expected >0", bc.(int64))
				return
			}
		})
	}
}
