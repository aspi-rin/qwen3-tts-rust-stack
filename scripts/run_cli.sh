#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEXT="${1:-你好，我是本地语音合成服务。}"
OUTPUT="${2:-$ROOT/outputs/output.wav}"
SPEAKER="${SPEAKER:-vivian}"
QUANT="${QUANT:-q5_k_m}"
THREADS="${THREADS:-4}"
FEATURES="${FEATURES:-vulkan}"
"$ROOT/scripts/build_cli.sh"
mkdir -p "$(dirname "$OUTPUT")" "$ROOT/models"
export RUST_LOG="${RUST_LOG:-info}"
time cargo run --manifest-path "$ROOT/upstream/Cargo.toml" --release --features "$FEATURES" --bin qwen3_tts -- \
  --model-dir "$ROOT/models" \
  --quant "$QUANT" \
  --threads "$THREADS" \
  --speakers-dir "$ROOT/upstream/speakers" \
  --speaker "$SPEAKER" \
  --text "$TEXT" \
  --output "$OUTPUT"
echo "Wrote $OUTPUT"
