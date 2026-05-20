# qwen3-tts-rust-stack

Self-hosted Text-to-Speech service stack for [`Qwen3-TTS-Rust`](https://github.com/cgisky1980/Qwen3-TTS-Rust), Docker Compose, and streaming-oriented local deployment.

The public host port is served by the stack adapter. It supports OpenAI-compatible `/v1/*` endpoints and passes Qwen-native `/api/*` endpoints through to upstream `qwen3_tts_server`:

```text
GET  /health
GET  /v1/models
POST /v1/audio/speech  # OpenAI-compatible TTS
GET  /api/speakers     # Qwen passthrough
POST /api/tts          # Qwen passthrough
GET  /api/tts/stream   # Qwen WebSocket passthrough
```

## Quick start

Requirements: Linux, Docker with Compose plugin, git submodules initialized.

```bash
git submodule update --init --recursive
cp .env.example .env
make up        # builds images, prepares models, then starts upstream server + adapter
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
| `QWEN3_TTS_DEBUG_LOGS` | `0` | Set to `1`/`true`/`on` to keep verbose upstream debug logs; default filters noisy debug lines. |
| `QWEN3_TTS_BIND_HOST` | `127.0.0.1` | Host interface for the public adapter port; set `0.0.0.0` to expose it to reverse proxies or the LAN. |

Container-internal paths are fixed: models at `/app/models`, speakers at `/app/speakers`, runtime libs at `/app/runtime`.

To expose the adapter beyond localhost, set `QWEN3_TTS_BIND_HOST=0.0.0.0` in `.env` and recreate the stack. The public service remains on port `9746`.

`QWEN3_TTS_DEBUG_LOGS=0` keeps upstream source unchanged and filters known noisy `Debug:` / `[Debug]` lines at container entrypoint level. To temporarily inspect upstream verbose logs:

```bash
QWEN3_TTS_DEBUG_LOGS=1 make up
# or edit .env and set QWEN3_TTS_DEBUG_LOGS=1, then recreate the service
```

## Development

The stack has a small local Rust wrapper for model preparation under `tools/qwen3-tts-model-download` and a Go OpenAI-compatible adapter under `services/openai-api`.

```bash
make fmt-check
make test
```

`make test` runs wrapper unit tests without compiling the upstream TTS dependency; the Docker smoke job still validates the real image binary.

## Build strategy

We do **not** use the official Qwen3-TTS-Rust release binary as the application because upstream `v0.1.6` only packages the one-shot `qwen3_tts` CLI, not `qwen3_tts_server`.

Instead, `docker/Dockerfile`:

1. builds `qwen3_tts_server` from the pinned upstream git submodule at `./upstream`, documented in [`upstream.lock`](./upstream.lock), with stack-local compatibility patches applied only to the Docker build copy;
2. builds the local `qwen3_tts_model_download` wrapper, which calls upstream model preparation logic and normalizes the `qwen3_assets.gguf` layout without patching upstream source;
3. pins the upstream build dependency resolver to `ort` rc.11 via a build-local `Cargo.lock`, leaving the checked-out submodule files unmodified;
4. vendors the pinned Linux Vulkan runtime bundle documented in [`runtime.lock`](./runtime.lock) for llama.cpp/ONNX shared libraries;
5. runs `qwen3-tts-model-download` before starting the long-running `qwen3-tts-server` HTTP/WebSocket service.

CPU mode currently reuses the same Linux Vulkan runtime bundle without `/dev/dri` passthrough and relies on llama.cpp/ggml CPU fallback.

## Basic checks

```bash
curl -fsS http://127.0.0.1:9746/health
curl -fsS http://127.0.0.1:9746/v1/models
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

For Qwen-native streaming, use the passthrough WebSocket endpoint:

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

OpenAI-compatible examples:

```bash
curl -N http://127.0.0.1:9746/v1/audio/speech \
  -H 'Content-Type: application/json' \
  -d '{"model":"qwen3-tts","input":"你好，这是 OpenAI 兼容接口测试。","voice":"vivian","response_format":"pcm"}' \
  -o /tmp/qwen3-openai.s16le.pcm

aplay -f S16_LE -r 24000 -c 1 /tmp/qwen3-openai.s16le.pcm
```

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
