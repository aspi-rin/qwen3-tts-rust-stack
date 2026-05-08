# Qwen3 upstream API contract schemas

This stack keeps the upstream `Qwen3-TTS-Rust` server contract separate from
the future OpenAI-compatible adapter contract.

The upstream schemas are pinned under:

```text
schemas/qwen3/
```

## Schema files

| File | Purpose |
| --- | --- |
| [`upstream-tts-request.schema.json`](../schemas/qwen3/upstream-tts-request.schema.json) | JSON request body accepted by upstream `POST /api/tts` and the first WebSocket message for `GET /api/tts/stream`. |
| [`upstream-tts-response.schema.json`](../schemas/qwen3/upstream-tts-response.schema.json) | JSON response returned by upstream `POST /api/tts`. |
| [`upstream-tts-stream-message.schema.json`](../schemas/qwen3/upstream-tts-stream-message.schema.json) | Client text messages for upstream `GET /api/tts/stream` WebSocket. |
| [`upstream-tts-stream-event.schema.json`](../schemas/qwen3/upstream-tts-stream-event.schema.json) | Server event metadata for upstream stream text events and binary PCM chunks. |
| [`upstream-speakers-response.schema.json`](../schemas/qwen3/upstream-speakers-response.schema.json) | JSON response returned by upstream `GET /api/speakers`. |
| [`model-download-request.schema.json`](../schemas/qwen3/model-download-request.schema.json) | Stack model-download helper argument contract. |

## Endpoints covered

```text
GET  /health
GET  /api/speakers
POST /api/tts
GET  /api/tts/stream
```

`/health` returns plain text `OK` and does not need a JSON schema.

## Stream payload note

Upstream `GET /api/tts/stream` is a WebSocket endpoint. Client messages are JSON
text messages plus the string `end`. Server audio chunks are binary `f32le`,
24kHz, mono PCM samples. Since JSON Schema cannot validate raw WebSocket binary
frames, `upstream-tts-stream-event.schema.json` pins the metadata contract that
our smoke tests and future adapter should assume.

## Adapter boundary

The OpenAI-compatible adapter should translate from the schemas in
[`schemas/openai/`](../schemas/openai/) to these upstream Qwen3 schemas. Keep
Qwen3-specific fields, speaker names, and binary `f32le` stream details out of
the strict OpenAI request schema unless we explicitly design an extension.
