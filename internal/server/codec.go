package server

import (
	"fmt"
	"strings"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type OutputCodec struct {
	webrtc.RTPCodecCapability
	rtp.Payloader
	GStreamerEncoder string
}

var DefaultOutputCodecs = map[string]webrtc.RTPCodecCapability{
	"audio": {MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
	"video": {MimeType: webrtc.MimeTypeH264, ClockRate: 90000},

	webrtc.MimeTypeH264: {MimeType: webrtc.MimeTypeH264, ClockRate: 90000},
	webrtc.MimeTypeH265: {MimeType: webrtc.MimeTypeH265, ClockRate: 90000},
	webrtc.MimeTypeVP8: {MimeType: webrtc.MimeTypeVP8, ClockRate: 90000},
	webrtc.MimeTypeVP9: {MimeType: webrtc.MimeTypeVP9, ClockRate: 90000},
	webrtc.MimeTypeAV1: {MimeType: webrtc.MimeTypeAV1, ClockRate: 90000},
	webrtc.MimeTypeOpus: {MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
}
type GStreamerParameters struct {
	EncodingName, Depayloader, DefaultEncoder, Payloader string
}
type CodecMapping struct {
	GStreamerParameters
}

var SupportedCodecs = map[string]GStreamerParameters{
	webrtc.MimeTypeH264: {"H264", "rtph264depay", "x264enc speed-preset=ultrafast tune=zerolatency key-int-max=20", "rtph264pay"},
	webrtc.MimeTypeH265: {"H265", "rtph265depay", "x265enc speed-preset=ultrafast tune=zerolatency key-int-max=20", "rtph265pay"},
	webrtc.MimeTypeVP8: {"VP8", "rtpvp8depay", "vp8enc deadline=1", "rtpvp8pay"},
	webrtc.MimeTypeVP9: {"VP9", "rtpvp9depay", "vp9enc deadline=1", "rtpvp9pay"},
	webrtc.MimeTypeAV1: {"AV1", "rtpav1depay", "av1enc deadline=1", "rtpav1pay"},

	webrtc.MimeTypeOpus: {"OPUS", "rtpopusdepay", "opusenc inband-fec=true", "rtpopuspay"},
	"audio/aac": {"MP4A-LATM", "rtpmp4adepay", "avenc_aac", "rtpmp4apay"},
	"audio/mpeg": {"MPEG", "rtpmpadepay", "lamemp3enc", "rtpmpapay"},
	"audio/speex": {"SPEEX", "rtpspeexdepay", "speexenc", "rtpspeexpay"},
	webrtc.MimeTypeG722: {"G722", "rtpg722depay", "avenc_g722", "rtpg722pay"},
	webrtc.MimeTypePCMA: {"PCMA", "rtppcmadepay", "alawenc", "rtppcmapay"},
	webrtc.MimeTypePCMU: {"PCMU", "rtppcmudepay", "mulawenc", "rtppcmupay"},
	"audio/ac3": {"AC3", "rtpac3depay", "avenc_ac3", "rtpac3pay"},
	"audio/vorbis": {"VORBIS", "rtpvorbisdepay", "vorbisenc", "rtpvorbispay"},
}

func PipelineString(from webrtc.RTPCodecParameters, to webrtc.RTPCodecCapability, encoder string) (string, error) {
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
		inputCaps := fmt.Sprintf("application/x-rtp,media=(string)video,encoding-name=(string)%s,clock-rate=(int)%d", fromParameters.EncodingName, from.ClockRate)
		outputCaps := fmt.Sprintf("application/x-rtp,media=(string)video,encoding-name=(string)%s,clock-rate=(int)%d", toParameters.EncodingName, to.ClockRate)

		return fmt.Sprintf(
			"appsrc format=time name=source ! %s ! rtpjitterbuffer ! %s ! queue ! decodebin ! queue ! videoconvert ! %s ! queue ! %s ! %s ! appsink name=sink",
			inputCaps, fromParameters.Depayloader, encoder, toParameters.Payloader, outputCaps), nil
	} else if strings.HasPrefix(from.MimeType, "audio") {
		inputCaps := fmt.Sprintf("application/x-rtp,media=(string)audio,encoding-name=(string)%s,clock-rate=(int)%d,payload=(int)%d", fromParameters.EncodingName, from.ClockRate, from.PayloadType)
		// outputCaps := fmt.Sprintf("application/x-rtp,media=(string)audio,encoding-name=(string)%s,clock-rate=(int)%d", toParameters.EncodingName, to.ClockRate)

		return fmt.Sprintf(
			"appsrc format=time name=source ! %s ! rtpjitterbuffer ! %s ! queue ! decodebin ! queue ! audioconvert ! audioresample ! %s ! %s pt=96 ! appsink name=sink",
			inputCaps, fromParameters.Depayloader, toParameters.DefaultEncoder, toParameters.Payloader), nil
	}
	return "", fmt.Errorf("unsupported codec %s", from.MimeType)
}
