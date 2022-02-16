package codecs

import (
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
	DefaultEncoder, Payloader string
}

var SupportedCodecs = map[string]GStreamerParameters{
	webrtc.MimeTypeH264: {
		"video/x-raw,format=I420 ! x264enc speed-preset=ultrafast tune=zerolatency key-int-max=20 ! video/x-h264,stream-format=byte-stream", "rtph264pay",
	},
	webrtc.MimeTypeH265: {
		"x265enc speed-preset=ultrafast tune=zerolatency key-int-max=20", "rtph265pay",
	},
	webrtc.MimeTypeVP8: {
		"vp8enc end-usage=cq error-resilient=partitions keyframe-max-dist=10 auto-alt-ref=true cpu-used=5", "rtpvp8pay",
	},
	webrtc.MimeTypeVP9: {
		"vp9enc end-usage=cq error-resilient=partitions keyframe-max-dist=10 auto-alt-ref=true cpu-used=5", "rtpvp9pay",
	},
	// webrtc.MimeTypeAV1: {
	// 	"rtpav1depay", "av1enc deadline=1", "rtpav1pay",
	// 	func(c webrtc.RTPCodecParameters) string {
	// 		return fmt.Sprintf("encoding-name=AV1", c.ClockRate, c.PayloadType)
	// 	},
	// },

	webrtc.MimeTypeOpus: {"opusenc inband-fec=true", "rtpopuspay"},
	"audio/AAC":         {"avenc_aac", "rtpmp4apay"},
	"audio/SPEEX":       {"speexenc", "rtpspeexpay"},
	webrtc.MimeTypeG722: {"avenc_g722", "rtpg722pay"},
	webrtc.MimeTypePCMA: {"alawenc", "rtppcmapay"},
	webrtc.MimeTypePCMU: {"mulawenc", "rtppcmupay"},
	"audio/AC3":         {"avenc_ac3", "rtpac3pay"},
}
