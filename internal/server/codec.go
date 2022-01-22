package server

import (
	"fmt"
	"strings"

	"github.com/pion/webrtc/v3"
)

var DefaultOutputCodecs = map[string]webrtc.RTPCodecParameters{
	webrtc.MimeTypeH264: {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000},
		PayloadType: 96,
	},
	webrtc.MimeTypeH265: {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH265, ClockRate: 90000},
		PayloadType: 98,
	},
	webrtc.MimeTypeVP8: {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000},
		PayloadType: 97,
	},
	webrtc.MimeTypeVP9: {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP9, ClockRate: 90000},
		PayloadType: 99,
	},
	webrtc.MimeTypeAV1: {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeAV1, ClockRate: 90000},
		PayloadType: 100,
	},

	webrtc.MimeTypeOpus: {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		PayloadType: 101,
	},
	webrtc.MimeTypeG722: {
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 8000, Channels: 1},
		PayloadType: 9,
	},
}
type GStreamerParameters struct {
	Depayloader, DefaultEncoder, Payloader string
	ToCaps func(webrtc.RTPCodecParameters) string
}
type CodecMapping struct {
	GStreamerParameters
}

var SupportedCodecs = map[string]GStreamerParameters{
	webrtc.MimeTypeH264: {
		"rtph264depay", "x264enc speed-preset=ultrafast tune=zerolatency key-int-max=20", "rtph264pay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=H264,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
		},
	},
	webrtc.MimeTypeH265: {
		"rtph265depay", "x265enc speed-preset=ultrafast tune=zerolatency key-int-max=20", "rtph265pay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=H265,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
		},
	},
	webrtc.MimeTypeVP8: {
		"rtpvp8depay", "vp8enc deadline=1", "rtpvp8pay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=VP8,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
		},
	},
	webrtc.MimeTypeVP9: {
		"rtpvp9depay", "vp9enc deadline=1", "rtpvp9pay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=VP9,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
		},
	},
	webrtc.MimeTypeAV1: {
		"rtpav1depay", "av1enc deadline=1", "rtpav1pay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=AV1,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
		},
	},

	webrtc.MimeTypeOpus: {"rtpopusdepay", "opusenc inband-fec=true", "rtpopuspay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=OPUS,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
		},
	},
	"audio/aac": {
		"rtpmp4adepay", "avenc_aac", "rtpmp4apay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=MP4A-LATM,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
		},
	},
	"audio/mpeg": {
		"rtpmpadepay", "lamemp3enc", "rtpmpapay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=MPA,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
		},
	},
	"audio/speex": {
		"rtpspeexdepay", "speexenc", "rtpspeexpay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=SPEEX,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
		},
	},
	webrtc.MimeTypeG722: {
		"rtpg722depay", "avenc_g722", "rtpg722pay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=G722,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
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
	"audio/ac3": {
		"rtpac3depay", "avenc_ac3", "rtpac3pay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=AC3,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
		},
	},
	"audio/vorbis": {
		"rtpvorbisdepay", "vorbisenc", "rtpvorbispay",
		func(c webrtc.RTPCodecParameters) string {
			return fmt.Sprintf("encoding-name=VORBIS,clock-rate=%d,payload=%d", c.ClockRate, c.PayloadType)
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
		inputCaps := fmt.Sprintf("application/x-rtp,media=(string)audio,%s", fromParameters.ToCaps(from))
		// outputCaps := fmt.Sprintf("application/x-rtp,media=(string)video,encoding-name=(string)%s,clock-rate=(int)%d", toParameters.EncodingName, to.ClockRate)

		return fmt.Sprintf(
			"appsrc format=time name=source ! %s ! rtpjitterbuffer ! %s ! queue ! decodebin ! queue ! videoconvert ! %s ! %s ! appsink name=sink sync=false async=false",
			inputCaps, fromParameters.Depayloader, encoder, toParameters.Payloader), nil
	} else if strings.HasPrefix(from.MimeType, "audio") {
		inputCaps := fmt.Sprintf("application/x-rtp,media=(string)audio,%s", fromParameters.ToCaps(from))
		// outputCaps := fmt.Sprintf("application/x-rtp,media=(string)audio,encoding-name=(string)%s,clock-rate=(int)%d", toParameters.EncodingName, to.ClockRate)

		return fmt.Sprintf(
			"appsrc format=time is-live=true name=source ! %s ! rtpjitterbuffer ! %s ! queue ! decodebin ! queue ! audioconvert ! audioresample ! %s ! %s ! appsink name=sink sync=false async=false",
			inputCaps, fromParameters.Depayloader, toParameters.DefaultEncoder, toParameters.Payloader), nil
	}
	return "", fmt.Errorf("unsupported codec %s", from.MimeType)
}
