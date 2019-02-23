FROM golang:1.11-alpine AS builder

RUN apk add bash ca-certificates git

WORKDIR /application

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -o app .

FROM alpine:3.9
WORKDIR /application/
COPY --from=builder /application/app .

ENTRYPOINT ["./app"]