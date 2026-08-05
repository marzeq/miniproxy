# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY config ./config
COPY proxy ./proxy

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/miniproxy \
    ./cmd/miniproxy

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/miniproxy /usr/local/bin/miniproxy

EXPOSE 80 443

ENTRYPOINT ["miniproxy"]
