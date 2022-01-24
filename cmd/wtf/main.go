package main

import (
	"fmt"
	"os"

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
			fmt.Println(msg)
		}
		return true
	}
}

func input() (*gst.Pipeline, *app.Sink) {
	r, err := gst.NewPipelineFromString("videotestsrc ! vp8enc ! rtpvp8pay ! appsink name=sink")
	if err != nil {
		panic(err)
	}

	sink, err := r.GetElementByName("sink")
	if err != nil {
		panic(err)
	}

	return r, app.SinkFromElement(sink)
}

func output() (*gst.Pipeline, *app.Source) {
	w, err := gst.NewPipelineFromString("appsrc name=source format=time ! rtpvp8depay ! vp8dec ! autovideosink")
	if err != nil {
		panic(err)
	}

	src, err := w.GetElementByName("source")
	if err != nil {
		panic(err)
	}

	return w, app.SrcFromElement(src)
}

func main() {
	gst.Init(&os.Args)

	mainLoop := glib.NewMainLoop(glib.MainContextDefault(), false)

	r, sink := input()
	w, src := output()

	r.GetPipelineBus().AddWatch(MonitorPipeline(mainLoop, r))
	w.GetPipelineBus().AddWatch(MonitorPipeline(mainLoop, w))

	sink.SetCallbacks(&app.SinkCallbacks{
		NewSampleFunc: func(sink *app.Sink) gst.FlowReturn {
			sample := sink.PullSample()
			if sample == nil {
				return gst.FlowEOS
			}
			return src.PushSample(sample)
		},
	})

	r.SetState(gst.StatePlaying)
	w.SetState(gst.StatePlaying)

	mainLoop.Run()
}
