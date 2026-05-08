# Qwen3-TTS-Rust 技术路线说明

## 背景

需要一个相对独立、可本地部署的 TTS 服务，用于实时语音场景。已有 PyTorch/ROCm 方案在当前环境下延迟和 RTF 偏高，不适合作为实时 TTS 主链路。

因此本轮验证非 PyTorch/ROCm 路线：**Qwen3-TTS-Rust + GGUF + llama.cpp backend + Vulkan/CPU fallback**。

## 核心需求

- 本地部署：服务独立运行，不依赖外部云 TTS，后续可容器化。
- 中文质量可接受：常见短句、中等长度回复自然度可接受，中英混读不明显异常。
- 实时性：优先关注 RTF；理想 `< 0.5`，最低 `< 1.0`。
- 可服务化：启动长期 HTTP/WebSocket 服务，优先保留 streaming 能力。
- Docker 友好：生产有 GPU，开发可能无 GPU；CPU backend 仅作为无 GPU fallback/基本可运行性路径。

## 推荐技术栈

```text
Qwen3-TTS-Rust source
+ qwen3_tts_server
+ GGUF
+ llama.cpp backend
+ Vulkan / CPU fallback deployment targets
+ Docker Compose
```

## 推进计划

1. 从源码构建并启动 `qwen3_tts_server`。
2. 验证 Vulkan backend 是否生效。
3. 验证 upstream WebSocket streaming TTS。
4. 测试中文质量、TTFB、RTF、稳定性。
5. 达标后再通过 stack 层 Go adapter 封 OpenAI 风格 `/v1/audio/speech`，保持 upstream source pure。
6. 最后接入上层系统。

## 验证矩阵

| 项目 | 方法 | 通过标准 |
| --- | --- | --- |
| 服务可运行 | `make up` + `GET /health` | 返回 OK |
| Vulkan 可见 | 宿主机 `vulkaninfo --summary` + `BACKEND=vulkan make up` | `/dev/dri` 可见，runtime 日志/RTF 符合预期 |
| Streaming | `ws://host:3000/api/tts/stream` | 能收到分块音频数据 |
| 中文质量 | 人工听测 | 无明显吞字/重复/爆音 |
| RTF | 记录服务端生成耗时/音频时长 | 初步 `< 1.0`，理想 `< 0.5` |
| 稳定性 | 连续多轮请求 | 多轮不崩溃，输出长度合理 |

## 风险

- Vulkan 在容器内依赖宿主机 `/dev/dri`、ICD、驱动版本，需单独验证。
- 上游 `v0.1.6` release 只发布 CLI binary；本项目从源码构建 server binary。
- Docker 镜像仍复用 pinned runtime bundle 中的 Linux Vulkan llama.cpp/ONNX shared libraries；CPU 模式不透传 GPU，依赖 CPU fallback。
- 上游 server 的 streaming endpoint 是 WebSocket，不是 OpenAI-style HTTP chunked streaming；后续可能需要加适配层。
- 首次启动会下载模型到 `./models`，生产部署建议预热并固化模型目录。
