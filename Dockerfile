FROM golang:latest

WORKDIR /go/src/work

ADD . /go/src/work

RUN go mod download

CMD /bin/bash
