package transcode

import (
	"fmt"

	"github.com/muxable/transcoder/internal/gst"
)

type Synchronizer struct {
	element *gst.Pipeline
}

func NewSynchronizer() (*Synchronizer, error) {
	pipeline, err := gst.NewPipeline()
	if err != nil {
		return nil, err
	}

	pipeline.SetState(gst.StatePlaying)

	return &Synchronizer{
		element: pipeline,
	}, nil
}

func (s *Synchronizer) Close() error {
	if r := s.element.SetState(gst.StateNull); r != gst.StateChangeSuccess {
		return fmt.Errorf("failed to set state to null: %+v", r)
	}
	return nil
}
