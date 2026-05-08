# Pinned OpenAI-compatible speech schema

This stack pins the request shape for the OpenAI-compatible adapter in:

```text
schemas/openai/openai-audio-speech-request.schema.json
```

Source baseline: OpenAI `POST /v1/audio/speech` / Create speech request body,
checked on 2026-05-08.

## Request body

Required fields:

| Field | Type | Notes |
| --- | --- | --- |
| `model` | string enum | `qwen3-tts`. |
| `input` | string | Text to synthesize; max 4096 chars. |
| `voice` | string enum or `{ "id": string }` | Local Qwen3 speaker, e.g. `vivian`. |

Optional fields:

| Field | Type | Default | Notes |
| --- | --- | --- | --- |
| `instructions` | string | unset | Voice/tone/pacing instructions; max 4096 chars. |
| `response_format` | enum | `mp3` | `mp3`, `opus`, `aac`, `flac`, `wav`, `pcm`. |
| `speed` | number | `1.0` | Range `0.25` to `4.0`. |
| `stream_format` | enum | `audio` | `audio` or `sse`. |

## First implementation policy

The schema fixes the OpenAI-compatible request shape while using local Qwen3
model and speaker identifiers.
The first adapter implementation may still reject unsupported runtime
combinations with a clear OpenAI-style 4xx error.

Planned first-pass support:

| Request | Behavior |
| --- | --- |
| `response_format=pcm`, `stream_format=audio` | HTTP streaming PCM, 24kHz mono. |
| `response_format=wav` | Non-streaming WAV. |
| `instructions` | Forward to upstream `instruction`. |
| `speed` | Accept, validate, initially ignore. |
| `stream_format=sse` | Validate schema, return unsupported until implemented. |
| `mp3`/`opus`/`aac`/`flac` | Validate schema, return unsupported until encoder support is added. |

## Pipecat compatibility target

Pipecat-style clients use this speech endpoint shape with local model/voice values:

```json
{
  "input": "...",
  "model": "qwen3-tts",
  "voice": "vivian",
  "response_format": "pcm",
  "instructions": "optional",
  "speed": 1.0
}
```

It reads the response with OpenAI SDK streaming response bytes and expects
24kHz mono PCM chunks. The adapter should support that subset first while
keeping the broader OpenAI schema pinned above.

## References

- OpenAI API Reference: `Create speech` (`POST /v1/audio/speech`).
- Pipecat `OpenAITTSService` documentation and source for the practical `pcm`
  streaming subset used by voice pipelines.

See [`openai-tts-contract.md`](./openai-tts-contract.md) for all related request, response, error, SSE, and model-list schemas.
