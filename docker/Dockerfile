# Release-based runtime image for Qwen3-TTS-Rust.
# This is the default Docker path for validation: use the upstream binary release
# instead of compiling Rust during image build.
FROM debian:bookworm-slim AS runtime

ARG QWEN3_TTS_RUST_VERSION=v0.1.6
ARG QWEN3_TTS_RUST_ASSET=qwen3-tts-linux-x64-vulkan.tar.gz
ARG QWEN3_TTS_RUST_SHA256=867bbde32a499a6e9ff380e30ec92a3e1331d6ab9b18a6cdab4f63b4edf6b823
ARG QWEN3_TTS_RUST_REPO=https://github.com/cgisky1980/Qwen3-TTS-Rust

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    tar \
    libasound2 \
    libstdc++6 \
    libgomp1 \
    libssl3 \
    mesa-vulkan-drivers \
    vulkan-tools \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app

RUN set -eux; \
    url="${QWEN3_TTS_RUST_REPO}/releases/download/${QWEN3_TTS_RUST_VERSION}/${QWEN3_TTS_RUST_ASSET}"; \
    curl -L --fail --retry 3 --retry-delay 2 -o /tmp/qwen3-tts-release.tar.gz "$url"; \
    echo "${QWEN3_TTS_RUST_SHA256}  /tmp/qwen3-tts-release.tar.gz" | sha256sum -c -; \
    tar -xzf /tmp/qwen3-tts-release.tar.gz -C /app; \
    rm /tmp/qwen3-tts-release.tar.gz; \
    if [ -d /app/runtime/llama-b7885 ]; then cp -a /app/runtime/llama-b7885/lib*.so* /app/runtime/; fi; \
    chmod +x /app/qwen3_tts; \
    mkdir -p /app/models /app/outputs

COPY scripts /app/scripts

ENV RUST_LOG=info
ENV MODEL_DIR=/app/models
ENV SPEAKERS_DIR=/app/speakers
ENV LD_LIBRARY_PATH=/app/runtime:/app/runtime/llama-b7885

ENTRYPOINT ["/app/qwen3_tts"]
CMD ["--help"]
