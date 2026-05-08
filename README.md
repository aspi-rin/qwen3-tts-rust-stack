# qwen3-tts-rust-stack

Self-hosted Text-to-Speech stack. 当前阶段用于验证 **Qwen3-TTS + GGUF + llama.cpp backend + Vulkan + Rust** 是否适合作为本地实时 TTS 主链路。

> 当前目标是技术路线验证，不是直接上线。优先跑通 CLI、确认 Vulkan、测试中文质量/RTF；达标后再封装 OpenAI 风格 HTTP API。

## 上游版本

- Upstream: <https://github.com/cgisky1980/Qwen3-TTS-Rust>
- Docker release pin: `v0.1.6`，详见 [`release.lock`](./release.lock)
  - Linux Vulkan: official `qwen3-tts-linux-x64-vulkan.tar.gz`
  - Linux CPU: upstream has no separate CPU asset; this stack reuses the same Linux Vulkan asset without GPU passthrough and relies on llama.cpp/ggml CPU fallback
- Source fallback commit: `32ed8f03c1ca9fbdcb3a888cb4006ca10ccfc74e`，详见 [`upstream.lock`](./upstream.lock)

## 验证顺序

1. 跑通 Qwen3-TTS-Rust CLI
2. 验证 llama.cpp Vulkan runtime 是否下载/加载
3. 测试中文质量、TTFB、RTF、稳定性
4. 如果达标，再封 HTTP API
5. 最后接入上层系统

## Quick start

Requirements: Linux, Docker with Compose plugin.

```bash
cp .env.example .env
make build-image
make run
```

默认 `BACKEND=vulkan`，会把 `/dev/dri` 暴露给容器。可选 backend：

```bash
BACKEND=vulkan make run   # 挂载 /dev/dri；AMD/Intel/NVIDIA Vulkan 路线
BACKEND=cpu make run      # 不挂 GPU 设备；复用 Linux Vulkan release 的 CPU fallback
```

首次运行会自动下载模型，耗时取决于网络。`make run` 会自动创建 `models/` 和 `outputs/`，并执行一条固定的 CLI smoke 合成，结果写到 `outputs/speech.wav`。


## Configuration

`.env` intentionally only contains runtime knobs, not sample invocation text/output:

| Variable | Default | Purpose |
| --- | --- | --- |
| `QWEN3_TTS_BACKEND` | `vulkan` | `vulkan` passes `/dev/dri`; `cpu` does not. |
| `QWEN3_TTS_QUANT` | `q5_k_m` | Qwen3-TTS quantization helper. |
| `QWEN3_TTS_SPEAKER` | `vivian` | Built-in speaker name. |

Container-internal paths are fixed: models at `/app/models`, speakers at `/app/speakers`, outputs at `/app/outputs`.

## Docker 构建与运行

默认 Dockerfile 使用官方 GitHub Release 二进制包，不在镜像内编译 Rust。当前固定版本见 [`release.lock`](./release.lock)。

注意：上游 `v0.1.6` 只发布了 Linux Vulkan asset；CPU 模式暂时复用该 asset，但不做 GPU passthrough，依赖 llama.cpp/ggml CPU fallback。

```bash
BACKEND=vulkan make build-image   # 基于官方 Linux Vulkan release binary 构建镜像
BACKEND=vulkan make run           # 使用 Compose 执行一次合成任务
make down                         # 清理 Compose 资源
```

如果在 AMD GPU 主机上运行，建议确认宿主机可用：

```bash
vulkaninfo --summary
```

## 初步通过标准

- RTF `< 1.0`
- 无明显吞字、重复、爆音
- 连续多次生成稳定

理想标准：RTF `< 0.5`，TTFB 较低，支持流式 PCM 输出。

## 后续 API 方向

如果 CLI 验证通过，API 设计靠近 OpenAI TTS：

```http
POST /v1/audio/speech
Content-Type: application/json
```

```json
{
  "model": "qwen3-tts",
  "input": "你好，我是本地语音合成服务。",
  "voice": "default",
  "response_format": "pcm"
}
```

输出优先级：raw PCM → WAV → 其他编码格式。详见 [`docs/api-contract.md`](./docs/api-contract.md)。

## 重要说明

- 默认 Docker 镜像使用官方 release asset；如需从源码构建，可手动使用 `docker/Dockerfile.source`。
- Pinned release `v0.1.6` 的 CLI 不支持 `--threads`；Compose 参数需以 `release.lock` 固定的 release `--help` 为准。
- 开发机可能没有 GPU，因此可用 `BACKEND=cpu` 跳过 GPU passthrough；性能仍需在目标 GPU 机器上确认。
- 上游 README 声称 Linux/Windows 默认 Vulkan，macOS 默认 Metal；实际是否生效需在目标机器验证 runtime 日志和 RTF。
- 上游当前也包含 `qwen3_tts_server`，但先不要把它视为最终服务 API；本轮先用 CLI 验证路线。
