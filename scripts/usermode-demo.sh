#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

GO_RUN=(env GOCACHE="${GOCACHE:-/tmp/gocache}" go run ./cmd/snb)

PIDS=()
LISTENER_PIDS=()
cleanup() {
  for pid in "${PIDS[@]:-}"; do
    kill "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
  done
}
trap cleanup EXIT INT TERM

echo "[1/4] starting usermode bridge"
"${GO_RUN[@]}" bridge udp \
  --port left:127.0.0.1:10001:127.0.0.1:11001 \
  --port right:127.0.0.1:10002:127.0.0.1:11002 \
  --port tap:127.0.0.1:10003:127.0.0.1:11003 &
PIDS+=("$!")

sleep 1

echo "[2/4] starting listeners"
"${GO_RUN[@]}" frame listen --listen 127.0.0.1:11001 --count 1 &
PIDS+=("$!")
LISTENER_PIDS+=("$!")
"${GO_RUN[@]}" frame listen --listen 127.0.0.1:11002 --count 1 &
PIDS+=("$!")
LISTENER_PIDS+=("$!")
"${GO_RUN[@]}" frame listen --listen 127.0.0.1:11003 --count 1 &
PIDS+=("$!")
LISTENER_PIDS+=("$!")

sleep 1

echo "[3/4] sending frames"
"${GO_RUN[@]}" frame send \
  --to 127.0.0.1:10001 \
  --src 02:00:00:00:00:01 \
  --dst ff:ff:ff:ff:ff:ff \
  --payload "hello-from-left"

"${GO_RUN[@]}" frame send \
  --to 127.0.0.1:10002 \
  --src 02:00:00:00:00:02 \
  --dst 02:00:00:00:00:01 \
  --payload "hello-from-right"

echo "[4/4] waiting for listeners to drain"
for pid in "${LISTENER_PIDS[@]}"; do
  wait "$pid"
done
