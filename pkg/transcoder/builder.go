package transcoder

import (
	"fmt"
	"strings"

	"github.com/muxable/transcoder/internal/codecs"
	"github.com/pion/webrtc/v3"
)

func NewPipelineBuilder(from, to string, via string) (string, error) {
	if from == "" {
		return "", fmt.Errorf("from codec parameters cannot be nil")
	}
	if to == "" {
		if strings.HasPrefix(from, "video") {
			to = codecs.DefaultOutputCodecs[webrtc.MimeTypeH264].MimeType
		} else if strings.HasPrefix(from, "audio") {
			to = codecs.DefaultOutputCodecs[webrtc.MimeTypeOpus].MimeType
		} else {
			return "", fmt.Errorf("unsupported codec: %s", from)
		}
	}

	// identify the depacketizer and packetizer
	inputParams, ok := codecs.SupportedCodecs[from]
	if !ok {
		return "", fmt.Errorf("unsupported codec %s", from)
	}
	outputParams, ok := codecs.SupportedCodecs[to]
	if !ok {
		return "", fmt.Errorf("unsupported codec %s", to)
	}

	if via == "" {
		via = outputParams.DefaultEncoder
	}

	elements := []string{inputParams.Depayloader}

	// construct the conversion pipeline

	if strings.HasPrefix(from, "video") {
		elements = append(elements, "decodebin ! queue ! videoconvert ! videorate ! queue", via)
	} else if strings.HasPrefix(from, "audio") {
		elements = append(elements, "decodebin ! queue ! audioconvert ! audioresample ! queue", via)
	}

	elements = append(elements, outputParams.Payloader)

	return strings.Join(elements, " ! "), nil
}
