package av

/*
#cgo pkg-config: libavcodec

#include <libavcodec/avcodec.h>
*/
import "C"
import (
	"errors"
	"io"

	h265writer "github.com/muxable/rtptools/pkg/h265/writer"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/h264writer"
	"github.com/pion/webrtc/v3/pkg/media/ivfwriter"
	"github.com/pion/webrtc/v3/pkg/media/oggwriter"
)

var AvCodec = map[string]uint32 {
	webrtc.MimeTypeVP8: C.AV_CODEC_ID_VP8,
	webrtc.MimeTypeVP9: C.AV_CODEC_ID_VP9,
	webrtc.MimeTypeH264: C.AV_CODEC_ID_H264,
	webrtc.MimeTypeH265: C.AV_CODEC_ID_HEVC,
	webrtc.MimeTypeG722: C.AV_CODEC_ID_ADPCM_G722,
	webrtc.MimeTypeOpus: C.AV_CODEC_ID_OPUS,
	webrtc.MimeTypePCMU: C.AV_CODEC_ID_PCM_MULAW,
	webrtc.MimeTypePCMA: C.AV_CODEC_ID_PCM_ALAW,
}

func MediaWriter(w io.Writer, codec webrtc.RTPCodecCapability) (media.Writer, error) {
	switch codec.MimeType {
	case webrtc.MimeTypeVP8, webrtc.MimeTypeVP9:
		return ivfwriter.NewWith(w)
	case webrtc.MimeTypeH264:
		return h264writer.NewWith(w), nil
	case webrtc.MimeTypeH265:
		return h265writer.NewWith(w), nil
	case webrtc.MimeTypeOpus:
		return oggwriter.NewWith(w, codec.ClockRate, codec.Channels)
	}
	return nil, errors.New("unsupported mime type")
}