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

# Upstream pinned commit currently needs two local build patches:
# - ort rc.12 SessionBuilder API needs mutable builders.
# - upstream leaves ort on a floating rc prerelease and enables CUDA/DirectML
#   features unconditionally; pin rc.11 and keep only dynamic CPU ONNX loading.
# - quantized downloads put qwen3_assets.gguf under the quant folder, but
#   assets are expected to remain F32 under gguf/. Keep these local until
#   upstream carries fixes.
# - upstream verifies the Windows Vulkan backend filename on every platform;
#   check the platform-correct shared object so container logs make Vulkan
#   backend failures obvious.
RUN sed -i 's/let builder = builder.with_execution_providers/let mut builder = builder.with_execution_providers/' src/models/onnx.rs \
 && sed -i 's/Ok(builder) => {/Ok(mut builder) => {/' src/models/onnx.rs \
 && sed -i 's/use ort::execution_providers::DirectMLExecutionProvider;/#[cfg(windows)]\nuse ort::execution_providers::DirectMLExecutionProvider;/' src/models/onnx.rs \
 && sed -i 's/version = "2.0.0-rc.11"/version = "=2.0.0-rc.11"/' Cargo.toml \
 && sed -i 's/features = \["ndarray", "load-dynamic", "cuda", "directml"\]/features = ["ndarray", "load-dynamic"]/' Cargo.toml \
 && sed -i '/qwen3_assets.gguf/,+1 s@format!("{}/qwen3_assets.gguf", quant_folder)@"gguf/qwen3_assets.gguf".to_owned()@g' src/download.rs \
 && sed -i 's@let runtime_dir = PathBuf::from("runtime");@let runtime_dir = PathBuf::from(std::env::var("QWEN3_TTS_RUNTIME_DIR").unwrap_or_else(|_| "/app/runtime".to_string()));@' src/models/onnx.rs \
 && sed -i '/let env_builder =/i\    std::env::set_var("ORT_DYLIB_PATH", &dll_path);\n    println!("  [ONNX] ORT_DYLIB_PATH={}", dll_path.display());' src/models/onnx.rs \
 && sed -i 's@let vulkan_path = runtime_path.join("ggml-vulkan.dll");@let vulkan_name = if cfg!(target_os = "windows") { "ggml-vulkan.dll" } else if cfg!(target_os = "macos") { "libggml-metal.dylib" } else { "libggml-vulkan.so" };\
            let vulkan_path = runtime_path.join(vulkan_name);@' src/models/llama/mod.rs \
 && sed -i '/match libloading::Library::new(&vulkan_path)/,+3 c\            match libloading::Library::new(&vulkan_path) {\
                Ok(_) => println!("  [System] Verified: {} is loadable.", vulkan_name),\
                Err(e) => eprintln!("  [System] WARNING: {} is NOT loadable: {}. This usually means missing Vulkan drivers or runtime.", vulkan_name, e),\
            }' src/models/llama/mod.rs

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
ARG ONNXRUNTIME_VERSION=1.23.2
ARG ONNXRUNTIME_SHA256=1fa4dcaef22f6f7d5cd81b28c2800414350c10116f5fdd46a2160082551c5f9b

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
    if [ -d /bundle/runtime/llama-b7885 ]; then cp -a /bundle/runtime/llama-b7885/lib*.so* /bundle/runtime/; fi; \
    ort_url="https://github.com/microsoft/onnxruntime/releases/download/v${ONNXRUNTIME_VERSION}/onnxruntime-linux-x64-${ONNXRUNTIME_VERSION}.tgz"; \
    curl -L --fail --retry 3 --retry-delay 2 -o /tmp/onnxruntime-linux-x64.tgz "$ort_url"; \
    echo "${ONNXRUNTIME_SHA256}  /tmp/onnxruntime-linux-x64.tgz" | sha256sum -c -; \
    tar -xzf /tmp/onnxruntime-linux-x64.tgz -C /tmp; \
    cp -a /tmp/onnxruntime-linux-x64-${ONNXRUNTIME_VERSION}/lib/libonnxruntime*.so* /bundle/runtime/; \
    rm -rf /tmp/onnxruntime-linux-x64.tgz /tmp/onnxruntime-linux-x64-${ONNXRUNTIME_VERSION}

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

RUN chmod +x /app/qwen3_tts_server /app/qwen3_tts_model_download \
 && mkdir -p /app/models

ENV QWEN3_TTS_BACKEND=${QWEN3_TTS_RUNTIME_BACKEND}
ENV RUST_LOG=info
ENV LD_LIBRARY_PATH=/app/runtime:/app/runtime/llama-b7885
ENV QWEN3_TTS_RUNTIME_DIR=/app/runtime
ENV ORT_DYLIB_PATH=/app/runtime/libonnxruntime.so

EXPOSE 3000
ENTRYPOINT ["/app/qwen3_tts_server"]
CMD ["--host", "0.0.0.0", "--model-dir", "/app/models", "--quant", "q5_k_m", "--speakers-dir", "/app/speakers", "--threads", "4"]
