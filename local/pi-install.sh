#!/usr/bin/env bash
# One command install for a Raspberry Pi (Raspberry Pi OS 64-bit):
#
#   curl -fsSL https://raw.githubusercontent.com/xo907/neko-rooms/master/local/pi-install.sh | bash
#
# Installs Docker and Tailscale when missing, gets this repository and starts
# neko-rooms with the Pi profile. Safe to run again to update.
set -euo pipefail

REPO="${NEKO_ROOMS_REPO:-https://github.com/xo907/neko-rooms.git}"
DIR="${NEKO_ROOMS_DIR:-$HOME/neko-rooms}"

say() { printf '\n\033[1m==> %s\033[0m\n' "$*"; }

if [ "$(uname -m)" != "aarch64" ]; then
  echo "This needs a 64-bit OS (aarch64), found $(uname -m)." >&2
  echo "Install Raspberry Pi OS (64-bit), the room browsers only exist for arm64." >&2
  exit 1
fi

if ! command -v git >/dev/null 2>&1 || ! command -v curl >/dev/null 2>&1; then
  say "Installing git and curl"
  sudo apt-get update -qq
  sudo apt-get install -y -qq git curl
fi

if ! command -v docker >/dev/null 2>&1; then
  say "Installing Docker"
  curl -fsSL https://get.docker.com | sudo sh
fi
sudo systemctl enable --now docker >/dev/null 2>&1 || true

# keep container logs small, SD cards wear out
if [ ! -f /etc/docker/daemon.json ]; then
  echo '{ "log-driver": "local", "log-opts": { "max-size": "10m" } }' | sudo tee /etc/docker/daemon.json >/dev/null
  sudo systemctl restart docker
fi

if ! id -nG "$USER" | grep -qw docker; then
  sudo usermod -aG docker "$USER"
fi

if ! command -v tailscale >/dev/null 2>&1; then
  say "Installing Tailscale"
  curl -fsSL https://tailscale.com/install.sh | sh
fi
if [ -z "$(tailscale ip -4 2>/dev/null || true)" ]; then
  say "Connecting to your tailnet, open the link below to sign in"
  sudo tailscale up
fi

say "Getting neko-rooms into $DIR"
if [ -d "$DIR/.git" ]; then
  git -C "$DIR" pull --ff-only
else
  git clone --depth 1 "$REPO" "$DIR"
fi

say "Starting neko-rooms"
cd "$DIR"
# the docker group only applies to new logins, sg uses it right away
if docker info >/dev/null 2>&1; then
  ./local/up.sh --pi
else
  sg docker -c "./local/up.sh --pi"
fi

cat <<'EOF'

Tips for smooth rooms
  - keep the Pi cool, a fan or a heatsink case avoids throttling
  - use a wired connection when you can
  - inside rooms, play videos at 720p or lower
  - update later by running the same install command again
EOF
