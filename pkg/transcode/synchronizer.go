package transcode

import (
	"github.com/tinyzimmer/go-gst/gst"
)

type Synchronizer struct {
	element *gst.Pipeline
}

func NewSynchronizer() (*Synchronizer, error) {
	pipeline, err := gst.NewPipeline("")
	if err != nil {
		return nil, err
	}

	pipeline.SetState(gst.StatePlaying)

	return &Synchronizer{
		element: pipeline,
	}, nil
}

func (s *Synchronizer) Close() error {
	return s.element.SetState(gst.StateNull)
}
