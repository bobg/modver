FROM golang:latest

ADD . /app
ADD $GOROOT $GOROOT

WORKDIR /app

RUN go build ./cmd/modver-action

ENTRYPOINT ["/app/modver-action"]
