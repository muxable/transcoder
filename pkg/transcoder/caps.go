package transcoder

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-plugins-base-1.0
#cgo LDFLAGS: -lgstsdp-1.0

#include <glib.h>
#include <gst/gst.h>
#include <gst/sdp/gstsdpmessage.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"unsafe"

	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

type Caps struct {
	caps *C.GstCaps
}

var (
	rtpmap          = C.CString("rtpmap")
	fmtp            = C.CString("fmtp")
	payload         = C.CString("payload")
	clockrate       = C.CString("clock-rate")
	media           = C.CString("media")
	ssrc            = C.CString("ssrc")
	encodingname    = C.CString("encoding-name")
	encodingparams  = C.CString("encoding-params")
	applicationxrtp = C.CString("application/x-rtp")
)

func CapsFromRTPCodecParameters(codec *webrtc.RTPCodecParameters) (*Caps, error) {
	var sdpMedia *C.GstSDPMedia

	if C.gst_sdp_media_new(&sdpMedia) != C.GstSDPResult(0) {
		return nil, errors.New("failed to create sdp media")
	}
	defer C.gst_sdp_media_free(sdpMedia)

	tokens := strings.Split(codec.MimeType, "/")

	if len(tokens) == 2 {
		cmedia := C.CString(tokens[0])
		defer C.free(unsafe.Pointer(cmedia))

		if C.gst_sdp_media_set_media(sdpMedia, cmedia) != C.GstSDPResult(0) {
			return nil, errors.New("failed to set media type")
		}

		if codec.Channels > 0 {
			crtpmap := C.CString(fmt.Sprintf("%d %s/%d/%d", codec.PayloadType, tokens[1], codec.ClockRate, codec.Channels))
			defer C.free(unsafe.Pointer(crtpmap))

			if C.gst_sdp_media_add_attribute(sdpMedia, rtpmap, crtpmap) != C.GstSDPResult(0) {
				return nil, errors.New("failed to add rtpmap attribute")
			}
		} else {
			crtpmap := C.CString(fmt.Sprintf("%d %s/%d", codec.PayloadType, tokens[1], codec.ClockRate))
			defer C.free(unsafe.Pointer(crtpmap))

			if C.gst_sdp_media_add_attribute(sdpMedia, rtpmap, crtpmap) != C.GstSDPResult(0) {
				return nil, errors.New("failed to add rtpmap attribute")
			}
		}
	}

	if codec.SDPFmtpLine != "" {
		cpt := C.CString(strconv.FormatUint(uint64(codec.PayloadType), 10))
		defer C.free(unsafe.Pointer(cpt))

		cfmtp := C.CString(fmt.Sprintf("%d %s", codec.PayloadType, codec.SDPFmtpLine))
		defer C.free(unsafe.Pointer(cfmtp))

		if C.gst_sdp_media_add_format(sdpMedia, cpt) != C.GstSDPResult(0) {
			return nil, errors.New("failed to add format")
		}

		if C.gst_sdp_media_add_attribute(sdpMedia, fmtp, cfmtp) != C.GstSDPResult(0) {
			return nil, errors.New("failed to add fmtp attribute")
		}
	}

	cdbg := C.gst_sdp_media_as_text(sdpMedia)
	defer C.free(unsafe.Pointer(cdbg))

	zap.L().Debug("SDP", zap.String("sdp", C.GoString(cdbg)))

	caps := C.gst_sdp_media_get_caps_from_media(sdpMedia, C.gint(codec.PayloadType))
	if C.gst_sdp_media_attributes_to_caps(sdpMedia, caps) != C.GstSDPResult(0) {
		return nil, errors.New("failed to add caps")
	}

	structure := C.gst_caps_get_structure(caps, C.guint(0))
	C.gst_structure_set_name(structure, applicationxrtp)

	c := &Caps{caps: caps}
	runtime.SetFinalizer(c, func(c *Caps) {
		C.gst_caps_unref(c.caps)
	})
	return c, nil
}

func (c *Caps) RTPCodecParameters() (*webrtc.RTPCodecParameters, error) {
	ccaps := C.gst_caps_to_string(c.caps)
	defer C.free(unsafe.Pointer(ccaps))

	var sdpMedia *C.GstSDPMedia
	if C.gst_sdp_media_new(&sdpMedia) != C.GstSDPResult(0) {
		return nil, errors.New("failed to create sdp media")
	}
	defer C.gst_sdp_media_free(sdpMedia)

	if C.gst_sdp_media_set_media_from_caps(c.caps, sdpMedia) != C.GstSDPResult(0) {
		return nil, errors.New("failed to set sdp media from caps")
	}

	structure := C.gst_caps_get_structure(c.caps, 0)

	pt := C.g_value_get_int(C.gst_structure_get_value(structure, payload))

	cfmtp := C.gst_sdp_media_get_attribute_val(sdpMedia, fmtp)

	fmtpstr := C.GoString(cfmtp)
	if fmtpstr != "" {
		fmtpstr = fmtpstr[len(strconv.FormatUint(uint64(pt), 10))+1:]
	}

	media := C.gst_structure_get_string(structure, media)
	if media == nil {
		return nil, errors.New("failed to get media type")
	}
	mime := C.gst_structure_get_string(structure, encodingname)
	if mime == nil {
		return nil, errors.New("failed to get encoding name")
	}

	cr := C.g_value_get_int(C.gst_structure_get_value(structure, clockrate))

	channelsStr := C.gst_structure_get_string(structure, encodingparams)
	var channels uint16
	if channelsStr != nil {
		c, err := strconv.ParseInt(C.GoString(channelsStr), 10, 16)
		if err == nil {
			channels = uint16(c)
		}
	}

	return &webrtc.RTPCodecParameters{
		PayloadType: webrtc.PayloadType(pt),
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:    fmt.Sprintf("%s/%s", C.GoString(media), C.GoString(mime)),
			ClockRate:   uint32(cr),
			Channels:    channels,
			SDPFmtpLine: fmtpstr,
		},
	}, nil
}

func (c *Caps) SSRC() (webrtc.SSRC, error) {
	structure := C.gst_caps_get_structure(c.caps, C.guint(0))

	cssrc := C.CString("ssrc")
	defer C.free(unsafe.Pointer(cssrc))

	var val C.uint

	if C.gst_structure_get_uint(structure, cssrc, &val) == C.gboolean(0) {
		return 0, errors.New("failed to get ssrc")
	}

	return webrtc.SSRC(val), nil
}

func (c *Caps) String() string {
	cstr := C.gst_caps_to_string(c.caps)
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr)
}

func CapsFromString(str string) *Caps {
	cstr := C.CString(str)
	defer C.free(unsafe.Pointer(cstr))
	c := &Caps{caps: C.gst_caps_from_string(cstr)}
	runtime.SetFinalizer(c, func(c *Caps) {
		C.gst_caps_unref(c.caps)
	})
	return c
}
