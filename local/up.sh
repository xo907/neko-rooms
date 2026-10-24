#!/usr/bin/env bash
# Builds and starts neko-rooms: on this machine, a tailnet, a public IP or
# behind Nginx Proxy Manager.
#
#   ./local/up.sh            build & start, reachable over tailscale / LAN
#   ./local/up.sh --no-build start without rebuilding
#   ./local/up.sh --pi       Raspberry Pi profile: published image, lighter rooms
#                            (picked automatically on a Pi, --no-pi to disable)
#   ./local/up.sh --public   reachable on the public IP, e.g. on a VPS, uses the
#                            published image (remembered, --no-public to undo)
#   ./local/up.sh --npm      behind Nginx Proxy Manager on this server, with your
#                            domain and HTTPS (remembered, --no-npm to undo)
#   ./local/up.sh --subtitles   live subtitles with a local speech server
#                            (whisper.cpp, x86 only; remembered, --no-subtitles)
#   ./local/up.sh --translate   subtitles in other languages too, with a local
#                            LibreTranslate (remembered, --no-translate)
#   ./local/up.sh down       stop (rooms keep running until stopped in the UI)
#   ./local/up.sh logs       follow logs
set -euo pipefail

cd "$(dirname "$0")/.."
ENV_FILE=".env.local"
touch "$ENV_FILE"

# saved reads a value from .env.local, setenv writes one
saved() { sed -n "s/^$1=//p" "$ENV_FILE" | head -n1; }
setenv() {
  if grep -q "^$1=" "$ENV_FILE"; then
    sed -i.bak "s|^$1=.*|$1=$2|" "$ENV_FILE" && rm -f "$ENV_FILE.bak"
  else
    echo "$1=$2" >> "$ENV_FILE"
  fi
}

action="up"
build=1
pi=""
public=""
npm=""
subtitles=""
translate=""
for arg in "$@"; do
  case "$arg" in
    --pi) pi=1 ;;
    --no-pi) pi=0 ;;
    --public) public=1 ;;
    --no-public) public=0 ;;
    --npm) npm=1 ;;
    --no-npm) npm=0 ;;
    --subtitles) subtitles=1 ;;
    --no-subtitles) subtitles=0 ;;
    --translate) translate=1; subtitles=1 ;;
    --no-translate) translate=0 ;;
    --no-build) build=0 ;;
    down|logs) action="$arg" ;;
    *) echo "unknown argument: $arg" >&2; sed -n '4,18p' "$0" >&2; exit 1 ;;
  esac
done

if [ -z "$pi" ]; then
  pi=0
  if grep -qi "raspberry pi" /proc/device-tree/model 2>/dev/null; then
    pi=1
    [ "$action" = "up" ] && echo "Raspberry Pi detected, using the Pi profile (--no-pi to disable)"
  fi
fi

# --npm and --public are remembered in .env.local
if [ -z "$npm" ]; then
  npm="$(saved NEKO_NPM)"
  npm="${npm:-0}"
elif [ "$action" = "up" ]; then
  setenv NEKO_NPM "$npm"
  # back to a site that is reachable directly
  [ "$npm" = 0 ] && setenv NEKO_ROOMS_BIND_IP 0.0.0.0
fi
if [ -z "$public" ]; then
  public="$(saved NEKO_PUBLIC)"
  public="${public:-0}"
elif [ "$action" = "up" ]; then
  setenv NEKO_PUBLIC "$public"
fi
# subtitles are remembered too
if [ -z "$subtitles" ]; then
  subtitles="$(saved NEKO_SUBTITLES)"
  subtitles="${subtitles:-0}"
elif [ "$action" = "up" ]; then
  setenv NEKO_SUBTITLES "$subtitles"
fi
if [ -z "$translate" ]; then
  translate="$(saved NEKO_TRANSLATE)"
  translate="${translate:-0}"
elif [ "$action" = "up" ]; then
  setenv NEKO_TRANSLATE "$translate"
fi
[ "$subtitles" = 1 ] || translate=0
if [ "$subtitles" = 1 ] && [ "$action" = "up" ] && [ "$(uname -m)" != "x86_64" ]; then
  echo "! subtitles need an x86 machine, the speech server has no $(uname -m) build. Skipping them." >&2
  subtitles=0
  translate=0
fi

# behind the proxy the room video still goes straight to the public IP
[ "$npm" = 1 ] && public=1

