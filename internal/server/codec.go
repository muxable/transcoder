package server

import (
	"fmt"
	"strings"

	"github.com/pion/webrtc/v3"
)

// mirroring https://chromium.googlesource.com/external/webrtc/+/95eb1ba0db79d8fd134ae61b0a24648598684e8a/webrtc/media/engine/payload_type_mapper.cc#27
//
// TODO: a better approach to this would be to list all supported codecs (including duplicates) and let WebRTC negotiate the codec
// then pass that to the transcoder. we will need to implement our own TrackLocal for this.
var DefaultOutputCodecs = map[string]webrtc.RTPCodecParameters{
	webrtc.MimeTypePCMU: {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMU, ClockRate: 8000, Channels: 1},
		PayloadType:        0,
	},
	"audio/GSM": {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/GSM", ClockRate: 8000, Channels: 1},
		PayloadType:        3,
	},
	"audio/G723": {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/G723", ClockRate: 8000, Channels: 1},
		PayloadType:        4,
	},
	"audio/LPC": {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/LPC", ClockRate: 8000, Channels: 1},
		PayloadType:        7,
	},
	webrtc.MimeTypePCMA: {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMA, ClockRate: 8000, Channels: 1},
		PayloadType:        8,
	},
	webrtc.MimeTypeG722: {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeG722, ClockRate: 8000, Channels: 1},
		PayloadType:        9,
	},
	"audio/L16": {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/L16", ClockRate: 44100, Channels: 2},
		PayloadType:        10,
	},
	"audio/QCELP": {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/QCELP", ClockRate: 8000, Channels: 1},
		PayloadType:        12,
	},
	"audio/CN": {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/CN", ClockRate: 8000, Channels: 1},
		PayloadType:        13,
	},
	"audio/MPA": {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/MPA", ClockRate: 90000, Channels: 1},
		PayloadType:        14,
	},
	"audio/G728": {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/G728", ClockRate: 8000, Channels: 1},
		PayloadType:        15,
	},
	"audio/DVI4": {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/DVI4", ClockRate: 22050, Channels: 1},
		PayloadType:        17,
	},
	"audio/G729": {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/G729", ClockRate: 8000, Channels: 1},
		PayloadType:        18,
	},

	webrtc.MimeTypeVP8: {
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeVP8,
			ClockRate:    90000,
			RTCPFeedback: []webrtc.RTCPFeedback{{Type: "goog-remb", Parameter: ""}, {Type: "ccm", Parameter: "fir"}, {Type: "nack", Parameter: ""}, {Type: "nack", Parameter: "pli"}},
		},
		PayloadType: 100,
	},
	webrtc.MimeTypeVP9: {
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeVP9,
			ClockRate:    90000,
			RTCPFeedback: []webrtc.RTCPFeedback{{Type: "goog-remb", Parameter: ""}, {Type: "ccm", Parameter: "fir"}, {Type: "nack", Parameter: ""}, {Type: "nack", Parameter: "pli"}},
		},
		PayloadType: 101,
	},
	webrtc.MimeTypeH264: {
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeH264,
			ClockRate:    90000,
			RTCPFeedback: []webrtc.RTCPFeedback{{Type: "goog-remb", Parameter: ""}, {Type: "ccm", Parameter: "fir"}, {Type: "nack", Parameter: ""}, {Type: "nack", Parameter: "pli"}},
		},
		PayloadType: 102,
	},
	webrtc.MimeTypeH265: {
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeH265,
			ClockRate:    90000,
			RTCPFeedback: []webrtc.RTCPFeedback{{Type: "goog-remb", Parameter: ""}, {Type: "ccm", Parameter: "fir"}, {Type: "nack", Parameter: ""}, {Type: "nack", Parameter: "pli"}},
		},
		PayloadType: 103,
	},
	webrtc.MimeTypeAV1: {
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeAV1,
			ClockRate:    90000,
			RTCPFeedback: []webrtc.RTCPFeedback{{Type: "goog-remb", Parameter: ""}, {Type: "ccm", Parameter: "fir"}, {Type: "nack", Parameter: ""}, {Type: "nack", Parameter: "pli"}},
		},
		PayloadType: 104,
	},

	webrtc.MimeTypeOpus: {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		PayloadType:        111,
	},
	"audio/AC3": {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/AC3", ClockRate: 48000, Channels: 1},
		PayloadType:        112,
	},
	"audio/VORBIS": {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/VORBIS", ClockRate: 90000, Channels: 1},
		PayloadType:        113,
	},
	"audio/AAC": {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/AAC", ClockRate: 48000, Channels: 2},
		PayloadType:        114,
	},
	"audio/SPEEX": {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/SPEEX", ClockRate: 48000, Channels: 1},
		PayloadType:        115,
	},
}

