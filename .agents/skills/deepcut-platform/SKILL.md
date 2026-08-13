---
name: deepcut-platform
description: DeepCut Live project-specific integration notes — SRS media server, River job queue, hexagonal module layout, HLS/recording paths, and platform env vars. Load when working on streaming, recording, VOD, chat, or infrastructure code in THIS repo.
---

# DeepCut Live — Platform Integration Notes

Project-specific knowledge for this repository only. Generic Go and
Next.js standards live in the `go-chi` and `nextjs` skills; this file
holds what is true about DeepCut's specific infrastructure.

## Backend Layout (hexagonal modules)

This backend uses hexagonal modules — NOT a flat handler/service/store
tree:

```
backend/internal/
├── modules/
│   ├── auth/       # Google OAuth, JWT (ECDSA), stream keys
│   ├── streams/    # live streams, SRS callbacks, poller, viewers, analytics
│   ├── chat/       # WebSocket chat, rate limiting, history
│   └── vods/       # VOD listing/search, River queue, ffmpeg worker
│   └── <module>/
│       ├── domain/       # entities, typed status constants, ports
│       ├── application/  # services (business logic)
│       └── adapter/
│           ├── http/     # chi handlers
│           ├── postgres/ # pgx repositories
│           └── river/    # queue adapter (vods only)
├── shared/{errs,render}/ # error kinds + JSON rendering
└── testutil/             # testcontainers DB helpers
```

Modules do not import each other's domain packages — each defines its
own ports (e.g. `streams/domain.AuthRepo`) and duplicated status
constants with matching values. Entry points: `cmd/server` (API) and
`cmd/worker` (River VOD processor).

## SRS Integration (ossrs/srs:5)

Traps that cost days when working with SRS in Docker:

- **The image loads `conf/docker.conf`, NOT `conf/srs.conf`.** The
  startup log line says exactly which file it read
  (`SRS on aarch64, conf:conf/docker.conf`). Mount custom config at
  `/usr/local/srs/conf/docker.conf` or your tuning silently never
  applies (10s fragments, no callbacks).
- **LL-HLS was removed in SRS 5.** `hls_ll_enabled` / `hls_ll_fragment`
  fail config validation. Use short full segments (`hls_fragment 2`).
- **`http_hooks` `client_id` is a string connection id** (`"5u9c4d30"`),
  not the numeric HTTP-API client id. Stored as TEXT
  (`streams.srs_client_id`).
- **`hls_ctx` is on by default** and wraps every playlist in a master
  playlist with a root-absolute child URI. Behind the Next.js proxy
  that strips `/hls/*`, players lose the prefix → 404. Set
  `hls_ctx off` unless per-session HLS auth is needed.
- **SRS does not send duration in `on_unpublish`** — the backend
  computes it from `started_at`.
- **Callbacks may not fire at all** in some Docker configurations —
  `StartSRSPoller` polls `GET /api/v1/clients/` (port 1985) as a
  fallback. Any side effect added to the callback path (`OnStreamStart`)
  MUST also be added to the poller path (`pollSRS`), and both end paths
  funnel through `stopStreamSideEffects`.
- **Verify file paths in the actual container.** Config directives like
  `hls_path /data/hls` may be overridden by image defaults. Run
  `docker compose exec srs find / -name "*.m3u8"` to see where files
  actually land.

### Media paths & URLs

| What | Where |
|---|---|
| Live HLS playlist (public URL) | `/hls/live/<streamKey>.m3u8` (Next proxy → SRS :8080) |
| Live thumbnails (written by backend ffmpeg) | `/data/hls/thumbnails/live/<streamID>.jpg` |
| Recordings (MPEG-TS, written by backend ffmpeg) | `/data/recordings/<streamID>.ts` |
| VOD HLS output (written by worker) | `/data/hls/vods/<streamID>/index.m3u8` |
| SRS HTTP API | `SRS_API_URL` (default `http://srs:1985`) |
| SRS HLS server | derived: API host on port 8080 (`srsHTTPURL`) |

⚠️ Known issue (documented, not yet fixed): the live HLS URL embeds the
raw stream key, which is secret. Do not extend this pattern; a remap to
opaque stream IDs is the intended fix.

## River (PostgreSQL job queue)

River has sharp validation edges that produce confusing failures:

- **Insert-only clients must register the job kind.** `Insert` fails
  with "job kind is not registered" unless a worker exists — even if the
  client never runs workers. The server registers a `noopWorker`.
- **`Queues` config requires `Workers` set and `MaxWorkers >= 1`** —
  both must be non-empty or `NewClient` errors.
- **`client.Start()` is non-blocking** — block on a signal/ctx
  afterwards or the worker process exits immediately.
- **Run `rivermigrate.Migrate()` before `NewClient`**, idempotently:
  if migration errors but `river_queue` already exists (partial apply
  from a crash), verify the schema state and continue
  (see `cmd/worker/main.go migrateRiverSchema`).

## Environment Variables

| Var | Default | Used by |
|---|---|---|
| `PORT` | 8081 | server |
| `DATABASE_URL` | localhost postgres `live/live` | server, worker |
| `BASE_URL` | http://localhost:3000 | OAuth redirects |
| `CORS_ORIGIN` | http://localhost:3000 | REST CORS |
| `SRS_CALLBACK_SECRET` | dev-srs-secret (warns) | SRS webhook auth |
| `SRS_API_URL` | http://srs:1985 | poller, disconnect, ffmpeg source URLs |
| `JWT_PRIVATE_KEY` / `JWT_PUBLIC_KEY` | generated per boot (warns) | auth |
| `VOD_WORKER_MAX_WORKERS` | 1 | worker |
| `VOD_ENCODING_PRESET` | copy (`720p`, `480p`) | worker transcode |
| `BACKEND_URL` | http://localhost:8081 | worker → server VOD status notify |
| `NEXT_PUBLIC_API_URL` | http://localhost:3000 | frontend API base (via proxy) |
| `NEXT_PUBLIC_WS_URL` | http://localhost:8081 | frontend WS base (direct) |
| `HLS_URL` | http://localhost:8080 | Next.js /hls rewrites |

## Frontend proxy quirks

- `frontend/src/proxy.ts` is Next 16 middleware (naming convention).
- Next.js rewrites do NOT proxy WebSocket upgrades — WS connects
  directly to the backend (`NEXT_PUBLIC_WS_URL`). Cookies still flow on
  localhost because SameSite=Lax treats ports as same-site.
- REST goes through the same-origin proxy (`/api/*` → backend), so the
  sign-in URL and API base use `NEXT_PUBLIC_API_URL` (port 3000).

## Known deferred issues (do not "fix" casually — need product decisions)

- Stream key exposed in public HLS URLs (security; needs opaque remap).
- VOD viewer heartbeats increment counts every tick (needs client dedup
  schema like live `stream_viewers`).
- `stream_analytics` peak/unique are written from `streams` columns that
  nothing updates (always 0; needs an aggregation step).
- VOD chat replay UI exists but never loads history (`getChatMessages`
  unused; `ChatPanel isVodReplay` has no HTTP branch).
- Search `category` param accepted but unimplemented in SQL.
- WS `OriginPatterns` hardcode localhost (config threading needed for
  production).

*Last updated: 2026-08-12*