# find Nginx Proxy Manager and how it can reach neko-rooms
npm_target=""
if [ "$npm" = 1 ] && [ "$action" = "up" ]; then
  npm_ctr="$(saved NPM_CONTAINER)"
  if [ -z "$npm_ctr" ]; then
    npm_ctr="$(docker ps --format '{{.Names}} {{.Image}}' | awk '$2 ~ /nginx-proxy-manager/ {print $1; exit}')"
  fi
  if [ -z "$npm_ctr" ]; then
    echo "! Nginx Proxy Manager is not running on this server." >&2
    echo "  Start it first, or set its container name: echo NPM_CONTAINER=<name> >> $ENV_FILE" >&2
    exit 1
  fi

  npm_net="$(docker inspect "$npm_ctr" --format '{{range $k, $v := .NetworkSettings.Networks}}{{$k}} {{end}}' | awk '{print $1}')"
  case "$npm_net" in
    host)
      setenv NPM_NETWORK ""
      setenv NEKO_ROOMS_BIND_IP 127.0.0.1
      npm_target="127.0.0.1" ;;
    bridge|"")
      # the default bridge has no names, publish on its gateway only
      gw="$(docker network inspect bridge --format '{{range .IPAM.Config}}{{.Gateway}}{{end}}')"
      setenv NPM_NETWORK ""
      setenv NEKO_ROOMS_BIND_IP "${gw:-172.17.0.1}"
      npm_target="${gw:-172.17.0.1}" ;;
    *)
      setenv NPM_NETWORK "$npm_net"
      setenv NEKO_ROOMS_BIND_IP 127.0.0.1
      npm_target="neko-rooms-local" ;;
  esac
fi

COMPOSE=(docker compose -f docker-compose.local.yml)
release=0
if [ "$pi" = 1 ] || [ "$public" = 1 ]; then
  # published multi-arch image, no build on small or remote machines
  release=1
  COMPOSE+=(-f docker-compose.release.yml)
fi
[ "$pi" = 1 ] && COMPOSE+=(-f docker-compose.pi.yml)
if [ "$npm" = 1 ]; then
  COMPOSE+=(-f docker-compose.npm.yml)
  [ -n "$(saved NPM_NETWORK)" ] && COMPOSE+=(-f docker-compose.npm-network.yml)
fi
[ "$subtitles" = 1 ] && COMPOSE+=(-f docker-compose.subtitles.yml)
[ "$translate" = 1 ] && COMPOSE+=(-f docker-compose.translate.yml)
COMPOSE+=(--env-file "$ENV_FILE")

case "$action" in
  down) "${COMPOSE[@]}" down; exit 0 ;;
  logs) "${COMPOSE[@]}" logs -f neko-rooms; exit 0 ;;
esac

is_private() {
  case "$1" in
    10.*|192.168.*|127.*|169.254.*|172.1[6-9].*|172.2[0-9].*|172.3[01].*) return 0 ;;
    100.6[4-9].*|100.[7-9][0-9].*|100.1[01][0-9].*|100.12[0-7].*) return 0 ;; # CGNAT, tailscale
  esac
  return 1
}

# IP that browsers use to reach the room video streams
ip=""
dns=""
if [ "$public" = 1 ]; then
  # NEKO_ROOMS_PUBLIC_IP in .env.local wins, e.g. behind a cloud NAT
  ip="$(saved NEKO_ROOMS_PUBLIC_IP)"
  if [ -z "$ip" ]; then
    ip="$(ip -4 route get 1.1.1.1 2>/dev/null | sed -n 's/.* src \([0-9.]*\).*/\1/p' || true)"
    if [ -z "$ip" ] || is_private "$ip"; then
      echo "! could not find a public IP on this machine (found: ${ip:-none})." >&2
      echo "  Add your public IP to $ENV_FILE and run again:" >&2
      echo "    echo NEKO_ROOMS_PUBLIC_IP=<your public ip> >> $ENV_FILE" >&2
      exit 1
    fi
  fi
