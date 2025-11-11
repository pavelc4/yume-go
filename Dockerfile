FROM golang:latest AS builder

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    upx-ucl git ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev") && \
    COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown") && \
    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w \
      -X main.version=$VERSION \
      -X main.commit=$COMMIT \
      -X main.buildTime=$BUILD_TIME \
      -extldflags '-static'" \
    -trimpath \
    -tags netgo \
    -o yume-go \
    cmd/bot/main.go && \
    upx --best --lzma yume-go

FROM debian:bookworm-slim

LABEL maintainer="Dimas <your-email@example.com>" \
      description="Yume-Go Telegram Waifu Gacha Bot"

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata procps && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN useradd -m -u 1000 -s /bin/bash yume

WORKDIR /home/yume

COPY --from=builder --chown=yume:yume /app/yume-go .

USER yume

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep yume-go > /dev/null || exit 1

ENTRYPOINT ["./yume-go"]
