# Pinned OpenAI-compatible speech schema

This stack pins the request shape for the OpenAI-compatible adapter in:

```text
schemas/openai-audio-speech-request.schema.json
```

Source baseline: OpenAI `POST /v1/audio/speech` / Create speech request body,
checked on 2026-05-08.

## Request body

Required fields:

| Field | Type | Notes |
| --- | --- | --- |
| `model` | string enum | `tts-1`, `tts-1-hd`, `gpt-4o-mini-tts`, `gpt-4o-mini-tts-2025-12-15`. |
| `input` | string | Text to synthesize; max 4096 chars. |
| `voice` | string enum or `{ "id": string }` | OpenAI built-in voice or custom voice object shape. |

Optional fields:

| Field | Type | Default | Notes |
| --- | --- | --- | --- |
| `instructions` | string | unset | Voice/tone/pacing instructions; max 4096 chars. |
| `response_format` | enum | `mp3` | `mp3`, `opus`, `aac`, `flac`, `wav`, `pcm`. |
| `speed` | number | `1.0` | Range `0.25` to `4.0`. |
| `stream_format` | enum | `audio` | `audio` or `sse`. |

## First implementation policy

The schema fixes the OpenAI-compatible request shape. Local Qwen speaker mapping
should be handled by adapter configuration, not by expanding this request schema.
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

Pipecat's `OpenAITTSService` uses OpenAI's speech endpoint with:

```json
{
  "input": "...",
  "model": "gpt-4o-mini-tts",
  "voice": "alloy",
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
