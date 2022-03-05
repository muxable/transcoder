package av

/*
#cgo pkg-config: libavcodec
#include <libavcodec/avcodec.h>
*/
import "C"
import (
	"github.com/pion/webrtc/v3"
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
