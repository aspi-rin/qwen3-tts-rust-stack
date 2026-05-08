# qwen3-tts-rust-stack

Self-hosted Text-to-Speech service stack for [`Qwen3-TTS-Rust`](https://github.com/cgisky1980/Qwen3-TTS-Rust), Docker Compose, and streaming-oriented local deployment.

Current service endpoints come from upstream `qwen3_tts_server`:

```text
GET  /health
GET  /api/speakers
POST /api/tts
GET  /api/tts/stream   # WebSocket streaming TTS
```

## Quick start

Requirements: Linux, Docker with Compose plugin, git submodules initialized.

```bash
git submodule update --init --recursive
cp .env.example .env
make up        # builds the local server image, then starts qwen3_tts_server
make down      # stop the service
```

Default `BACKEND=vulkan` passes `/dev/dri` into the container. Available backends:

```bash
BACKEND=vulkan make up   # Vulkan path; intended for AMD/Intel/NVIDIA Vulkan-capable hosts
BACKEND=cpu make up      # no GPU passthrough; uses llama.cpp/ggml CPU fallback
```

The first startup downloads Qwen3-TTS model files into `./models`, so it can take time depending on network speed.

## Configuration

`.env` intentionally keeps only service/runtime knobs:

| Variable | Default | Purpose |
| --- | --- | --- |
| `QWEN3_TTS_BACKEND` | `vulkan` | `vulkan` passes `/dev/dri`; `cpu` does not. |
| `QWEN3_TTS_HOST` | `127.0.0.1` | Host interface exposed by Compose. |
| `QWEN3_TTS_PORT` | `3000` | Host port mapped to the service. |
| `QWEN3_TTS_QUANT` | `q5_k_m` | Qwen3-TTS quantization helper: `none`, `q5_k_m`, or `q8_0`. |

Container-internal paths are fixed: models at `/app/models`, speakers at `/app/speakers`, runtime libs at `/app/runtime`.

## Build strategy

We do **not** use the official Qwen3-TTS-Rust release binary as the application because upstream `v0.1.6` only packages the one-shot `qwen3_tts` CLI, not `qwen3_tts_server`.

Instead, `docker/Dockerfile`:

1. builds `qwen3_tts_server` from the pinned upstream git submodule at `./upstream`, documented in [`upstream.lock`](./upstream.lock);
2. vendors the pinned Linux Vulkan runtime bundle documented in [`runtime.lock`](./runtime.lock) for llama.cpp/ONNX shared libraries;
3. starts a long-running HTTP/WebSocket service with Docker Compose.

CPU mode currently reuses the same Linux Vulkan runtime bundle without `/dev/dri` passthrough and relies on llama.cpp/ggml CPU fallback.

## Basic checks

```bash
curl -fsS http://127.0.0.1:3000/health
curl -fsS http://127.0.0.1:3000/api/speakers
```

For streaming, use the upstream WebSocket endpoint:

```text
ws://127.0.0.1:3000/api/tts/stream
```

The initial API is upstream-compatible. A later stack layer can add an OpenAI-style endpoint:

```http
POST /v1/audio/speech
```

with raw PCM/WAV streaming semantics.

## Validation goals

Initial pass:

- Service starts reliably.
- WebSocket streaming path works.
- Chinese output has no obvious dropped words, repetition, or clipping.
- RTF `< 1.0` on target hardware.

Ideal:

- RTF `< 0.5`.
- Low TTFB.
- Streamed PCM chunks suitable for realtime playback.
