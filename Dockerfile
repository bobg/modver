FROM golang:latest

ADD . /app

WORKDIR /app

RUN go build ./cmd/modver-action

ENTRYPOINT ["/app/modver-action"]
