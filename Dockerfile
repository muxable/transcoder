# syntax=docker/dockerfile:1

FROM golang:1.17-alpine

RUN apk add gstreamer gstreamer-dev gstreamer-tools gst-libav musl-dev gcc gst-plugins-base gst-plugins-base-dev gst-plugins-good gst-plugins-bad

WORKDIR /app

COPY go.* ./

RUN go mod download

COPY . ./

RUN go build -v -o /transcoder cmd/main.go

CMD [ "/transcoder" ]