FROM golang:latest

ADD . /app

WORKDIR /app

ENTRYPOINT ["/app/action.sh"]
