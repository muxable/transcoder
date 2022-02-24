package transcoder

import (
	"fmt"

	"github.com/muxable/transcoder/internal/codecs"
	"github.com/pion/webrtc/v3"
)

func NewPipelineBuilder(kind webrtc.RTPCodecType, to string, via string) (string, error) {
	if to == "" {
		switch kind {
		case webrtc.RTPCodecTypeVideo:
			to = codecs.DefaultOutputCodecs[webrtc.MimeTypeH264].MimeType
		case webrtc.RTPCodecTypeAudio:
			to = codecs.DefaultOutputCodecs[webrtc.MimeTypeOpus].MimeType
		}
	}

	// identify the depacketizer and packetizer
	outputParams, ok := codecs.SupportedCodecs[to]
	if !ok {
		return "", fmt.Errorf("unsupported codec %s", to)
	}

	if via == "" {
		via = outputParams.DefaultEncoder
	}

	switch kind {
	case webrtc.RTPCodecTypeVideo:
		return fmt.Sprintf("decodebin ! queue ! videoconvert ! queue ! %s ! %s mtu=1200", via, outputParams.Payloader), nil
	case webrtc.RTPCodecTypeAudio:
		return fmt.Sprintf("decodebin ! queue ! audioconvert ! queue ! %s ! %s mtu=1200", via, outputParams.Payloader), nil
	}
	return "", fmt.Errorf("unsupported codec %s", to)
}
