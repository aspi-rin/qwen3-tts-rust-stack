# Source-built Qwen3-TTS-Rust server image.
# We build qwen3_tts_server from the pinned upstream git submodule because the
# official release artifacts only package the one-shot CLI binary, not the
# streaming server.
FROM rust:1-bookworm AS builder

ARG FEATURES=vulkan

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    clang \
    cmake \
    libasound2-dev \
    libssl-dev \
    pkg-config \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /build/upstream
COPY upstream/ /build/upstream/
COPY tools/qwen3-tts-model-download/ /build/tools/qwen3-tts-model-download/

# Keep upstream source unmodified in the image build. The only dependency-level
# constraint is a stack-local Cargo.lock pin for ort rc.11; without a lock,
# Cargo may resolve to newer rc releases with incompatible SessionBuilder APIs.
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    cargo update -p ort --precise 2.0.0-rc.11 \
 && cd /build/tools/qwen3-tts-model-download \
 && cargo update -p ort --precise 2.0.0-rc.11

RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/build/upstream/target \
    cargo build --release --features "${FEATURES}" --bin qwen3_tts_server \
 && cp target/release/qwen3_tts_server /tmp/qwen3_tts_server \
 && CARGO_TARGET_DIR=/build/upstream/target cargo build --release --features "${FEATURES}" --manifest-path /build/tools/qwen3-tts-model-download/Cargo.toml \
 && cp target/release/qwen3_tts_model_download /tmp/qwen3_tts_model_download

FROM debian:bookworm-slim AS runtime-bundle

ARG QWEN3_TTS_RUNTIME_VERSION=v0.1.6
ARG QWEN3_TTS_RUNTIME_ASSET=qwen3-tts-linux-x64-vulkan.tar.gz
ARG QWEN3_TTS_RUNTIME_SHA256=867bbde32a499a6e9ff380e30ec92a3e1331d6ab9b18a6cdab4f63b4edf6b823
ARG QWEN3_TTS_RUNTIME_REPO=https://github.com/cgisky1980/Qwen3-TTS-Rust
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    tar \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /bundle
RUN set -eux; \
    url="${QWEN3_TTS_RUNTIME_REPO}/releases/download/${QWEN3_TTS_RUNTIME_VERSION}/${QWEN3_TTS_RUNTIME_ASSET}"; \
    curl -L --fail --retry 3 --retry-delay 2 -o /tmp/qwen3-tts-runtime.tar.gz "$url"; \
    echo "${QWEN3_TTS_RUNTIME_SHA256}  /tmp/qwen3-tts-runtime.tar.gz" | sha256sum -c -; \
    tar -xzf /tmp/qwen3-tts-runtime.tar.gz -C /bundle; \
    rm /tmp/qwen3-tts-runtime.tar.gz; \
    if [ -d /bundle/runtime/llama-b7885 ]; then cp -a /bundle/runtime/llama-b7885/lib*.so* /bundle/runtime/; fi

# The final runtime uses Debian trixie so mesa-vulkan-drivers is new enough for
# recent AMD GPUs. Debian bookworm's Mesa can report newer cards as
# "amdgpu: unknown", which prevents the Vulkan backend from loading.
FROM debian:trixie-slim AS runtime

ARG QWEN3_TTS_RUNTIME_BACKEND=vulkan

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    libasound2t64 \
    libgomp1 \
    libssl3t64 \
    libstdc++6 \
    mesa-vulkan-drivers \
    vulkan-tools \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /tmp/qwen3_tts_server /app/qwen3_tts_server
COPY --from=builder /tmp/qwen3_tts_model_download /app/qwen3_tts_model_download
COPY --from=builder /build/upstream/speakers /app/speakers
COPY --from=runtime-bundle /bundle/runtime /app/runtime
COPY docker/qwen3_tts_entrypoint.sh /app/qwen3_tts_entrypoint

RUN chmod +x /app/qwen3_tts_server /app/qwen3_tts_model_download /app/qwen3_tts_entrypoint \
 && mkdir -p /app/models

ENV QWEN3_TTS_BACKEND=${QWEN3_TTS_RUNTIME_BACKEND}
ENV RUST_LOG=info
ENV LD_LIBRARY_PATH=/app/runtime:/app/runtime/llama-b7885
ENV QWEN3_TTS_RUNTIME_DIR=/app/runtime
ENV ORT_DYLIB_PATH=/app/runtime/libonnxruntime.so
ENV QWEN3_TTS_DEBUG_LOGS=0

EXPOSE 3000
ENTRYPOINT ["/app/qwen3_tts_entrypoint"]
CMD ["--host", "0.0.0.0", "--model-dir", "/app/models", "--quant", "q5_k_m", "--speakers-dir", "/app/speakers", "--threads", "4"]
