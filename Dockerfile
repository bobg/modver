FROM golang:latest

ADD . /app

WORKDIR /app

RUN go build ./cmd/modver-action

ENV GOROOT $GOPATH

RUN env

ENTRYPOINT ["/app/modver-action"]
