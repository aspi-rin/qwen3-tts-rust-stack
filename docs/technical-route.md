# Qwen3-TTS-Rust 技术路线说明

## 背景

需要一个相对独立、可本地部署的 TTS 服务，用于实时语音场景。已有 PyTorch/ROCm 方案在当前环境下延迟和 RTF 偏高，不适合作为实时 TTS 主链路。

因此本轮验证非 PyTorch/ROCm 路线：**Qwen3-TTS-Rust + GGUF + llama.cpp backend + Vulkan**。

## 核心需求

- 本地部署：服务独立运行，不依赖外部云 TTS，后续可容器化。
- 中文质量可接受：常见短句、中等长度回复自然度可接受，中英混读不明显异常。
- 实时性：优先关注 RTF；理想 `< 0.5`，最低 `< 1.0`。
- 可服务化：后续封装 HTTP API，优先 raw PCM/WAV；若底层支持流式生成，保留流式能力。
- Docker 友好：生产有 GPU，开发可能无 GPU，因此验证脚本允许在无 GPU 环境只完成构建/静态检查。

## 推荐技术栈

```text
Qwen3-TTS
+ GGUF
+ llama.cpp backend
+ CPU / Vulkan / CUDA deployment targets
+ Rust
```

## 推进计划

1. 优先使用官方 release binary 和 Docker Compose 跑通 Qwen3-TTS-Rust CLI；必要时再从源码构建。
2. 验证 Vulkan backend 是否生效。
3. 测试中文质量、TTFB、RTF、稳定性。
4. 达标后封 HTTP API。
5. 最后再接入上层系统。

## 验证矩阵

| 项目 | 方法 | 通过标准 |
| --- | --- | --- |
| CLI 可运行 | `scripts/run_cli.sh` | 生成有效 WAV |
| Vulkan 可见 | `scripts/verify_vulkan.sh` | `vulkaninfo` 可列出 GPU，runtime 下载/加载 Vulkan 版本 |
| 中文质量 | 人工听测 | 无明显吞字/重复/爆音 |
| RTF | `scripts/benchmark_cli.py` | 初步 `< 1.0`，理想 `< 0.5` |
| 稳定性 | 连续多轮生成 | 多轮不崩溃，输出长度合理 |
| 流式潜力 | 检查上游 streaming API / WebSocket | 能输出分块 PCM/f32 样本 |

## 风险

- Vulkan 在容器内依赖宿主机 `/dev/dri`、ICD、驱动版本，需单独验证。
- Docker 默认固定官方 release `v0.1.6`。该版本上游只发布 Linux Vulkan asset；CPU 暂时复用该 asset 无 GPU passthrough，CUDA 需后续补 Linux CUDA source-build image。后续升级需同步更新 `release.lock` 和 SHA256。
- 上游首次运行自动下载模型，生产部署建议预热并固化缓存目录。
- CLI RTF 不等同服务端 TTFB；封 API 后需要单独测首包延迟和并发行为。
