#!/usr/bin/env bash
# Builds the neko-rooms:local image in small, memory capped steps.
#
# The root Dockerfile compiles the client and the server in parallel, which
# can exhaust smaller machines and WSL VMs. Here every step runs alone in a
# container with fixed limits, then local/Dockerfile packs the results.
set -euo pipefail

cd "$(dirname "$0")/.."
ROOT="$PWD"
OUT="$ROOT/local/.build"
MEM="${BUILD_MEMORY:-2g}"

rm -rf "$OUT" && mkdir -p "$OUT"

# trust the same CAs as the host (TLS inspecting proxies, antivirus…)
CERTS=()
[ -d /etc/ssl/certs ] && CERTS=(-v /etc/ssl/certs:/etc/ssl/certs:ro)

echo "==> building client"
docker run --rm --network host --memory="$MEM" --memory-swap="$MEM" --cpus=2 "${CERTS[@]}" \
  -v "$ROOT/client:/src:ro" -v neko-rooms-npm:/root/.npm -v "$OUT:/out" \
  node:20-bookworm-slim sh -ec '
    mkdir /build && cd /src && tar --exclude=./node_modules --exclude=./dist -cf - . | tar -xf - -C /build
    cd /build && npm ci --no-audit --no-fund --loglevel=error && npm run build >/dev/null
    cp -r dist /out/www'

echo "==> building server"
docker run --rm --network host --memory="$MEM" --memory-swap="$MEM" --cpus=1 "${CERTS[@]}" \
  -e GOMAXPROCS=1 -e GOGC=40 -e GOFLAGS=-p=1 -e CGO_ENABLED=0 \
  -v "$ROOT:/src:ro" -v neko-rooms-gomod:/go/pkg/mod -v neko-rooms-gocache:/root/.cache/go-build -v "$OUT:/out" \
  -w /src golang:1.24-bookworm go build -ldflags="-s -w" -o /out/neko_rooms ./cmd/neko_rooms

echo "==> packing image"
cp local/Dockerfile "$OUT/Dockerfile"
docker build -q -t neko-rooms:local "$OUT"
rm -rf "$OUT"
