#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
"$ROOT/scripts/bootstrap_upstream.sh"
FEATURES="${FEATURES:-vulkan}"
BIN="${BIN:-qwen3_tts}"
if [[ -n "$FEATURES" ]]; then
  cargo build --manifest-path "$ROOT/upstream/Cargo.toml" --release --features "$FEATURES" --bin "$BIN"
else
  cargo build --manifest-path "$ROOT/upstream/Cargo.toml" --release --bin "$BIN"
fi
