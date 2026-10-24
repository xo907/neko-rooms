# API

Everything the site does goes through a JSON API under `/api`. Scripts use it with an API token.

## Tokens

Create one under **Account → API tokens**. It is shown once, copy it then. A token can do everything its account can, set an expiry and revoke tokens you no longer use.

```bash
TOKEN=nrk_...
curl -H "Authorization: Bearer $TOKEN" https://rooms.example.com/api/rooms
```

HTTP basic auth with a username and password works too, tokens are better: they can be revoked one by one and never reveal the password.

## Rooms

| | |
|---|---|
| `GET /api/rooms` | rooms you own (all of them for admins) |
| `POST /api/rooms?start=true` | create a room, the body is its settings, `meta` sets title, description, category, visibility and start page |
| `GET /api/rooms/{id}` | one room |
| `POST /api/rooms/{id}/start` · `/stop` · `/restart` · `/pause` | run it or not |
| `POST /api/rooms/{id}/recreate` | recreate it, optionally with new settings |
| `POST /api/rooms/{id}/upgrade` | pull the newest image and recreate the room on it, in the background |
| `GET /api/upgrades/{name}` | how an upgrade is going: `pulling`, `recreating`, `done` (`up_to_date` when there was nothing newer) or `failed` |
| `PUT /api/rooms/{id}/meta` | title, description, category, visibility, `scheduled_at` (watch party), `start_url`, `keep_awake` |
| `DELETE /api/rooms/{id}` | remove it |

Creating a room on an image the server does not have yet downloads it first. When that takes longer than about 40 seconds the answer is `503` with `Retry-After` and `X-Neko-Rooms-Pulling: <image>`: ask again in a few seconds.

```bash
curl -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -X POST "https://rooms.example.com/api/rooms?start=true" \
  -d '{"name": "movie-night", "neko_image": "ghcr.io/m1k1o/neko/firefox:3.1.6",
       "meta": {"title": "Movie night", "visibility": "public", "start_url": "https://www.youtube.com"}}'
```

## Public

These work without signing in, for what a visitor may see:

| | |
|---|---|
| `GET /api/public/rooms` | the room directory |
| `GET /api/public/rooms/{name}` | one room, `?invite=` for invite links |
| `POST /api/public/rooms/{name}/join` | the room address with credentials (`202 {"waking": true}` while a sleeping room starts) |
| `GET /api/public/rooms/{name}/event.ics` | a watch party as a calendar event |
| `GET /api/public/og/rooms/{name}.png` | the link preview image of a room |

### Rewind

For people in a room, with the room ticket from the join address (`nrt`) as `?t=`:

| | |
|---|---|
| `GET /api/public/live/{name}/rewind` | what can be played back: `segments` with `n`, `start` and `end` in ms on the recording timeline, `now` on that timeline, `epoch` its start in unix ms |
| `GET /api/public/live/{name}/rewind/{n}` | segment `n`, a WebM (VP8, Opus) with timestamps on the recording timeline, about 5 seconds long |

Segments are made to be appended one after another to a `MediaSource` source buffer in `segments` mode.

### Subtitles

| | |
|---|---|
| `GET /api/public/live/{name}/captions?t=` | the languages viewers can pick, `orig` is the language spoken; empty when subtitles are off |
| `POST /api/public/live/{name}/captions` | `{"t": ticket, "lang": "es"}` picks a language, `""` turns them off |

Lines arrive on the live channel (`GET /api/public/live/{name}/events`) as `{"type": "caption", "lang": "es", "text": "…"}`, only in the language picked. Pick again after reconnecting.