type GStreamerParameters struct {
	Depayloader, DefaultEncoder, Payloader string
	ToCaps                                 func(webrtc.RTPCodecParameters) string
}
type CodecMapping struct {
	GStreamerParameters
}

var SupportedCodecs = map[string]GStreamerParameters{
	webrtc.MimeTypeH264: {
		"rtph264depay", "video/x-raw,format=I420 ! x264enc pass=qual tune=zerolatency key-int-max=20", "rtph264pay config-interval=1",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=H264,clock-rate=%d,payload=%d,packetization-mode=(string)1,profile-level-id=(string)42001f", c.ClockRate, c.PayloadType)
		},
	},
	webrtc.MimeTypeH265: {
		"rtph265depay", "video/x-raw,format=I420 ! x265enc pass=qual speed-preset=ultrafast tune=zerolatency key-int-max=20", "rtph265pay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=H265,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
		},
	},
	webrtc.MimeTypeVP8: {
		"rtpvp8depay", "vp8enc end-usage=cq error-resilient=partitions keyframe-max-dist=10 auto-alt-ref=true cpu-used=5", "rtpvp8pay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=VP8,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
		},
	},
	webrtc.MimeTypeVP9: {
		"rtpvp9depay", "vp9enc end-usage=cq error-resilient=partitions keyframe-max-dist=10 auto-alt-ref=true cpu-used=5", "rtpvp9pay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=VP9,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
		},
	},
	// webrtc.MimeTypeAV1: {
	// 	"rtpav1depay", "av1enc deadline=1", "rtpav1pay",
	// 	func(c webrtc.RTPCodecParameters) string {
	// 		return fmt.Sprintf("encoding-name=AV1,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
	// 	},
	// },

	webrtc.MimeTypeOpus: {"rtpopusdepay", "opusenc inband-fec=true", "rtpopuspay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=OPUS,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
		},
	},
	"audio/AAC": {
		"rtpmp4adepay", "avenc_aac", "rtpmp4apay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=MP4A-LATM,clock-rate=%d,payload=%d,cpresent=(string)0,config=(string)40002320", c.ClockRate, c.PayloadType)
		},
	},
	"audio/SPEEX": {
		"rtpspeexdepay", "speexenc", "rtpspeexpay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=SPEEX,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
		},
	},
	webrtc.MimeTypeG722: {
		"rtpg722depay", "avenc_g722", "rtpg722pay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
		},
	},
	webrtc.MimeTypePCMA: {
		"rtppcmadepay", "alawenc", "rtppcmapay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=PCMA,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
		},
	},
	webrtc.MimeTypePCMU: {
		"rtppcmudepay", "mulawenc", "rtppcmupay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=PCMU,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
		},
	},
	"audio/AC3": {
		"rtpac3depay", "avenc_ac3", "rtpac3pay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=AC3,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
		},
	},
}

func PipelineString(from, to webrtc.RTPCodecParameters, encoder string) (string, error) {
	fromParameters, ok := SupportedCodecs[from.MimeType]
	if !ok {
		return "", fmt.Errorf("unsupported codec %s", from.MimeType)
	}
	toParameters, ok := SupportedCodecs[to.MimeType]
	if !ok {
		return "", fmt.Errorf("unsupported codec %s", to.MimeType)
	}

	if encoder == "" {
		encoder = toParameters.DefaultEncoder
	}

	if strings.HasPrefix(from.MimeType, "video") {
		inputCaps := fmt.Sprintf("application/x-rtp,media=(string)video,%s", fromParameters.ToCaps(from))

		return fmt.Sprintf(
			"appsrc is-live=true format=time name=source ! %s ! rtpjitterbuffer ! %s ! queue ! decodebin ! queue ! videoconvert ! videorate ! %s ! %s ! queue ! appsink name=sink sync=false async=false",
			inputCaps, fromParameters.Depayloader, encoder, toParameters.Payloader), nil
	} else if strings.HasPrefix(from.MimeType, "audio") {
		inputCaps := fmt.Sprintf("application/x-rtp,media=(string)audio,%s", fromParameters.ToCaps(from))

		return fmt.Sprintf(
			"appsrc is-live=true format=time name=source ! %s ! rtpjitterbuffer ! %s ! queue ! decodebin ! queue ! audioconvert ! audioresample ! %s ! %s mtu=1200 ! appsink name=sink sync=false async=false",
			inputCaps, fromParameters.Depayloader, encoder, toParameters.Payloader), nil
	}
	return "", fmt.Errorf("unsupported codec %s", from.MimeType)

}
