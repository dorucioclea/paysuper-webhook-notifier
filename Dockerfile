FROM golang:1.11.2-alpine AS builder

RUN apk add bash ca-certificates git

WORKDIR /application

ENV GO111MODULE=on
ENV MICRO_BROKER=rabbitmq
ENV MICRO_BROKER_ADDRESS=amqp://p1pay-rabbitmq
ENV CENTRIFUGO_URL=https://cf.tst.protocol.one

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -o $GOPATH/bin/payone_notifier

ENTRYPOINT $GOPATH/bin/payone_notifier