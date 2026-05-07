#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "== Vulkan devices =="
if command -v vulkaninfo >/dev/null 2>&1; then
  vulkaninfo --summary || true
else
  echo "vulkaninfo not installed"
fi
echo "== DRM devices =="
ls -l /dev/dri 2>/dev/null || echo "/dev/dri not available"
echo "== llama runtime artifacts =="
find "$ROOT/data/models" "$ROOT/upstream/runtime" -maxdepth 3 \( -name '*llama*' -o -name '*ggml*' \) -print 2>/dev/null || true
echo "Note: final confirmation is runtime logs + RTF on target GPU."
