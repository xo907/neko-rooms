# neko-rooms

<p align="center">
  <img src="https://img.shields.io/github/v/release/m1k1o/neko-rooms" alt="release">
  <img src="https://img.shields.io/github/license/m1k1o/neko-rooms" alt="license">
  <img src="https://img.shields.io/docker/pulls/m1k1o/neko-rooms" alt="pulls">
  <img src="https://img.shields.io/github/issues/m1k1o/neko-rooms" alt="issues">
  <a href="https://discord.gg/3U6hWpC" ><img src="https://discordapp.com/api/guilds/665851821906067466/widget.png" alt="Chat on discord"></a>
</p>

Simple room management system for [n.eko](https://github.com/m1k1o/neko). Self hosted rabb.it alternative.

> **This fork** adds a public homepage with a live room directory, user accounts with admin/user roles, friends, room visibility (public / friends / private with invite links) and an admin panel with full branding & theme customization. See [docs/admin-panel.md](./docs/admin-panel.md).
>
> ```bash
> docker pull ghcr.io/xo907/neko-rooms:latest
> ```

<div align="center">
  <img src="https://github.com/m1k1o/neko-rooms/raw/master/docs/rooms.png" alt="rooms">
  <img src="https://github.com/m1k1o/neko-rooms/raw/master/docs/new_room.png" alt="new room">
  <img src="https://github.com/m1k1o/neko-rooms/raw/master/docs/neko.gif" alt="n.eko">
</div>

## Run locally (no domain, share over Tailscale)

```bash
./local/up.sh        # Windows: .\local\up.ps1
```

Builds this checkout, starts it on `http://localhost:8188` and prints the tailnet URLs. See [docs/local.md](./docs/local.md).

## Watch together

- **Now playing**: room cards, link previews and Discord posts show the title of the page open in the room.
- **Reactions, raised hands and an up next queue** inside rooms (web and Android): send emoji that float over everyone's video, raise a hand to get control next, add links everyone can vote on and play the top one.
- **Watch parties**: give a room a start time, people RSVP and add it to their calendar, the room starts on its own.
- **Sleep when empty**: rooms nobody is in stop after 30 minutes (Admin → Settings) and wake up when someone joins.
- **Discord**: paste a channel webhook in Admin → Settings to post when public rooms go live, people start watching, or a party is planned.
- **Room templates**: YouTube, Twitch, music or movie night rooms with a start page (and ad blocking where browser policies are available).
- **Link previews**: share `yoursite/r/room` links and chat apps show the room, its host and what is on.

## Android app

**WatchParty** is a native Android app for any neko-rooms server: rooms list, sign in, friends, profile photos and a native room player with chat, remote control, touch input and keyboard.

Download `WatchParty.apk` from the [latest release](https://github.com/xo907/neko-rooms/releases?q=android-v&expanded=true) on your phone, open it to install, then enter your server's address. Source in [android/](./android), built by GitHub Actions; push a tag like `android-v0.2.0` to publish a new release.

## Run on a VPS (public IP, no domain)

On a server with Docker and git, open it to everyone on its public IP:

```bash
git clone https://github.com/xo907/neko-rooms.git && cd neko-rooms && ./local/up.sh --public
```

It uses the published image, prints the address and the firewall ports to open. Run `./local/up.sh` again later to update. Details: [docs/local.md](./docs/local.md#vps-public-ip).

### With Nginx Proxy Manager (domain + HTTPS)

If the server runs [Nginx Proxy Manager](https://nginxproxymanager.com), use `--npm` instead:

```bash
git clone https://github.com/xo907/neko-rooms.git && cd neko-rooms && ./local/up.sh --npm
```

It joins NPM's docker network, keeps its own port private and prints exactly what to enter in the NPM web UI (proxy host, Websockets, SSL, one Advanced snippet). Step by step: [docs/local.md](./docs/local.md#nginx-proxy-manager-domain--https).

## Run on a Raspberry Pi

On a Raspberry Pi 4 or 5 with a **64-bit Raspberry Pi OS or Debian**, run (as a normal user with sudo, or as root):

```bash
curl -fsSL https://raw.githubusercontent.com/xo907/neko-rooms/master/local/pi-install.sh | bash
```

The script installs Docker and Tailscale if needed (and asks you to sign in to your tailnet), then starts neko-rooms with a lightweight Pi profile and prints the address to open. Run the same command again to update. Details and tuning: [docs/local.md](./docs/local.md#raspberry-pi).

On a minimal Debian without `curl`, run `apt install curl` first.

## Zero-knowledge installation (with HTTPS)

No experience with Docker and reverse proxy? No problem! Follow these steps to set up your Neko Rooms quickly and securely:

- Rent a VPS with public IP and OS Ubuntu.
- Get a domain name pointing to your IP (you can even get some for free).
- Run install script and follow instructions.
- Secure using HTTPs thanks to Let's Encrypt and Traefik or NGINX.

```bash
wget -O neko-rooms-traefik.sh https://raw.githubusercontent.com/xo907/neko-rooms/master/traefik/install
sudo bash neko-rooms-traefik.sh
```

### Community Installation Scripts

We have community-contributed installation scripts available. Check out our [community installation guides](./community/README.md) for instructions on installing neko-rooms on various Linux distributions. These scripts are maintained by the community and support different Linux distributions like Arch Linux, Fedora, and more.

## How to start

If you want to use Traefik as reverse proxy, visit [installation guide for traefik as reverse proxy](./traefik/).

Otherwise modify variables in `docker-compose.yml` and just run `docker-compose up -d`.

### Download images / update

You need to pull all your images, that you want to use with neko-room. Otherwise, you might get this error: `Error response from daemon: No such image:` (see issue #1).

```sh
docker pull ghcr.io/m1k1o/neko/firefox
docker pull ghcr.io/m1k1o/neko/chromium
# etc...
```

If you want to update neko image, you need to pull new image and recreate all rooms, that use old image. To update neko rooms, simply run:

```sh
docker-compose pull
docker-compose up -d
```

### Enable storage

You might have encountered this error:

> Mounts cannot be specified because storage is disabled or unavailable.

If you didn't specify storage yet, you can do it using [this tutorial](./docs/storage.md).

### Use nvidia GPU

If you want to use nvidia GPU, you need to install [nvidia-docker](https://github.com/NVIDIA/nvidia-docker).

Change neko images to nvidia images in `docker-compose.yml` using envorinment variable `NEKO_ROOMS_NEKO_IMAGES`:

```bash
NEKO_ROOMS_NEKO_IMAGES="
  ghcr.io/m1k1o/neko/nvidia-chromium:latest
  ghcr.io/m1k1o/neko/nvidia-google-chrome:latest
  ghcr.io/m1k1o/neko/nvidia-microsoft-edge:latest
  ghcr.io/m1k1o/neko/nvidia-brave:latest
"
```

When creating new room, you need to specify to use GPU in expext settings.

### Docs

For more information visit [docs](./docs).

### Roadmap:
 - [x] add GUI
 - [x] add HTTPS support
 - [x] add authentication provider for traefik
 - [x] allow specifying custom ENV variables
 - [x] allow mounting directories for persistent data
 - [x] optionally remove Traefik as dependency
 - [ ] add upgrade button
 - [ ] auto pull images, that do not exist
 - [ ] add bearer token to for API
 - [ ] add docker SSH / TCP support
 - [ ] add docker swarm support
 - [ ] add k8s support
