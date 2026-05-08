FROM golang:1.22-bookworm AS builder

WORKDIR /src
COPY services/openai-api/go.mod services/openai-api/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY services/openai-api/ ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /tmp/qwen3_openai_api .

FROM debian:trixie-slim AS runtime

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
 && rm -rf /var/lib/apt/lists/*

COPY --from=builder /tmp/qwen3_openai_api /app/qwen3_openai_api
RUN chmod +x /app/qwen3_openai_api

EXPOSE 8080
ENTRYPOINT ["/app/qwen3_openai_api"]
