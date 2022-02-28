package av

/*
#cgo pkg-config: libavformat
#include <libavformat/avformat.h>
#include "demux.h"
*/
import "C"
import (
	"errors"
	"io"
	"log"
	"os"
	"unsafe"

	"github.com/mattn/go-pointer"
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

type DemuxContext struct {
	codec       webrtc.RTPCodecParameters
	avformatctx *C.AVFormatContext
	in          rtpio.RTPReader
	sdpfile     *os.File
}

var (
	csdp              = C.CString("sdp")
	csdpflags         = C.CString("sdp_flags")
	ccustomio         = C.CString("custom_io")
	creorderqueuesize = C.CString("reorder_queue_size")
)

func NewDemuxer(codec webrtc.RTPCodecParameters, in rtpio.RTPReader) *DemuxContext {
	return &DemuxContext{
		codec: codec,
		in:    in,
	}
}

//export goReadPacketFunc
func goReadPacketFunc(opaque unsafe.Pointer, buf *C.uint8_t, bufsize C.int) C.int {
	d := pointer.Restore(opaque).(*DemuxContext)
	p, err := d.in.ReadRTP()
	if err != nil {
		if err == io.EOF {
			d.sdpfile.Close()
			os.Remove(d.sdpfile.Name())
		} else {
			zap.L().Error("failed to read RTP packet", zap.Error(err))
		}
		return C.int(0)
	}

	b, err := p.Marshal()
	if err != nil {
		zap.L().Error("failed to marshal RTP packet", zap.Error(err))
		return C.int(0)
	}

	if C.int(len(b)) > bufsize {
		zap.L().Error("RTP packet too large", zap.Int("size", len(b)))
		return C.int(0)
	}

	C.memcpy(unsafe.Pointer(buf), unsafe.Pointer(&b[0]), C.ulong(len(b)))

	return C.int(len(b))
}

func (c *DemuxContext) init() error {
	fileformat := C.av_find_input_format(csdp)
	if fileformat == nil {
		return errors.New("failed to find sdp input format")
	}

	avformatctx := C.avformat_alloc_context()
	if avformatctx == nil {
		return errors.New("failed to create format context")
	}

	var opts *C.AVDictionary
	defer C.av_dict_free(&opts)
	if averr := C.av_dict_set(&opts, csdpflags, ccustomio, 0); averr < 0 {
		return av_err("av_dict_set", averr)
	}
	if averr := C.av_dict_set_int(&opts, creorderqueuesize, C.int64_t(0), 0); averr < 0 {
		return av_err("av_dict_set", averr)
	}
	// if averr := C.av_dict_set(&opts, C.CString("protocol_whitelist"), C.CString("file,udp,rtp"), 0); averr < 0 {
	// 	return av_err("av_dict_set", averr)
	// }

	sdpfile, err := NewTempSDP(c.codec)
	if err != nil {
		return err
	}

	cfilename := C.CString(sdpfile.Name())
	defer C.free(unsafe.Pointer(cfilename))

	if averr := C.avformat_open_input(&avformatctx, cfilename, fileformat, &opts); averr < C.int(0) {
		return av_err("avformat_open_input", averr)
	}

	buf := C.av_malloc(4096)
	if buf == nil {
		return errors.New("failed to allocate buffer")
	}

	avioctx := C.avio_alloc_context((*C.uchar)(buf), 4096, 0, pointer.Save(c), (*[0]byte)(C.cgoReadPacketFunc), nil, nil)
	if avioctx == nil {
		return errors.New("failed to allocate avio context")
	}

	avformatctx.pb = avioctx

	if averr := C.avformat_find_stream_info(avformatctx, nil); averr < C.int(0) {
		return av_err("avformat_find_stream_info", averr)
	}

	c.avformatctx = avformatctx
	c.sdpfile = sdpfile

	return nil
}

func (c *DemuxContext) ReadAVPacket(p *AVPacket) error {
	log.Printf("read frame")
	averr := C.av_read_frame(c.avformatctx, p.packet)
	log.Printf("read frame return %v", averr)
	if averr < 0 {
		err := av_err("av_read_frame", averr)
		if err == io.EOF {
			// TODO: is this necessary? does ffmpeg do it automatically?
			p.packet = nil
		}
		C.avformat_free_context(c.avformatctx)
		if err := c.sdpfile.Close(); err != nil {
			return err
		}
		return err
	}
	return nil
}
