FROM golang:latest

VOLUME $GOROOT

ADD . /app

WORKDIR /app

RUN go build ./cmd/modver-action

ENTRYPOINT ["/app/modver-action"]
