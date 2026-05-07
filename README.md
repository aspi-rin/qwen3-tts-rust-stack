# Qwen3-TTS-Rust 本地路线验证栈

本仓库用于验证 **Qwen3-TTS + GGUF + llama.cpp backend + Vulkan + Rust** 是否适合作为本地实时 TTS 主链路。

> 当前目标是技术路线验证，不是直接上线。优先跑通 CLI、确认 Vulkan、测试中文质量/RTF；达标后再封装 OpenAI 风格 HTTP API。

## 上游版本

- Upstream: <https://github.com/cgisky1980/Qwen3-TTS-Rust>
- Pinned commit: `32ed8f03c1ca9fbdcb3a888cb4006ca10ccfc74e`
- 详见 [`upstream.lock`](./upstream.lock)

## 验证顺序

1. 跑通 Qwen3-TTS-Rust CLI
2. 验证 llama.cpp Vulkan runtime 是否下载/加载
3. 测试中文质量、TTFB、RTF、稳定性
4. 如果达标，再封 HTTP API
5. 最后接入上层系统

## 快速开始（本机）

```bash
# 1) 拉取上游到 ./upstream，并固定到 upstream.lock 里的 commit
./scripts/bootstrap_upstream.sh

# 2) 编译 CLI（默认 release + vulkan feature）
./scripts/build_cli.sh

# 3) 生成一条中文测试音频
./scripts/run_cli.sh "你好，我是本地语音合成服务。" data/outputs/hello.wav

# 4) 连续基准测试，输出 RTF
./scripts/benchmark_cli.py --rounds 3 --speaker vivian
```

首次运行会自动下载模型、ONNX Runtime、llama.cpp runtime，耗时取决于网络。

## Docker 构建与运行

```bash
# 构建验证镜像
make docker-build

# CPU/无 GPU 开发机可做构建和基本 smoke test（可能无法达到实时）
make docker-smoke

# Vulkan GPU 机器：需要把 /dev/dri 暴露给容器
make docker-vulkan-test
```

如果在 AMD GPU 主机上运行，建议确认宿主机可用：

```bash
vulkaninfo --summary
```

容器侧 Vulkan smoke test：

```bash
docker run --rm --device=/dev/dri \
  --entrypoint bash \
  -v "$PWD/data:/app/data" \
  qwen3-tts-rust-stack:local \
  /app/scripts/verify_vulkan.sh
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

- 开发机可能没有 GPU，因此本仓库脚本不强制完整推理必须成功。
- 上游 README 声称 Linux/Windows 默认 Vulkan，macOS 默认 Metal；实际是否生效需在目标机器验证 runtime 日志和 RTF。
- 上游当前也包含 `qwen3_tts_server`，但先不要把它视为最终服务 API；本轮先用 CLI 验证路线。
