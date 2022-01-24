# syntax=docker/dockerfile:1

FROM golang:1.17-alpine

RUN apk add gstreamer gstreamer-dev gstreamer-tools gst-libav musl-dev gcc gst-plugins-base gst-plugins-base-dev gst-plugins-good gst-plugins-bad gst-plugins-ugly

WORKDIR /app

COPY go.* ./

RUN go mod download

COPY . ./

RUN go build -v -o /transcoder cmd/main.go

ENV APP_ENV=production
ENV GST_DEBUG=3

RUN go test ./...

CMD [ "/transcoder" ]