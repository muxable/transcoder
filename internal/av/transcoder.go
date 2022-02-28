package av

/*
#cgo pkg-config: libavutil
#include <libavutil/log.h>
*/
import "C"
import (
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
)

func init() {
	C.av_log_set_level(24)
}

type Transcoder struct {
	rtpio.RTPWriteCloser
	rtpio.RTPReader
}

func NewTranscoder(from webrtc.RTPCodecParameters, to webrtc.RTPCodecCapability) (*Transcoder, error) {
	r, w := rtpio.RTPPipe()
	demux := NewDemuxer(from, r)
	decode := NewDecoder(from, demux)
	encode := NewEncoder(to, decode)
	mux := NewMuxer(to, encode)

	return &Transcoder{
		RTPWriteCloser: w,
		RTPReader: mux,
	}, nil
}
