# HTTP API 草案

目标：对齐 OpenAI TTS 风格，同时保留本地流式 PCM 能力。

## 非流式

`POST /v1/audio/speech`

请求：

```json
{
  "model": "qwen3-tts",
  "input": "你好，我是本地语音合成服务。",
  "voice": "default",
  "response_format": "pcm",
  "instruction": "自然、清晰",
  "temperature": 0.7,
  "top_k": 40,
  "top_p": 0.9,
  "seed": 42
}
```

响应格式：

- `response_format=pcm`: `audio/pcm; rate=24000; format=f32le` 或最终确定的 PCM 格式。
- `response_format=wav`: `audio/wav`。

## 流式

优先方案：HTTP chunked response，直接返回 PCM chunks。

```http
POST /v1/audio/speech?stream=true
Accept: audio/pcm
```

备选方案：WebSocket 或 SSE + base64 音频块。

## 映射关系

| OpenAI 风格字段 | Qwen3-TTS-Rust 字段 |
| --- | --- |
| `input` | CLI/API `text` |
| `voice` | speaker name 或 voice file |
| `response_format` | pcm/wav 输出选择 |
| `model` | 固定 `qwen3-tts`，后续可映射 quant/backend |
