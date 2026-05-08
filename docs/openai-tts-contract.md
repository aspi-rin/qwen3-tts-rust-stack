# OpenAI-compatible TTS contract schemas

The `schemas/openai/` directory pins the OpenAI-compatible TTS contract that the stack adapter
will implement. The schemas intentionally cover both OpenAI's request shape and
the local adapter response/error behavior needed for stable clients.

## Schema files

OpenAI-compatible schemas live under `schemas/openai/`. Qwen3 upstream schemas
are documented separately in [`qwen3-upstream-contract.md`](./qwen3-upstream-contract.md)
and live under `schemas/qwen3/`.


| File | Purpose |
| --- | --- |
| [`openai-audio-speech-request.schema.json`](../schemas/openai/openai-audio-speech-request.schema.json) | JSON request body for `POST /v1/audio/speech`. |
| [`openai-audio-speech-response.schema.json`](../schemas/openai/openai-audio-speech-response.schema.json) | Successful binary audio/SSE response metadata contract. |
| [`openai-audio-speech-sse-event.schema.json`](../schemas/openai/openai-audio-speech-sse-event.schema.json) | SSE event payloads for `stream_format=sse`. |
| [`openai-error.schema.json`](../schemas/openai/openai-error.schema.json) | OpenAI-style JSON error envelope. |
| [`openai-model-object.schema.json`](../schemas/openai/openai-model-object.schema.json) | Model object returned by `/v1/models`. |
| [`openai-model-list-response.schema.json`](../schemas/openai/openai-model-list-response.schema.json) | `/v1/models` response. |

## Endpoints covered

```text
GET  /v1/models
POST /v1/audio/speech
```

Auxiliary endpoint:

```text
GET /health
```

`/health` is a stack health endpoint and does not need OpenAI compatibility.

## Binary response note

OpenAI `Create speech` returns audio bytes directly for normal audio streaming.
JSON Schema cannot validate raw response bodies, so
`openai-audio-speech-response.schema.json` captures response metadata and the
supported content types/formats that the adapter must emit.

First-pass implementation target:

```text
response_format=pcm, stream_format=audio -> audio/pcm, 24kHz mono s16le chunks
response_format=wav                      -> audio/wav, non-streaming
```

Other official formats (`mp3`, `opus`, `aac`, `flac`) remain in the request
schema but can return `unsupported_feature_error` until encoder support exists.

## Local model and voice mapping

The request schema follows OpenAI's TTS request shape, but model and voice
values are local Qwen3 identifiers. We intentionally do not pretend to expose
OpenAI model or voice inventories.
