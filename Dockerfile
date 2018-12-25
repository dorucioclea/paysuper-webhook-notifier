FROM golang:1.11.2-alpine AS builder

RUN apk add bash ca-certificates git

WORKDIR /application

ENV GO111MODULE=on
ENV MICRO_REGISTRY_ADDRESS=p1pay-consul
ENV MICRO_BROKER=rabbitmq
ENV MICRO_BROKER_ADDRESS=amqp://p1pay-rabbitmq
ENV CENTRIFUGO_URL=http://localhost:8000
ENV CENTRIFUGO_KEY=H383Js2!dhssjks#9!


COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -o $GOPATH/bin/payone_notifier

ENTRYPOINT $GOPATH/bin/payone_notifier