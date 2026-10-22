# Run locally, share over Tailscale

No domain, reverse proxy or TLS needed. neko-rooms serves the site and proxies
the rooms itself, and room links are relative, so the site works on whatever
address you open it on: `localhost`, a LAN IP, your tailscale IP or MagicDNS name.

## Requirements

- Docker with the compose plugin (on Windows: Docker inside WSL or Docker Desktop)
- [Tailscale](https://tailscale.com/download) on the machine running Docker, and on
  every device that should join

## Start

```bash
./local/up.sh
```

On Windows, from PowerShell (runs the same script inside WSL):

```powershell
.\local\up.ps1                 # default WSL distro
.\local\up.ps1 -Distro Debian  # a specific one
```

The script:

1. detects your tailscale IP (LAN IP as a fallback) and writes it to `.env.local`
   as `NEKO_ROOMS_NAT1TO1`, the address browsers use for the room video (WebRTC)
2. builds the image from this checkout and starts it with `docker-compose.local.yml`
3. pulls the default room image (`ghcr.io/m1k1o/neko/firefox:3.1.6`)
4. prints the URLs and, on first start, the one-time setup token for the admin account

Then open `http://localhost:8188` on this machine, or `http://<tailscale-ip>:8188`
/ `http://<machine>.<tailnet>.ts.net:8188` from any device on your tailnet
(the script prints the exact addresses).

Other commands: `./local/up.sh --no-build`, `./local/up.sh logs`, `./local/up.sh down`.

## Settings

Put overrides in `.env.local` (created on first run, ignored by git):

| Variable | Default | |
| --- | --- | --- |
| `NEKO_ROOMS_PORT` | `8188` | port of the web UI on the host, the next free one is picked if 8188 is taken |
| `NEKO_ROOMS_EPR` | `59000-59049` | one UDP + TCP port per room, published on the host |
| `NEKO_ROOMS_NAT1TO1` | detected | IP that viewers reach the room streams on |
| `NEKO_ROOMS_ADMIN_USERNAME` | `admin` | first admin account |
| `NEKO_ROOMS_ADMIN_PASSWORD` | empty | empty = setup token in the logs |
| `TZ` | system | time zone |

Users, branding and settings live in the `neko-rooms-local-data` docker volume.

## VPS (public IP)

```bash
git clone https://github.com/xo907/neko-rooms.git && cd neko-rooms && ./local/up.sh --public
```

`--public` makes the site and the room video reachable on the server's public
IP and uses the published image instead of building. It is remembered in
`.env.local`, so `./local/up.sh` alone keeps it (`--no-public` to go back).

- open the ports it prints: TCP for the website (8188) and TCP + UDP for the
  room video (59000-59049, one per room), in the server firewall and in the
  VPS provider's firewall if it has one
- if the public IP is not on the server itself (some clouds use NAT), set it:
  `echo NEKO_ROOMS_PUBLIC_IP=<public ip> >> .env.local`
- every room runs a full browser, plan for about 1.5 GB of RAM and one or two
  CPU cores per active room

Plain HTTP sends passwords unencrypted, for HTTPS use Nginx Proxy Manager below.

## Nginx Proxy Manager (domain + HTTPS)

On a server that already runs [Nginx Proxy Manager](https://nginxproxymanager.com)
(NPM), with a domain or subdomain pointing to the server:

```bash
git clone https://github.com/xo907/neko-rooms.git && cd neko-rooms && ./local/up.sh --npm
```

`--npm` finds the NPM container, joins its docker network and keeps the
neko-rooms port private (only NPM can reach it). It trusts the proxy headers,
so sign-in cookies are secure and rate limits see real visitor IPs. The room
video still goes straight to the public IP. It is remembered in `.env.local`
(`--no-npm` to undo).

Then add the proxy host in the NPM web UI (usually `http://<server ip>:81`),
the script prints these values for your server:

1. **Hosts > Proxy Hosts > Add Proxy Host**
2. **Details:** Domain Names `rooms.example.com`, Scheme `http`,
   Forward Hostname / IP `neko-rooms-local`, Forward Port `8080`,
   turn on **Block Common Exploits** and **Websockets Support**
3. **SSL:** Request a new SSL Certificate, turn on **Force SSL** and **HTTP/2 Support**
4. **Advanced**, Custom Nginx Configuration:
   ```nginx
   proxy_read_timeout 1h;
   proxy_send_timeout 1h;
   proxy_buffering off;
   ```
5. Save and open `https://rooms.example.com`.

Open TCP and UDP 59000-59049 in the firewall for the room video. Ports 80/443
are already NPM's, the neko-rooms port stays closed.

If NPM runs on the default docker bridge or the host network, the script prints
an IP to forward to instead of `neko-rooms-local`. If it can not find NPM, set
its container name: `echo NPM_CONTAINER=<name> >> .env.local`.

## Raspberry Pi

One command on a 64-bit Raspberry Pi OS or Debian, Pi 4 or 5. Run it as a normal
user with sudo, or as root (plain Debian often has no sudo). On a minimal
Debian without curl, `apt install curl` first:

```bash
curl -fsSL https://raw.githubusercontent.com/xo907/neko-rooms/master/local/pi-install.sh | bash
```

It installs Docker and Tailscale when missing, clones this repository to
`~/neko-rooms` and runs `./local/up.sh --pi`. On a Pi, `up.sh` picks the Pi
profile ([docker-compose.pi.yml](../docker-compose.pi.yml)) by itself:

- pulls the published multi-arch image instead of building on the Pi
- offers only Firefox and Chromium rooms (the browsers with arm64 images)
- starts new rooms at **1024x576, 30 fps**, so the Pi can encode the stream
  while the room's browser is playing video
- allows up to 5 rooms and keeps logs small to spare the SD card

Every room runs a full browser and the Pi encodes its video in software, so
plan for **one active room** on a Pi 4. For smooth playback:

- keep the Pi cool (fan or heatsink case), it slows down when hot
- use Ethernet for the Pi if you can
- inside rooms, play videos at 720p or lower
- want it sharper? add `NEKO_PI_SCREEN=1280x720@30` to `~/neko-rooms/.env.local`
  and run `./local/up.sh` again (new rooms only; a busy Pi 4 may stutter)

Update: run the install command again, or `cd ~/neko-rooms && git pull && ./local/up.sh`.

The room size can be set on any install with `NEKO_ROOMS_ROOM_DEFAULTS_SCREEN`
(also `NEKO_ROOMS_ROOM_DEFAULTS_VIDEO_CODEC` and `..._VIDEO_PIPELINE`); it
pre-fills new rooms, existing rooms keep their settings.

## Notes

- **Video stays black / keeps connecting:** the room ports (`NEKO_ROOMS_EPR`) must be
  reachable on the `NEKO_ROOMS_NAT1TO1` IP. Over Tailscale that works out of the box.
  If you use the LAN IP instead, allow those ports in the host firewall.
- **Docker in WSL on Windows:** run Tailscale inside WSL too (that is the IP the
  script picks up). `localhost:8188` in a Windows browser is forwarded to WSL.
- **Clipboard sync** in rooms needs a secure context (HTTPS or `localhost`). For
  HTTPS on the tailnet, `tailscale serve --bg 8188` gives you
  `https://<machine>.<tailnet>.ts.net` without any other changes.
- Setting `NEKO_ROOMS_INSTANCE_URL` switches back to absolute room links on that URL.
