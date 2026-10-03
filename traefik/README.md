# Installation guide for traefik as reverse proxy

## Zero-knowledge installation

If you don't have any clue about docker and stuff but only want to have fun with friends in a shared browser, we got you covered!

- Rent a VPS with public IP and OS Ubuntu.
- Get a domain name pointing to your IP (you can even get some for free).
- Run install script and follow instructions.

```bash
wget -O neko-rooms.sh https://raw.githubusercontent.com/xo907/neko-rooms/master/traefik/install
sudo bash neko-rooms.sh
```

## How to start

You need to have installed `Docker` and `docker-compose`. You need to have a custom domain pointing to your server's IP.

You can watch installation video provided by *Dr R1ck*:

https://www.youtube.com/watch?v=cCmnw-pq0gA

### Installation guide

You only need `.env.example`, `docker-compose.yml` and `traefik/`.

#### Do I need to use traefik?

- This project started with Traefik as a needed dependency. That, however, changed. Traefik must not be used but the original setup can still be used.
- Traefik is used to forward traffic to the rooms. You can put nginx in front of it.
- See example configuration for [nginx](./nginx).

You can use `docker-compose.http.yml` that will expose this service to `8080` or any port. Authentication is optional. Start it quickly with `docker-compose -f docker-compose.http.yml up -d`.

### Step 1

Copy `.env.example` to `.env` and customize.

```bash
cp .env.example .env
```

### Step 2

Set the first admin account in `.env` (`NEKO_ROOMS_ADMIN_USERNAME` / `NEKO_ROOMS_ADMIN_PASSWORD`). Leave the password empty to get a one-time setup token in the logs instead. Further users are managed in the admin panel, see [docs/admin-panel.md](../docs/admin-panel.md).

`usersfile` is no longer used for neko-rooms itself, but the compose file still mounts it:

```bash
touch usersfile
```

### Step 3 (HTTPS only)

Create `acme.json`

```bash
touch acme.json
chmod 600 acme.json
```

Update your email in `traefik.yml`.
