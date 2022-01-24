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
  encoder, err := gst.FactoryElementMake("vp8enc")
  if err != nil {
    panic(err)
  }
  decoder, err := gst.FactoryElementMake("vp8dec")
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

  pipeline.Add(source, encoder, decoder, sink)

  gst.Link(source, encoder, decoder, sink)

  pipeline.SetState(gst.StatePlaying)

  loop := gst.MainLoopNew()

  loop.Wait()
  
  log.Printf("returned")
  pipeline.SetState(gst.StateNull)

  pipeline.Close()
  loop.Close()

  select{}
}