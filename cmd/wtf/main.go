package main

import (
	"log"

	"github.com/muxable/transcoder/internal/gst"
)

func input() (*gst.Pipeline, *gst.Element) {
	source, err := gst.NewElement("videotestsrc")
	if err != nil {
		panic(err)
	}
	encoder, err := gst.NewElement("x264enc")
	if err != nil {
		panic(err)
	}
	sink, err := gst.NewElement("appsink")
	if err != nil {
		panic(err)
	}
	pipeline, err := gst.NewPipeline()
	if err != nil {
		panic(err)
	}

	pipeline.Add(source, encoder, sink)
	gst.Link(source, encoder, sink)

	return pipeline, sink
}

func output() (*gst.Pipeline, *gst.Element) {
	source, err := gst.NewElement("appsrc",
		gst.Property{Name: "format", Value: gst.FormatTime})
	if err != nil {
		panic(err)
	}
	decoder, err := gst.NewElement("avdec_h264")
	if err != nil {
		panic(err)
	}
	sink, err := gst.NewElement("autovideosink")
	if err != nil {
		panic(err)
	}
	pipeline, err := gst.NewPipeline()
	if err != nil {
		panic(err)
	}

	pipeline.Add(source, decoder, sink)
	gst.Link(source, decoder, sink)

	return pipeline, source
}

func main() {
	r, sink := input()
	r.SetState(gst.StatePlaying)

	w, source := output()
	w.SetState(gst.StatePlaying)

	loop := gst.MainLoopNew()

	go func() {
		for {
			s, err := sink.ReadSample()
			if err != nil {
				log.Printf("failed to pull sample: %v", err)
				break
			}
			if err := source.WriteSample(s); err != nil {
				log.Printf("failed to write sample: %v", err)
				break
			}
		}
	}()

	loop.Wait()

	log.Printf("returned")

	r.SetState(gst.StateNull)
	r.Close()
	w.SetState(gst.StateNull)
	w.Close()

	loop.Close()

	select {}
}
