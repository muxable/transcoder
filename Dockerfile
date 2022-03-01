# syntax=docker/dockerfile:1

FROM golang:1.17-alpine

RUN apk add ffmpeg

WORKDIR /app

COPY go.* ./

RUN go mod download

COPY . ./

RUN go build -v -o /transcoder cmd/main.go

ENV APP_ENV=production

EXPOSE 5000-5200/udp
EXPOSE 50051/tcp

CMD [ "/transcoder" ]
