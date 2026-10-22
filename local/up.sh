#!/usr/bin/env bash
# Builds and starts neko-rooms for local / tailnet use, no domain needed.
#
#   ./local/up.sh            build & start
#   ./local/up.sh --no-build start without rebuilding
#   ./local/up.sh --pi       Raspberry Pi profile: published image, lighter rooms
#                            (picked automatically on a Pi, --no-pi to disable)
#   ./local/up.sh down       stop (rooms keep running until stopped in the UI)
#   ./local/up.sh logs       follow logs
set -euo pipefail

cd "$(dirname "$0")/.."
ENV_FILE=".env.local"

action="up"
build=1
pi=""
for arg in "$@"; do
  case "$arg" in
    --pi) pi=1 ;;
    --no-pi) pi=0 ;;
    --no-build) build=0 ;;
    down|logs) action="$arg" ;;
    *) echo "unknown argument: $arg" >&2; sed -n '3,9p' "$0" >&2; exit 1 ;;
  esac
done

if [ -z "$pi" ]; then
  pi=0
  if grep -qi "raspberry pi" /proc/device-tree/model 2>/dev/null; then
    pi=1
    [ "$action" = "up" ] && echo "Raspberry Pi detected, using the Pi profile (--no-pi to disable)"
  fi
fi

COMPOSE=(docker compose -f docker-compose.local.yml)
[ "$pi" = 1 ] && COMPOSE+=(-f docker-compose.pi.yml)
COMPOSE+=(--env-file "$ENV_FILE")

case "$action" in
  down) touch "$ENV_FILE"; "${COMPOSE[@]}" down; exit 0 ;;
  logs) "${COMPOSE[@]}" logs -f neko-rooms; exit 0 ;;
esac

# IP that devices use to reach this machine: tailscale first, LAN otherwise
ip=""
if command -v tailscale >/dev/null 2>&1; then
  ip="$(tailscale ip -4 2>/dev/null | head -n1 || true)"
fi
if [ -z "$ip" ]; then
  ip="$(hostname -I 2>/dev/null | awk '{print $1}' || true)"
  echo "! tailscale not found, falling back to LAN IP $ip"
fi
if [ -z "$ip" ]; then
  ip="127.0.0.1"
  echo "! no network IP found, rooms will only work on this machine"
fi

dns=""
if command -v tailscale >/dev/null 2>&1; then
  dns="$(tailscale status --json 2>/dev/null | sed -n 's/.*"DNSName": *"\([^"]*\)\.".*/\1/p' | head -n1 || true)"
fi

# keep user edits, only (re)write the detected IP
touch "$ENV_FILE"
if grep -q '^NEKO_ROOMS_NAT1TO1=' "$ENV_FILE"; then
  sed -i.bak "s/^NEKO_ROOMS_NAT1TO1=.*/NEKO_ROOMS_NAT1TO1=$ip/" "$ENV_FILE" && rm -f "$ENV_FILE.bak"
else
  echo "NEKO_ROOMS_NAT1TO1=$ip" >> "$ENV_FILE"
fi
grep -q '^TZ=' "$ENV_FILE" || echo "TZ=$(cat /etc/timezone 2>/dev/null || echo UTC)" >> "$ENV_FILE"

# web UI port: keep the saved one, otherwise the first free one from 8188
port_busy() {
  if command -v ss >/dev/null 2>&1 && ss -Hltn 2>/dev/null | awk '{print $4}' | grep -qE "[:.]$1\$"; then
    return 0
  fi
  (exec 3<>"/dev/tcp/127.0.0.1/$1") 2>/dev/null
}
port="$(sed -n 's/^NEKO_ROOMS_PORT=//p' "$ENV_FILE" | head -n1)"
if [ -z "$port" ]; then
  port=8188
  while port_busy "$port"; do port=$((port + 1)); done
  echo "NEKO_ROOMS_PORT=$port" >> "$ENV_FILE"
fi

if [ "$pi" = 1 ]; then
  # the Pi uses the published multi-arch image, building there takes ages
  "${COMPOSE[@]}" pull neko-rooms
elif [ "$build" = 1 ] || ! docker image inspect neko-rooms:local >/dev/null 2>&1; then
  bash local/build.sh
fi
"${COMPOSE[@]}" up -d --force-recreate

# pull the default room image once, rooms cannot start without it
if ! docker image inspect ghcr.io/m1k1o/neko/firefox >/dev/null 2>&1; then
  echo "pulling ghcr.io/m1k1o/neko/firefox (first room image)..."
  docker pull ghcr.io/m1k1o/neko/firefox
fi

echo
echo "neko-rooms is running:"
echo "  this machine  http://localhost:$port"
echo "  tailnet       http://$ip:$port"
[ -n "$dns" ] && echo "  MagicDNS      http://$dns:$port"

# first start: show the one-time setup token
token=""
for _ in $(seq 1 15); do
  token="$(docker logs neko-rooms-local 2>&1 | sed 's/\x1b\[[0-9;]*m//g' \
    | sed -n 's/.*setup_token[":= ]*\([A-Za-z0-9_-]\{8,\}\).*/\1/p' | tail -n1 || true)"
  [ -n "$token" ] && break
  sleep 1
done
if [ -n "$token" ]; then
  echo
  echo "first start: open the site and create the admin account with setup token"
  echo "  $token"
fi
