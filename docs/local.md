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
3. pulls the default room image (`ghcr.io/m1k1o/neko/firefox`)
4. prints the URLs and, on first start, the one-time setup token for the admin account

Then open `http://localhost:8080` on this machine, or `http://<tailscale-ip>:8080`
/ `http://<machine>.<tailnet>.ts.net:8080` from any device on your tailnet.

Other commands: `./local/up.sh --no-build`, `./local/up.sh logs`, `./local/up.sh down`.

## Settings

Put overrides in `.env.local` (created on first run, ignored by git):

| Variable | Default | |
| --- | --- | --- |
| `NEKO_ROOMS_PORT` | `8080` | port of the web UI on the host |
| `NEKO_ROOMS_EPR` | `59000-59049` | one UDP + TCP port per room, published on the host |
| `NEKO_ROOMS_NAT1TO1` | detected | IP that viewers reach the room streams on |
| `NEKO_ROOMS_ADMIN_USERNAME` | `admin` | first admin account |
| `NEKO_ROOMS_ADMIN_PASSWORD` | empty | empty = setup token in the logs |
| `TZ` | system | time zone |

Users, branding and settings live in the `neko-rooms-local-data` docker volume.

## Notes

- **Video stays black / keeps connecting:** the room ports (`NEKO_ROOMS_EPR`) must be
  reachable on the `NEKO_ROOMS_NAT1TO1` IP. Over Tailscale that works out of the box.
  If you use the LAN IP instead, allow those ports in the host firewall.
- **Docker in WSL on Windows:** run Tailscale inside WSL too (that is the IP the
  script picks up). `localhost:8080` in a Windows browser is forwarded to WSL.
- **Clipboard sync** in rooms needs a secure context (HTTPS or `localhost`). For
  HTTPS on the tailnet, `tailscale serve --bg 8080` gives you
  `https://<machine>.<tailnet>.ts.net` without any other changes.
- Setting `NEKO_ROOMS_INSTANCE_URL` switches back to absolute room links on that URL.
