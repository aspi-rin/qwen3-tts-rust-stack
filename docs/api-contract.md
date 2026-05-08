# HTTP API 草案

The stack adapter exposes OpenAI-compatible `/v1/*` routes and bypasses Qwen-native `/api/*` routes to the upstream Rust server. `/api/tts`, `/api/tts/stream` (WebSocket), and `/api/speakers` remain available on the public adapter port for compatibility.


目标：对齐 OpenAI TTS 风格，同时保留本地 Qwen API 和流式 PCM 能力。

## 非流式

`POST /v1/audio/speech`

请求：

```json
{
  "model": "qwen3-tts",
  "input": "你好，我是本地语音合成服务。",
  "voice": "vivian",
  "response_format": "pcm",
  "instructions": "自然、清晰"
}
```

响应格式：

- `response_format=pcm`: `audio/pcm`，24kHz mono s16le HTTP chunked bytes。
- `response_format=wav`: `audio/wav`。

## 流式

优先方案：HTTP chunked response，直接返回 PCM chunks。

```http
POST /v1/audio/speech
Content-Type: application/json

{"response_format":"pcm","stream_format":"audio", ...}
```

`stream_format=sse` 和 mp3/opus/aac/flac 编码暂不实现，先返回 OpenAI-style unsupported error。

## 映射关系

| OpenAI 风格字段 | Qwen3-TTS-Rust 字段 |
| --- | --- |
| `input` | CLI/API `text` |
| `voice` | Qwen speaker；也支持 `{"id":"vivian"}` |
| `response_format` | pcm/wav 输出选择 |
| `model` | 本地 `qwen3-tts` |
| `instructions` | `instruction` |
## Adapter implementation plan

The current plan is documented in [`openai-adapter.md`](./openai-adapter.md).
The compatibility layer should be implemented as a separate Go service in the
stack, not as upstream `Qwen3-TTS-Rust` source changes.

## Pinned schemas

The strict OpenAI-compatible request schema is pinned in [`../schemas/openai/openai-audio-speech-request.schema.json`](../schemas/openai/openai-audio-speech-request.schema.json) and documented in [`openai-speech-schema.md`](./openai-speech-schema.md).

The current upstream Qwen3 server contract is pinned under [`../schemas/qwen3/`](../schemas/qwen3/) and documented in [`qwen3-upstream-contract.md`](./qwen3-upstream-contract.md).
