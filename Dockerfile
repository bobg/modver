FROM golang:latest

RUN env

VOLUME ["${GOROOT}"]

ADD . /app

WORKDIR /app

RUN go build ./cmd/modver-action

ENTRYPOINT ["/app/modver-action"]
