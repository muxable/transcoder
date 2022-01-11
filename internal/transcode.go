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
	case "video/h265":
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

func PipelineString(codec webrtc.RTPCodecParameters) (string, error) {
	appsrc := fmt.Sprintf("appsrc format=time do-timestamp=true name=source ! %s ! queue", ToRTPCaps(codec))
	// appsink outputs rtp's because the pion h264 payloader is more reliable and easier to debug
	// than the gstreamer payloader.
	appsink := "queue ! appsink name=rtpsink"

	switch codec.MimeType {
	// video codecs
	case "video/h265":
		return appsrc + ` ! rtph265depay ! decodebin ! queue ! videoconvert ! x264enc speed-preset=ultrafast tune=zerolatency key-int-max=20 ! ` + appsink, nil
	case webrtc.MimeTypeH264:
		return appsrc + ` ! rtph264depay ! decodebin ! queue ! videoconvert ! x264enc speed-preset=ultrafast tune=zerolatency key-int-max=20 ! ` + appsink, nil
	case webrtc.MimeTypeVP8:
		return appsrc + ` ! rtpvp8depay ! decodebin ! queue ! videoconvert ! x264enc speed-preset=ultrafast tune=zerolatency key-int-max=20 ! ` + appsink, nil
	case webrtc.MimeTypeVP9:
		return appsrc + ` ! rtpvp9depay ! decodebin ! queue ! videoconvert ! x264enc speed-preset=ultrafast tune=zerolatency key-int-max=20 ! ` + appsink, nil
	// audio codecs
	case webrtc.MimeTypeOpus:
		return appsrc + ` ! rtpopusdepay ! decodebin ! queue ! audioconvert ! opusenc ! ` + appsink, nil
	case "audio/aac":
		return appsrc + ` ! rtpmp4adepay ! decodebin ! queue ! audioconvert ! opusenc ! ` + appsink, nil
	case webrtc.MimeTypeG722:
		return appsrc + ` ! rtpg722depay ! decodebin ! queue ! audioconvert ! audioresample ! opusenc ! ` + appsink, nil
	}
	return "", fmt.Errorf("unsupported codec %s", codec.MimeType)
}

func TargetCodec(codec webrtc.RTPCodecCapability) (*webrtc.RTPCodecCapability, rtp.Payloader, error) {
	switch codec.MimeType {
	case "video/h265", webrtc.MimeTypeH264, webrtc.MimeTypeVP8, webrtc.MimeTypeVP9:
		return &webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000}, &codecs.H264Payloader{}, nil
	case webrtc.MimeTypeOpus, "audio/aac", webrtc.MimeTypeG722:
		return &webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000}, &codecs.OpusPayloader{}, nil
	}
	return nil, nil, fmt.Errorf("unsupported codec %s", codec.MimeType)
}

func NewPipeline(codec webrtc.RTPCodecParameters) (*gst.Bin, error) {
	pipelineStr, err := PipelineString(codec)
	if err != nil {
		return nil, err
	}
	return gst.ParseBinFromDescription(pipelineStr, false)
}

func TranscodePeerConnection(pc *webrtc.PeerConnection) error {
	pipeline, err := gst.PipelineNew("transcode")
	if err != nil {
		return err
	}

	pipeline.SetState(gst.StatePlaying)

	pc.OnTrack(func(tr *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		zap.L().Debug("OnTrack", zap.String("kind", tr.Kind().String()), zap.Uint8("payloadType",
			uint8(tr.Codec().PayloadType)))

		targetCodec, payloader, err := TargetCodec(tr.Codec().RTPCodecCapability)
		if err != nil {
			zap.L().Error("could not determine target codec", zap.Error(err))
			return
		}

		packetizer := NewTSPacketizer(1200, payloader, rtp.NewRandomSequencer())

		tl, err := webrtc.NewTrackLocalStaticRTP(*targetCodec, tr.ID(), fmt.Sprintf("%s-transcode", tr.StreamID()))
		if err != nil {
			zap.L().Error("could not create track local", zap.Error(err))
			return
		}

		if _, err := pc.AddTrack(tl); err != nil {
			zap.L().Error("could not add track", zap.Error(err))
			return
		}

		bin, err := NewPipeline(tr.Codec())
		if err != nil {
			zap.L().Error("could not create pipeline", zap.Error(err))
			return
		}

		pipeline.Add(&bin.Element)

		bin.SetState(gst.StatePlaying)

		source := bin.GetByName("source")
		rtpsink := bin.GetByName("rtpsink")

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
			for rtpsink != nil {
				sample, err := rtpsink.PullSample()
				if err != nil {
					if rtpsink.IsEOS() {
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
	})

	return nil
}
