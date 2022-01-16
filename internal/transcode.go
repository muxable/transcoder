package internal

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0

#include "transcode.h"
*/
import "C"
import (
	"fmt"
	"io"

	"github.com/notedit/gst"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

func ToRTPCaps(codec webrtc.RTPCodecParameters) string {
	switch codec.MimeType {
	// video codecs
	case "video/H265":
		return fmt.Sprintf("application/x-rtp,media=(string)video,clock-rate=(int)%d,encoding-name=(string)H265,payload=(int)%d", codec.ClockRate, codec.PayloadType)
	case webrtc.MimeTypeH264:
		return fmt.Sprintf("application/x-rtp,media=(string)video,clock-rate=(int)%d,encoding-name=(string)H264,payload=(int)%d", codec.ClockRate, codec.PayloadType)
	case webrtc.MimeTypeVP8:
		return fmt.Sprintf("application/x-rtp,media=(string)video,clock-rate=(int)%d,encoding-name=(string)VP8,payload=(int)%d", codec.ClockRate, codec.PayloadType)
	case webrtc.MimeTypeVP9:
		return fmt.Sprintf("application/x-rtp,media=(string)video,clock-rate=(int)%d,encoding-name=(string)VP9,payload=(int)%d", codec.ClockRate, codec.PayloadType)
		// audio codecs
	case webrtc.MimeTypeOpus:
		return fmt.Sprintf("application/x-rtp,media=(string)audio,clock-rate=(int)%d,encoding-name=(string)OPUS,payload=(int)%d", codec.ClockRate, codec.PayloadType)
	case "audio/aac":
		return fmt.Sprintf("application/x-rtp,media=(string)audio,clock-rate=(int)%d,encoding-name=(string)MP4A-LATM,payload=(int)%d", codec.ClockRate, codec.PayloadType)
	case webrtc.MimeTypeG722:
		return fmt.Sprintf("application/x-rtp,media=(string)audio,clock-rate=(int)%d,encoding-name=(string)G722,payload=(int)%d", codec.ClockRate, codec.PayloadType)
	}
	return "application/x-rtp"
}

func PipelineString(trCodec webrtc.RTPCodecParameters, encodingStr string) (string, error) {
	appsrc := fmt.Sprintf("appsrc format=time name=source ! %s ! queue", ToRTPCaps(trCodec))
	// appsink outputs rtp's because the pion h264 payloader is more reliable and easier to debug
	// than the gstreamer payloader.
	appsink := encodingStr + " ! queue ! appsink name=sink"

	switch trCodec.MimeType {
	// video codecs
	case "video/H265":
		return appsrc + ` ! rtpjitterbuffer ! rtph265depay ! decodebin ! queue ! videoconvert ! ` + appsink, nil
	case webrtc.MimeTypeH264:
		return appsrc + ` ! rtpjitterbuffer ! rtph264depay ! decodebin ! queue ! videoconvert ! ` + appsink, nil
	case webrtc.MimeTypeVP8:
		return appsrc + ` ! rtpjitterbuffer ! rtpvp8depay ! decodebin ! queue ! videoconvert ! ` + appsink, nil
	case webrtc.MimeTypeVP9:
		return appsrc + ` ! rtpjitterbuffer ! rtpvp9depay ! decodebin ! queue ! videoconvert ! ` + appsink, nil
	// audio codecs
	case webrtc.MimeTypeOpus:
		return appsrc + ` ! rtpjitterbuffer ! rtpopusdepay ! decodebin ! queue ! audioconvert ! audioresample ! ` + appsink, nil
	case "audio/aac":
		return appsrc + ` ! rtpjitterbuffer ! rtpmp4adepay ! decodebin ! queue ! audioconvert ! audioresample ! ` + appsink, nil
	case webrtc.MimeTypeG722:
		return appsrc + ` ! rtpjitterbuffer ! rtpg722depay ! decodebin ! queue ! audioconvert ! audioresample ! ` + appsink, nil
	}
	return "", fmt.Errorf("unsupported codec %s", trCodec.MimeType)
}

func EncodingPipelineStr(mimeType string) (string, error) {
	switch mimeType {
	case webrtc.MimeTypeH264, "" /* default */:
		return "x264enc speed-preset=ultrafast tune=zerolatency key-int-max=20", nil
	case webrtc.MimeTypeVP8:
		return "vp8enc deadline=1", nil
	case webrtc.MimeTypeOpus:
		return "opusenc", nil
	}
	return "", fmt.Errorf("unsupported codec %s", mimeType)
}

func TargetCodec(mimeType string) (*webrtc.RTPCodecCapability, rtp.Payloader, error) {
	switch mimeType {
	case webrtc.MimeTypeH264, "" /* default */:
		return &webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000}, &codecs.H264Payloader{}, nil
	case webrtc.MimeTypeVP8:
		return &webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}, &codecs.VP8Payloader{}, nil
	case webrtc.MimeTypeOpus:
		return &webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000}, &codecs.OpusPayloader{}, nil
	}
	return nil, nil, fmt.Errorf("unsupported codec %s", mimeType)
}


func TranscodeTrackRemote(parent *gst.Pipeline, tr *webrtc.TrackRemote, pipelineStr, mimeType string) (webrtc.TrackLocal, error) {
	targetCodec, payloader, err := TargetCodec(mimeType)
	if err != nil {
		return nil, err
	}

	packetizer := NewTSPacketizer(1200, payloader, rtp.NewRandomSequencer())

	tl, err := webrtc.NewTrackLocalStaticRTP(*targetCodec, tr.ID(), tr.StreamID())
	if err != nil {
		return nil, err
	}

	if pipelineStr == "" {
		pipelineStr, err = EncodingPipelineStr(mimeType)
		if err != nil {
			return nil, err
		}
	}

	transcodingPipelineStr, err := PipelineString(tr.Codec(), pipelineStr)
	if err != nil {
		return nil, err
	}

	bin, err := gst.ParseBinFromDescription(transcodingPipelineStr, false)
	if err != nil {
		return nil, err
	}

	parent.Add(&bin.Element)

	bin.SetState(gst.StatePlaying)

	source := bin.GetByName("source")
	sink := bin.GetByName("sink")

	go func() {
		buf := make([]byte, 1400)
		for source != nil {
			i, _, err := tr.Read(buf)
			if err != nil {
				if err == io.EOF {
					if err := source.EndOfStream(); err != nil {
						zap.L().Error("could not end of stream", zap.Error(err))
					}
					return
				}
				zap.L().Error("could not read rtp", zap.Error(err))
				return
			}
			if err := source.PushBuffer(buf[:i]); err != nil {
				zap.L().Error("could not push buffer", zap.Error(err))
			}
		}
	}()

	go func() {
		for sink != nil {
			sample, err := sink.PullSample()
			if err != nil {
				if sink.IsEOS() {
					return
				}
				zap.L().Error("could not pull sample", zap.Error(err))
				return
			}

			// GStreamer doesn't set the buffer duration for RTP packets, so we compute the timestamp
			// based on the dts.
			rtpts := uint32(uint64(sample.Dts) / 1000 * (uint64(targetCodec.ClockRate) / 1000) / 1000)
			for _, p := range packetizer.Packetize(sample.Data, rtpts) {
				if err := tl.WriteRTP(p); err != nil {
					zap.L().Error("could not write rtp", zap.Error(err))
				}
			}
		}
	}()

	return tl, nil
}
