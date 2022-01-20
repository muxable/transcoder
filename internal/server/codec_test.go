package server

import (
	"testing"

	"github.com/muxable/transcoder/internal/gst"
)

func TestSupportedCodecs(t *testing.T) {
	for _, c := range SupportedCodecs {
		// try creating elements
		if _, err := gst.ParseBinFromDescription(c.Depayloader); err != nil {
			t.Errorf("failed to create element from %s: %v", c.Depayloader, err)
		}
		if _, err := gst.ParseBinFromDescription(c.Payloader); err != nil {
			t.Errorf("failed to create element from %s: %v", c.Payloader, err)
		}
		if _, err := gst.ParseBinFromDescription(c.DefaultEncoder); err != nil {
			t.Errorf("failed to create element from %s: %v", c.DefaultEncoder, err)
		}
	}
	// verify that invalid bins fail.
	if _, err := gst.ParseBinFromDescription("invalid"); err == nil {
		t.Errorf("expected error for invalid bin")
	}
}
