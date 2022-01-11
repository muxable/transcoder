# syntax=docker/dockerfile:1

FROM golang:1.17-alpine

RUN apk add gstreamer gstreamer-dev gstreamer-tools gst-libav musl-dev gcc gst-plugins-base gst-plugins-base-dev gst-plugins-good gst-plugins-bad gst-plugins-ugly

# verify encoder/decoder elements
RUN GST_DEBUG=3 gst-launch-1.0 videotestsrc num-buffers=1 ! x264enc ! decodebin ! fakesink
RUN GST_DEBUG=3 gst-launch-1.0 videotestsrc num-buffers=1 ! x265enc ! decodebin ! fakesink
RUN GST_DEBUG=3 gst-launch-1.0 videotestsrc num-buffers=1 ! vp8enc ! decodebin ! fakesink
RUN GST_DEBUG=3 gst-launch-1.0 videotestsrc num-buffers=1 ! vp9enc ! decodebin ! fakesink

RUN GST_DEBUG=3 gst-launch-1.0 audiotestsrc num-buffers=1 ! opusenc ! decodebin ! fakesink
RUN GST_DEBUG=3 gst-launch-1.0 audiotestsrc num-buffers=1 ! avenc_aac ! decodebin ! fakesink
RUN GST_DEBUG=3 gst-launch-1.0 audiotestsrc num-buffers=1 ! avenc_g722 ! decodebin ! fakesink

WORKDIR /app

COPY go.* ./

RUN go mod download

COPY . ./

RUN go build -v -o /transcoder cmd/main.go

CMD [ "/transcoder" ]