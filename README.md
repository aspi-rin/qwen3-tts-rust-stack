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
make up        # builds the local image, prepares models, then starts qwen3_tts_server
make down      # stop the service
```

Default `BACKEND=vulkan` passes `/dev/dri` into the container. Available backends:

```bash
BACKEND=vulkan make up   # Vulkan path; intended for AMD/Intel/NVIDIA Vulkan-capable hosts
BACKEND=cpu make up      # no GPU passthrough; uses llama.cpp/ggml CPU fallback
```

The first startup runs the `qwen3-tts-model-download` model download service to download Qwen3-TTS model files into `./models`, so it can take time depending on network speed.

## Configuration

`.env` intentionally keeps only service/runtime knobs:

| Variable | Default | Purpose |
| --- | --- | --- |
| `QWEN3_TTS_BACKEND` | `vulkan` | `vulkan` passes `/dev/dri`; `cpu` does not. |
| `QWEN3_TTS_QUANT` | `q5_k_m` | Qwen3-TTS quantization helper: `none`, `q5_k_m`, or `q8_0`. |

Container-internal paths are fixed: models at `/app/models`, speakers at `/app/speakers`, runtime libs at `/app/runtime`.

## Development

The stack has a small local Rust wrapper for model preparation under `tools/qwen3-tts-model-download`.

```bash
make fmt-check
make test
```

`make test` runs wrapper unit tests without compiling the upstream TTS dependency; the Docker smoke job still validates the real image binary.

## Build strategy

We do **not** use the official Qwen3-TTS-Rust release binary as the application because upstream `v0.1.6` only packages the one-shot `qwen3_tts` CLI, not `qwen3_tts_server`.

Instead, `docker/Dockerfile`:

1. builds `qwen3_tts_server` from the pinned upstream git submodule at `./upstream`, documented in [`upstream.lock`](./upstream.lock);
2. builds the local `qwen3_tts_model_download` wrapper, which calls upstream model preparation logic and normalizes the `qwen3_assets.gguf` layout without patching upstream source;
3. pins the upstream build dependency resolver to `ort` rc.11 via a build-local `Cargo.lock`, leaving upstream source files unmodified;
4. vendors the pinned Linux Vulkan runtime bundle documented in [`runtime.lock`](./runtime.lock) for llama.cpp/ONNX shared libraries;
5. runs `qwen3-tts-model-download` before starting the long-running `qwen3-tts-server` HTTP/WebSocket service.

CPU mode currently reuses the same Linux Vulkan runtime bundle without `/dev/dri` passthrough and relies on llama.cpp/ggml CPU fallback.

## Basic checks

```bash
curl -fsS http://127.0.0.1:9746/health
curl -fsS http://127.0.0.1:9746/api/speakers
```

For Vulkan deployments, confirm the container loaded the Linux Vulkan backend
and that llama.cpp assigned layers to a GPU device:

```bash
docker logs qwen3-tts-server 2>&1 | grep -Ei 'vulkan|backend|offload|assigned to device'
```

The runtime image uses Debian trixie for newer Mesa Vulkan drivers. Debian
bookworm's Mesa can fail to recognize newer AMD GPUs and silently fall back to
CPU-heavy execution.

For streaming, use the upstream WebSocket endpoint:

```text
ws://127.0.0.1:9746/api/tts/stream
```

The repo includes a stdlib-only WebSocket smoke client that records streamed
f32le PCM, writes a WAV copy, and reports TTFB/chunk/RTF metrics:

```bash
python3 tools/stream-smoke/qwen3_tts_stream_smoke.py \
  --text '你好，这是本地流式语音合成测试。'
aplay /tmp/qwen3-tts-stream.wav
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