else
  # tailscale first, LAN otherwise
  if command -v tailscale >/dev/null 2>&1; then
    ip="$(tailscale ip -4 2>/dev/null | head -n1 || true)"
    dns="$(tailscale status --json 2>/dev/null | sed -n 's/.*"DNSName": *"\([^"]*\)\.".*/\1/p' | head -n1 || true)"
  fi
  if [ -z "$ip" ]; then
    ip="$(hostname -I 2>/dev/null | awk '{print $1}' || true)"
    echo "! tailscale not found, falling back to LAN IP $ip"
  fi
  if [ -z "$ip" ]; then
    ip="127.0.0.1"
    echo "! no network IP found, rooms will only work on this machine"
  fi
fi

# keep user edits, only (re)write the detected IP
setenv NEKO_ROOMS_NAT1TO1 "$ip"
[ -n "$(saved TZ)" ] || setenv TZ "$(cat /etc/timezone 2>/dev/null || echo UTC)"

# web UI port: keep the saved one, otherwise the first free one from 8188
port_busy() {
  if command -v ss >/dev/null 2>&1 && ss -Hltn 2>/dev/null | awk '{print $4}' | grep -qE "[:.]$1\$"; then
    return 0
  fi
  (exec 3<>"/dev/tcp/127.0.0.1/$1") 2>/dev/null
}
port="$(saved NEKO_ROOMS_PORT)"
if [ -z "$port" ]; then
  port=8188
  while port_busy "$port"; do port=$((port + 1)); done
  setenv NEKO_ROOMS_PORT "$port"
fi

if [ "$release" = 1 ]; then
  "${COMPOSE[@]}" pull neko-rooms
elif [ "$build" = 1 ] || ! docker image inspect neko-rooms:local >/dev/null 2>&1; then
  bash local/build.sh
fi
"${COMPOSE[@]}" up -d --force-recreate

# pull the default room image once, rooms cannot start without it
if ! docker image inspect ghcr.io/m1k1o/neko/firefox:3.1.6 >/dev/null 2>&1; then
  echo "pulling ghcr.io/m1k1o/neko/firefox:3.1.6 (first room image)..."
  docker pull ghcr.io/m1k1o/neko/firefox:3.1.6
fi

epr="$(saved NEKO_ROOMS_EPR)"
epr="${epr:-59000-59049}"

echo
echo "neko-rooms is running."
if [ "$npm" = 1 ]; then
  fport=8080
  [ "$npm_target" = "neko-rooms-local" ] || fport="$port"
  cat <<EOF

Add it in the Nginx Proxy Manager web UI (usually http://<server ip>:81):

  Hosts > Proxy Hosts > Add Proxy Host

  Details tab
    Domain Names            rooms.example.com  (a DNS A record pointing to $ip)
    Scheme                  http
    Forward Hostname / IP   $npm_target
    Forward Port            $fport
    Block Common Exploits   on
    Websockets Support      on

  SSL tab
    SSL Certificate         Request a new SSL Certificate
    Force SSL               on
    HTTP/2 Support          on

  Advanced tab, Custom Nginx Configuration (keeps live updates connected)
    proxy_read_timeout 1h;
    proxy_send_timeout 1h;
    proxy_buffering off;

Save, then open https://rooms.example.com (your domain).

Open TCP and UDP ${epr/-/:} in your firewall for the room video, it goes
straight to $ip, not through the proxy. Port $port stays private.
EOF
elif [ "$public" = 1 ]; then
  echo "  this machine  http://localhost:$port"
  echo "  public        http://$ip:$port"
  echo
  echo "Open these ports in your firewall (and the VPS provider's, if it has one):"
  printf '  %-26s %s\n' "TCP $port" "the website" "TCP and UDP ${epr/-/:}" "room video, one port per room"
  echo
  echo "Note: this is plain HTTP, passwords are sent unencrypted. For HTTPS use"
  echo "./local/up.sh --npm with Nginx Proxy Manager, see docs/local.md."
else
  echo "  this machine  http://localhost:$port"
  echo "  tailnet       http://$ip:$port"
  [ -n "$dns" ] && echo "  MagicDNS      http://$dns:$port"
fi
if [ "$public" = 1 ] && command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q "Status: active"; then
  echo
  echo "ufw is active, for example:"
  [ "$npm" = 1 ] || echo "  ufw allow $port/tcp"
  echo "  ufw allow ${epr/-/:}/tcp && ufw allow ${epr/-/:}/udp"
fi

if [ "$subtitles" = 1 ]; then
  echo
  echo "Subtitles: the speech server downloads its model on first start"
  [ "$translate" = 1 ] && echo "and the translation server its languages, a few minutes"
  echo "(./local/up.sh logs). Then CC appears in rooms."
fi

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
