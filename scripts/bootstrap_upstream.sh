#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOCK="$ROOT/upstream.lock"
REPO="$(grep '^repo=' "$LOCK" | cut -d= -f2-)"
COMMIT="$(grep '^commit=' "$LOCK" | cut -d= -f2-)"
UPSTREAM="$ROOT/upstream"
if [[ ! -d "$UPSTREAM/.git" ]]; then
  git clone "$REPO" "$UPSTREAM"
fi
git -C "$UPSTREAM" fetch --depth 1 origin "$COMMIT"
git -C "$UPSTREAM" checkout --detach "$COMMIT"
echo "Upstream ready: $(git -C "$UPSTREAM" rev-parse HEAD)"
