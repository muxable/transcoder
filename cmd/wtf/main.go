package main

import (
	"log"

	"github.com/muxable/transcoder/internal/gst"
)

func main() {
  source, err := gst.FactoryElementMake("videotestsrc")
  if err != nil {
    panic(err)
  }
  sink, err := gst.FactoryElementMake("autovideosink")
  if err != nil {
    panic(err)
  }
  pipeline, err := gst.PipelineNew()
  if err != nil {
    panic(err)
  }

  pipeline.Add(source, sink)

  source.Link(sink)

  pipeline.SetState(gst.StatePlaying)

  loop := gst.MainLoopNew()

  loop.Wait()
  
  log.Printf("returned")
  pipeline.SetState(gst.StateNull)

  pipeline.Close()
  loop.Close()

  select{}
}