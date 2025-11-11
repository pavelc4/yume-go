FROM golang:latest AS builder

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    upx-ucl git ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -trimpath -o yume-go cmd/bot/main.go && \
    upx --best --lzma yume-go

FROM debian:bookworm-slim

LABEL maintainer="Dimas <your-email@example.com>" \
    description="Yume-Go Telegram Waifu Gacha Bot"

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates procps && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN useradd -m -u 1000 -s /bin/bash yume

WORKDIR /home/yume

COPY --from=builder --chown=yume:yume /app/yume-go .

USER yume

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep yume-go > /dev/null || exit 1

ENTRYPOINT ["./yume-go"]
