# OpenAI-compatible API Adapter Plan

## Goal

Add an OpenAI-style TTS API without modifying the pinned `Qwen3-TTS-Rust`
upstream source. The adapter is a stack-layer service that translates OpenAI-like
HTTP requests into the current upstream HTTP/WebSocket API.

```text
Client
  -> qwen3-tts-openai-api (Go adapter, OpenAI /v1 plus Qwen /api passthrough)
  -> qwen3-tts-server (upstream Rust server, internal only)
  -> Qwen3-TTS-Rust engine
```

This keeps the upstream submodule pure while allowing the deployment/API layer
to evolve independently.

## Why Go

Use Go for the adapter instead of adding more Rust or introducing Python:

- Single static-ish binary and simple container image.
- Low maintenance cost compared with Rust async glue code.
- Better production ergonomics than Python for long-running streaming IO.
- Adapter workload is mostly JSON, HTTP, WebSocket, and small PCM conversion;
  the expensive model inference remains in the upstream Rust server.

## Service Layout

Proposed repo layout:

```text
services/
└── openai-api/
    ├── go.mod
    ├── main.go
    └── internal/
        ├── audio/      # f32le -> s16le, optional WAV helpers
        ├── config/     # env/config parsing
        └── upstream/   # upstream HTTP + WebSocket client
```

Proposed compose service:

```text
qwen3-tts-openai-api
```

Ports:

```text
127.0.0.1:9746 -> qwen3-tts-openai-api
```

The upstream Rust server is internal to Compose. The adapter serves
OpenAI-compatible `/v1/*` routes and reverse-proxies Qwen-native `/api/*` routes
to upstream, so both API families share one public host port.

## Pinned schema

The OpenAI-compatible schemas are pinned under [`schemas/`](../schemas/) and summarized in [`openai-tts-contract.md`](./openai-tts-contract.md). Implementation should validate requests against these schemas before applying runtime support checks.

## Endpoints

### `GET /health`

Returns adapter health. It should also be able to check upstream health via
`GET http://qwen3-tts-server:3000/health`.

### `GET /v1/models`

Minimal OpenAI-style model list:

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4o-mini-tts",
      "object": "model",
      "owned_by": "openai-compatible-local"
    }
  ]
}
```

### `POST /v1/audio/speech`

Request shape:

```json
{
  "model": "gpt-4o-mini-tts",
  "input": "你好，我是本地语音合成服务。",
  "voice": {"id": "vivian"},
  "response_format": "pcm",
  "speed": 1.0,
  "instructions": "自然、清晰"
}
```

First version semantics:

| Field | Behavior |
| --- | --- |
| `model` | Required; accepts the pinned OpenAI model enum and maps to local Qwen3-TTS. |
| `input` | Required; maps to upstream `text`. |
| `voice` | Required; OpenAI built-in voices map to the configured default speaker; `{"id": "vivian"}` maps directly to a Qwen speaker. |
| `response_format` | `pcm` or `wav` supported initially; other pinned OpenAI formats return `unsupported_feature_error`. |
| `speed` | Accepted for client compatibility; ignored initially. |
| `instructions` | Optional; forwarded to upstream `instruction`. |

## Response Formats

### `response_format=pcm`

Use the upstream WebSocket streaming endpoint and return true HTTP chunked PCM.

Upstream:

```text
ws://qwen3-tts-server:3000/api/tts/stream
```

Adapter response:

```http
Content-Type: audio/pcm
Transfer-Encoding: chunked
X-Audio-Sample-Rate: 24000
X-Audio-Channels: 1
X-Audio-Format: s16le
```

Conversion:

```text
upstream f32le mono 24kHz -> adapter s16le mono 24kHz
```

Rationale: s16le is smaller and easier for typical playback pipelines than raw
float32 PCM.

### `response_format=wav`

Use upstream non-streaming HTTP endpoint:

```text
POST http://qwen3-tts-server:3000/api/tts
```

Upstream returns base64-encoded WAV in JSON. Adapter decodes it and returns:

```http
Content-Type: audio/wav
```

First version should keep WAV non-streaming to avoid ambiguous WAV header length
handling in chunked responses.

## Upstream Mapping

| Adapter | Upstream |
| --- | --- |
| `/v1/audio/speech`, `response_format=wav` | `POST /api/tts` |
| `/v1/audio/speech`, `response_format=pcm` | `GET /api/tts/stream` WebSocket |
| `input` | `text` |
| `voice` | `speaker` |
| `instructions` | `instruction` |

## Implementation Phases

1. **Skeleton**
   - Go service with `/health` and `/v1/models`.
   - Dockerfile and compose service.
   - Adapter port `9746`.

2. **WAV path**
   - Implement non-streaming `/v1/audio/speech` with `response_format=wav`.
   - Forward to upstream `/api/tts`.
   - Decode base64 WAV and return `audio/wav`.

3. **Streaming PCM path**
   - Implement `response_format=pcm`.
   - Bridge upstream WebSocket binary chunks to HTTP chunked response.
   - Convert f32le samples to s16le on the fly.
   - Flush after each chunk to preserve streaming behavior.

4. **Qwen upstream passthrough**
   - Reverse-proxy `/api/*` to the upstream Rust server so Qwen-native clients
     can use the adapter port directly.

5. **Smoke tests**
   - Add curl examples for WAV and PCM.
   - Add a small smoke script or reuse existing streaming smoke logic for the
     adapter endpoint.

## Non-goals for First Version

- Authentication.
- Rate limiting.
- mp3/opus/aac encoding.
- SSE audio events.
- Multiple model IDs.
- Implementing `speed` behavior.
- Modifying upstream `Qwen3-TTS-Rust` source.

## Example Usage

WAV:

```bash
curl -sS http://127.0.0.1:9746/v1/audio/speech \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "gpt-4o-mini-tts",
    "input": "你好，这是 OpenAI 风格接口测试。",
    "voice": {"id": "vivian"},
    "response_format": "wav"
  }' \
  -o /tmp/qwen3-openai.wav
```

PCM streaming:

```bash
curl -N http://127.0.0.1:9746/v1/audio/speech \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "gpt-4o-mini-tts",
    "input": "你好，这是流式 PCM 测试。",
    "voice": {"id": "vivian"},
    "response_format": "pcm"
  }' \
  -o /tmp/qwen3-openai.s16le.pcm

aplay -f S16_LE -r 24000 -c 1 /tmp/qwen3-openai.s16le.pcm
```
