# VOD Recording & Processing Pipeline

**Status:** Review  
**Owner:** Eliseo  
**Created:** 2026-08-11

## Requirements

### User Story 1: Stream Recording (US1)

As a streamer, I want my live streams to be automatically recorded to
disk so that they become available as VODs after the stream ends.

**Acceptance Criteria:**

- Given I start streaming via RTMP, When SRS receives the stream, Then
  SRS records the stream to `/data/recordings/{streamId}.mp4` (separate
  from the live HLS segments at `/data/hls/live/`)
- Given my stream is recording, When SRS writes segments, Then the
  recording accumulates at `/data/recordings/` as a single MP4 file
- Given my stream ends (`on_unpublish`), When SRS calls the backend
  callback, Then the recording file is complete and its path is passed
  to the backend via `recording_path` query parameter
- Given the docker-compose setup, When services start, Then a
  `/data/recordings` volume is mounted to both SRS and the backend

### User Story 1b: Live Stream Thumbnail (US1b)

As a viewer browsing live streams, I want to see a thumbnail image on
each live stream card so that I can preview the content before clicking.

**Acceptance Criteria:**

- Given a stream is live, When the backend detects the stream is active,
  Then a background goroutine captures a frame from the HLS stream every
  10 seconds and saves it to `/data/hls/thumbnails/live/{streamId}.jpg`
- Given the thumbnail is saved, When SRS serves the file, Then it is
  accessible at `/hls/thumbnails/live/{streamId}.jpg` through the Next.js
  proxy
- Given the thumbnail capture fails (e.g., stream not yet producing
  segments), When the goroutine retries on the next interval, Then the
  failure is logged but does not affect the stream
- Given the stream ends, When the `on_unpublish` callback fires, Then
  the thumbnail goroutine is stopped and the live thumbnail file is
  deleted (replaced by the VOD thumbnail after processing)
- Given the live stream thumbnail exists, When the frontend renders a
  `LiveStreamCard`, Then it shows the thumbnail image at
  `/hls/thumbnails/live/{streamId}.jpg` instead of the 🎬 placeholder
- Given the thumbnail URL is requested before the first capture, When
  the browser loads it, Then it shows a 404 (frontend falls back to
  🎬 placeholder via `onError`)

### User Story 2: Enqueue Processing on Stream End (US2)

As the backend, I want to enqueue a VOD processing job when a stream
ends so that the recording is transcoded and the VOD becomes watchable.

**Acceptance Criteria:**

- Given a stream ends (`on_unpublish`), When the backend receives the
  callback, Then it sets `recording_status = 'processing'` and enqueues
  a job with `{ streamId, recordingPath }` into the PostgreSQL-backed
  job queue (River)
- Given the job is enqueued, When the worker picks it up, Then it
  begins processing the MP4 recording
- Given the job enqueue fails, When the backend can't insert the job,
  Then it logs the error and sets `recording_status = 'failed'` with
  an error message
- Given a stream ends with no recording path (DVR disabled or misconfig),
  When the backend receives the callback, Then it sets
  `recording_status = 'failed'` with message "No recording available"

### User Story 3: VOD Processing Worker (US3)

As the processing worker, I want to transcode the MP4 recording into
HLS segments and generate a thumbnail so that the VOD is playable in
the browser.

**Acceptance Criteria:**

- Given a processing job, When the worker starts, Then it:
  a. Creates `/data/hls/vods/{vodId}/` directory
  b. Runs ffmpeg to convert MP4 → HLS segments (`.m3u8` + `.ts`)
  c. Runs ffmpeg to extract a thumbnail at 10% into the video → `/data/thumbnails/{vodId}.jpg`
  d. Updates the stream record: `recording_status = 'ready'`,
     `hls_path = '/hls/vods/{vodId}/index.m3u8'`
- Given ffmpeg fails, When the worker catches the error, Then it sets
  `recording_status = 'failed'` with an error message and retries up
  to 3 times
- Given processing succeeds, When the status is set to `'ready'`, Then
  the VOD appears in search results and the VOD watch page shows the
  player (no longer "Processing — available soon")
- Given the worker processes a job, When the `recording_status`
  transitions from `'processing'` to `'ready'` or `'failed'`, Then a
  WebSocket event is emitted to notify connected frontends

### User Story 4: WebSocket VOD Status Notification (US4)

As the frontend, I want to receive real-time WebSocket events when a VOD
finishes processing so that the UI transitions from "Processing" to the
video player without a page refresh.

**Acceptance Criteria:**

- Given a viewer is on the `/vods/{id}` page for a VOD with status
  `'processing'`, When the worker finishes processing and emits a
  WebSocket event, Then the frontend automatically transitions from the
  "Processing" state to the video player
- Given the WebSocket event is `vod_status` with payload
  `{ vodId, status, hlsUrl }`, When the frontend receives it, Then it
  updates the UI for the matching VOD
- Given the viewer is NOT on the VOD page, When the event fires, Then
  it is silently ignored (no UI to update) [NEEDS CLARIFICATION: should
  we also refresh search results or the homepage if they're showing
  VODs?]
- Given the WebSocket connection drops, When the viewer returns to the
  `/vods/{id}` page, Then the page fetches the current status from the
  API (server-rendered, no reliance on WebSocket for initial state)

### User Story 5: Frontend Real-Time Status Update (US5)

As a viewer watching a VOD page, I want the "Processing" message to
transition to the video player automatically when processing completes.

**Acceptance Criteria:**

- Given I am on `/vods/{id}` for a VOD with `recordingStatus: 'processing'`,
  When the WebSocket `vod_status` event fires with `status: 'ready'`,
  Then the page transitions from the processing state to the video player
  without a full page reload
- Given I am on `/vods/{id}` for a VOD with `recordingStatus: 'processing'`,
  When the WebSocket `vod_status` event fires with `status: 'failed'`,
  Then the page transitions to the "This recording is unavailable" state
- Given the page initially loads with `recordingStatus: 'ready'`, When
  the page renders, Then the video player is shown immediately (no
  WebSocket dependency for the happy path)

## Non-Goals

- ❌ **Adaptive bitrate transcoding** — single quality HLS output for v1
  (per main spec non-goal)
- ❌ **Clip/highlight creation** — no clip extraction from VODs
- ❌ **Thumbnail customization** — always auto-generated at 10% position
- ❌ **VOD deletion/retention policies** — VODs kept forever (per main spec)
- ❌ **Multi-region CDN for VODs** — single origin for v1
- ❌ **Resumable processing** — if the worker crashes mid-job, the job
  retries from scratch
- ❌ **Priority queue** — all VODs processed FIFO; no streamer-based
  priority
- ❌ **Admin dashboard for queue management** — no retry/failure UI
  beyond database inspection
- ❌ **Multiple worker instances** — single worker process for v1

## Open Questions

- ✅ US4: Real-time card updates on both the VOD watch page AND
  homepage/search cards. When `vod_status` event fires, all visible
  VodCards and the VOD watch page update.
- ✅ US3: ffmpeg encoding configurable via `VOD_ENCODING_PRESET` env var
  (default: `copy` = source quality passthrough, `-c:v copy -c:a copy`).
  Options: `copy`, `720p` (scale to 720p @ 2Mbps), `480p` (480p @ 1Mbps).
- ✅ US3: Thumbnail served via SRS static files. Worker writes to
  `/data/hls/thumbnails/{vodId}.jpg`. Generated during post-processing.
- ✅ US2: Async enqueue — the backend inserts the job into the PostgreSQL
  queue (River) and returns immediately. A separate `cmd/worker` process
  polls the queue and processes jobs. No goroutines in the main server.

## Design

### Architecture

#### 1. System Overview

```mermaid
flowchart TD
    OBS[OBS Studio] -->|RTMP :1935| SRS

    SRS -->|HLS live| HLS_LIVE[/data/hls/live/]
    SRS -->|DVR record| RECORDINGS[/data/recordings/]
    SRS -->|on_publish| BACKEND[Backend :8081]
    SRS -->|on_unpublish| BACKEND

    BACKEND -->|INSERT job| RIVER[(PostgreSQL River queue)]
    BACKEND -->|goroutine| LIVE_THUMB[Live thumbnail every 10s]
    LIVE_THUMB -->|ffmpeg capture| THUMBS[/data/hls/thumbnails/live/]

    RIVER -->|poll| WORKER[cmd/worker]
    WORKER -->|ffmpeg transcode| VOD_HLS[/data/hls/vods/]
    WORKER -->|ffmpeg thumbnail| VOD_THUMBS[/data/hls/thumbnails/]
    WORKER -->|UPDATE status| DB[(PostgreSQL)]
    WORKER -->|WebSocket notify| BACKEND

    BACKEND -->|vod_status WS event| FRONTEND[Next.js Frontend]
```

#### 2. Technology Decisions

**Decision: River (github.com/riverqueue/river) for job queue.**
- **Why:** Go-native, PostgreSQL-backed — zero new infrastructure. Jobs
  survive restarts, support retries, and are visible via SQL queries.
- **Rejected:** Redis — adds a new container, requires persistence config.
- **Rejected:** RabbitMQ/NATS — overkill for a single-worker pipeline.
- **Rejected:** Raw goroutines — jobs lost on restart, no retry mechanism.

**Decision: Separate `cmd/worker` binary for async processing.**
- **Why:** Clean separation — the main server handles HTTP/WebSocket, the
  worker handles ffmpeg. The worker can be scaled independently. Crash or
  restart of the worker doesn't affect live streaming.
- **Rejected:** Goroutine in main server — couples ffmpeg lifecycle to HTTP
  server; ffmpeg crashes could bring down the whole server.

**Decision: ffmpeg for all media processing.**
- **Why:** Industry standard, already available, handles MP4→HLS, thumbnail
  extraction, and encoding in a single tool. Env-configurable presets.
- **Rejected:** GStreamer — more complex API, larger dependency footprint.
- **Rejected:** Cloud transcoding service — requires network egress, adds
  cost, not suitable for self-hosted v1.

**Decision: Live thumbnail via goroutine in main server, not the worker.**
- **Why:** Live thumbnails must run during the stream, before the worker
  sees the job. The goroutine starts on `on_publish` and stops on
  `on_unpublish`. Uses ffmpeg to capture a frame from the HLS stream
  (no RTMP SDK needed).
- **Rejected:** Worker polling — adds latency; the worker only wakes up
  for queued jobs.

#### 3. SRS Configuration Changes

Add DVR block to `data/srs.conf`:

```nginx
vhost __defaultVhost__ {
    hls { ... }  # unchanged

    dvr {
        enabled         on;
        dvr_path        /data/recordings/[stream].[timestamp].mp4;
        dvr_plan        session;
        dvr_duration    0;  # unlimited, stops on unpublish
    }

    http_hooks { ... }  # unchanged
}
```

**SRS passes `recording_path` in the `on_unpublish` callback** as a
query parameter, pointing to the completed MP4 file.

#### 4. Data Model Changes

**New columns on `streams` table:**

```sql
-- Already exists (DB default):
-- recording_status TEXT DEFAULT 'pending'

-- Add column for the VOD HLS playlist path (set by worker after processing):
ALTER TABLE streams ADD COLUMN vod_hls_path TEXT;

-- Add column for the permanent VOD thumbnail (set by worker after processing):
ALTER TABLE streams ADD COLUMN vod_thumbnail_path TEXT;

-- Add column for error message when processing fails:
ALTER TABLE streams ADD COLUMN recording_error TEXT;
```

**River migration table** — created automatically by River on first run.
No manual migration needed.

**recording_status state machine:**

```
pending ──▶ processing ──▶ ready
                │
                └──▶ failed (with recording_error set)
```

All transitions happen in the backend (enqueue) or worker (processing complete).

#### 5. API Contracts

##### 5.1 WebSocket: `vod_status` event (NEW)

Emitted by the worker after updating `recording_status`. The main server
receives this via a shared notification channel and broadcasts to
connected frontends.

**Event shape (server → client):**

```json
{
  "type": "vod_status",
  "vodId": "uuid",
  "status": "ready | failed",
  "hlsUrl": "/hls/vods/{vodId}/index.m3u8",
  "thumbnailUrl": "/hls/thumbnails/{vodId}.jpg",
  "error": "optional error message if failed"
}
```

**Delivery:** Broadcast to all connected WebSocket clients. Frontend
filters by `vodId` to update only relevant components.

##### 5.2 `GET /api/vods/{vodID}` — Response Change

Adds new fields to the existing VOD response:

```json
{
  "id": "uuid",
  "userId": "uuid",
  "userName": "string",
  "title": "string | null",
  "startedAt": "ISO8601",
  "endedAt": "ISO8601 | null",
  "durationSeconds": "int | null",
  "peakViewers": "int",
  "totalViewers": "int",
  "recordingPath": "string | null",    // raw MP4 path (existing)
  "recordingStatus": "pending | processing | ready | failed",
  "hlsUrl": "/hls/vods/{id}/index.m3u8 | null",  // NEW: derived from vod_hls_path
  "thumbnailUrl": "/hls/thumbnails/{id}.jpg | null",  // NEW
  "recordingError": "string | null",  // NEW
  "createdAt": "ISO8601"
}
```

**Note:** `hlsUrl` and `thumbnailUrl` are `null` when `recordingStatus`
is `'pending'` or `'processing'`. They are populated by the worker when
status transitions to `'ready'`.

##### 5.3 `GET /api/streams/live` — Response Change

Adds `thumbnailUrl` to each live stream:

```json
{
  "streams": [{
    "userId": "uuid",
    "streamerName": "string",
    "thumbnailUrl": "/hls/thumbnails/live/{streamId}.jpg | null",  // NEW
    ...
  }],
  "total": "int"
}
```

**Note:** `thumbnailUrl` is `null` for the first ~10 seconds of a stream
(before the first capture). Frontend falls back to 🎬 placeholder via
`onError` on the `<img>` tag.

#### 6. Frontend Type Changes

```typescript
// Updated VodDetail
export interface VodDetail {
  // ... existing fields ...
  recordingStatus: "pending" | "processing" | "ready" | "failed";
  hlsUrl: string | null;       // was derived, now from backend
  thumbnailUrl: string | null; // NEW
  recordingError: string | null; // NEW
}

// Updated LiveStream
export interface LiveStream {
  // ... existing fields ...
  thumbnailUrl: string | null; // already exists, now populated for live streams
}
```

#### 7. Docker Changes

```yaml
# docker-compose.yml additions
services:
  backend:
    volumes:
      - ./data/hls:/data/hls
      - ./data/recordings:/data/recordings  # NEW

  srs:
    volumes:
      - ./data/srs.conf:/usr/local/srs/conf/srs.conf
      - ./data/hls:/data/hls
      - ./data/recordings:/data/recordings  # NEW

  worker:  # NEW service
    build:
      context: ./backend
      dockerfile: Dockerfile.worker
    environment:
      DATABASE_URL: postgres://live:live@postgres:5432/live?sslmode=disable
      VOD_ENCODING_PRESET: copy
    volumes:
      - ./data/hls:/data/hls
      - ./data/recordings:/data/recordings
    depends_on:
      postgres:
        condition: service_healthy
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

#### 8. Architecture Non-Goals

- ❌ No REST endpoints for queue management (inspect via DB)
- ❌ No worker scaling — single worker instance for v1
- ❌ No SRS DVR segmentation — single MP4 per stream
- ❌ No thumbnail API — served as static files by SRS
- ❌ No recording retention policy — kept forever
- ❌ No auth on WebSocket `vod_status` — public event, no sensitive data

## Task Checklist

> **Role tags:** (Backend), (Worker), (Frontend), (Infra)
> **[P]** = can run in parallel with other [P] tasks.

### Phase 1 — Infrastructure (blockers for everything)

1. [x] (Infra) **Add SRS DVR configuration**
   - Files: `data/srs.conf`
   - Add `dvr {}` block under `vhost __defaultVhost__`: record to
     `/data/recordings/[stream].[timestamp].mp4` with `dvr_plan session`
   - Verifies: SRS `on_unpublish` callback includes `recording_path` param
   - Satisfies: US1 AC1, AC2

2. [x] (Infra) **Add DB migration for new columns**
   - Files: new migration SQL files in `backend/db/migrations/`
   - Add `vod_hls_path TEXT`, `vod_thumbnail_path TEXT`,
     `recording_error TEXT` to `streams` table
   - Satisfies: data model

3. [x] (Infra) **Add River dependency + worker Docker setup**
   - Files: `backend/go.mod`, `Dockerfile.worker`, `docker-compose.yml`
   - Add `github.com/riverqueue/river` to `go.mod`
   - Create `Dockerfile.worker` (Go binary + ffmpeg)
   - Add `worker` service to `docker-compose.yml` with `/data/hls` and
     `/data/recordings` volumes
   - Add `/data/recordings` volume to `backend` and `srs` services
   - Satisfies: US2, US3 infrastructure

### Phase 2 — Backend (can partially parallelize)

4. [x] (Backend) **Define VOD queue interface + River adapter**
   - Files: `backend/internal/modules/vods/domain/queue.go`,
     `backend/internal/modules/vods/adapter/river/`
   - Interface: `VODQueue { Enqueue(ctx, streamID, recordingPath string) error }`
   - River adapter: creates River client, inserts `VODProcessArgs` job
   - Tests: mock queue for handler/service tests
   - Satisfies: US2 AC1

5. [x] (Backend) **Update `OnStreamEnd` to set status + enqueue**
   - Files: `backend/internal/modules/streams/domain/repository.go`,
     `backend/internal/modules/streams/adapter/postgres/repo.go`,
     `backend/internal/modules/streams/application/service.go`
   - Add `UpdateRecordingStatus(ctx, streamID, status, errorMsg string) error`
     to repo
   - In `OnStreamEnd`: set `recording_status = 'processing'` if
     `recordingPath != ""`, else set `'failed'` with error message
   - Inject `VODQueue` into `StreamService`, enqueue job if status set to
     `'processing'`
   - Satisfies: US2 AC1, AC2, AC3, AC4

6. [ ] [P] (Backend) **Add WebSocket `vod_status` broadcast**
   - Files: `backend/internal/modules/streams/application/hub.go`
   - Add `NotifyVODStatus(vodID, status, hlsURL, thumbnailURL, error string)`
     to `StreamHub`
   - Expose via shared channel for worker to call post-processing
   - Tests: verify broadcast reaches connected clients
   - Satisfies: US4 AC1, AC2

7. [ ] [P] (Backend) **Add live thumbnail goroutine**
   - Files: `backend/internal/modules/streams/application/service.go`
   - On `OnStreamStart`: spawn goroutine that runs ffmpeg every 10s:
     captures frame from HLS stream to
     `/data/hls/thumbnails/live/{streamId}.jpg`
   - On `OnStreamEnd`: signal goroutine to stop; delete live thumbnail
   - Store active goroutine cancel funcs in a `sync.Map` keyed by stream ID
   - Tests: mock ffmpeg, verify goroutine lifecycle
   - Satisfies: US1b AC1, AC2, AC3, AC4

8. [x] (Backend) **Update API responses with new fields**
   - Files: `backend/internal/modules/vods/adapter/http/handler.go`,
     `backend/internal/modules/streams/adapter/http/handler.go`
   - `GET /api/vods/{id}`: add `hlsUrl`, `thumbnailUrl`, `recordingError`
     (derive from new columns)
   - `GET /api/streams/live`: add `thumbnailUrl` pointing to
     `/hls/thumbnails/live/{streamId}.jpg`
   - Satisfies: API contract

### Phase 3 — Worker

9. [x] (Worker) **Create `cmd/worker` binary**
   - Files: `backend/cmd/worker/main.go`
   - Initialize River client (connects to same PostgreSQL)
   - Register `VODProcessWorker`
   - Start worker loop
   - Graceful shutdown on SIGTERM
   - Satisfies: US3 infrastructure

10. [x] (Worker) **Implement VOD processing job**
    - Files: `backend/internal/modules/vods/application/worker.go`
    - River job handler:
      a. Read `VODProcessArgs { StreamID, RecordingPath }`
      b. Create `/data/hls/vods/{streamId}/` directory
      c. Run ffmpeg MP4 to HLS with env-configurable encoding
      d. Run ffmpeg thumbnail extraction at 10% position
      e. Update stream: `vod_hls_path`, `vod_thumbnail_path`,
         `recording_status = 'ready'`
      f. Emit `vod_status` WebSocket event via shared channel
    - On ffmpeg failure: set `recording_status = 'failed'`,
      `recording_error`, retry up to 3 times
    - Tests: mock ffmpeg, verify status transitions, verify retry logic
    - Satisfies: US3 AC1, AC2, AC3, AC4

### Phase 4 — Frontend

11. [ ] [P] (Frontend) **Update types for new response fields**
    - Files: `frontend/src/types/index.ts`
    - Add `hlsUrl: string | null`, `thumbnailUrl: string | null`,
      `recordingError: string | null` to `VodDetail`
    - `LiveStream.thumbnailUrl` already exists, now populated by backend
    - Satisfies: frontend type alignment

12. [ ] [P] (Frontend) **VodView: listen for `vod_status` WebSocket events**
    - Files: `frontend/src/app/vods/[id]/VodView.tsx`
    - Add WebSocket connection when `recordingStatus` is `'processing'`
      or `'pending'`
    - On `vod_status` event matching this `vodId`: update state to show
      player (if `ready`) or error (if `failed`)
    - Clean up WebSocket on unmount or status change
    - Tests: mock WebSocket, verify state transitions
    - Satisfies: US5 AC1, AC2, AC3

13. [ ] [P] (Frontend) **LiveStreamCard: show live thumbnails**
    - Files: `frontend/src/components/LiveStreamCard.tsx`
    - When `thumbnailUrl` is set, render `<img>` with `onError` fallback
      to existing placeholder
    - No changes needed when `thumbnailUrl` is `null`
    - Satisfies: US1b AC5, AC6

14. [x] (Frontend) **Use backend-provided `hlsUrl` in VOD page**
    - Files: `frontend/src/app/vods/[id]/page.tsx`
    - Use `vod.hlsUrl` from API response instead of constructing
      `/hls/vods/${id}/index.m3u8`
    - Satisfies: uses real backend data

15. [x] (Frontend) **Full build + lint + test suite**
    - Files: all
    - `npx tsc --noEmit`, `npm run lint`, `npm test` — all pass
    - `go build ./...`, `go vet ./...` — all pass
    - Satisfies: all acceptance criteria verified

### Parallel Execution Plan

```
Phase 1:  Task 1, Task 2, Task 3      (parallel infra)
Phase 2:  Task 4 → Task 5 → {6, 7} → 8
Phase 3:  Task 9 → Task 10            (worker depends on backend queue)
Phase 4:  Task 11 → {12, 13} → 14 → 15
```

Total: 15 tasks across 4 phases. Backend + Worker = 10 tasks, Frontend = 5 tasks.

## Implementation Notes

Findings discovered during implementation that deviate from or clarify the
approved design. Added by the engineering team, flagged for the PM.

1. **SRS 5.0 dropped LL-HLS.** `hls_ll_enabled` / `hls_ll_fragment` no longer
exist in SRS 5.0.213 (config validation fails with "illegal
vhost.hls.hls_ll_enabled"). We use plain HLS with 2s fragments / 6-segment
window instead. The `specs/hls-low-latency.md` LL-HLS tuning is not
applicable on this SRS version.
2. **The `ossrs/srs:5` image ignores `conf/srs.conf`.** Its entrypoint loads
`conf/docker.conf`, so our tuned config must be mounted at
`/usr/local/srs/conf/docker.conf` (see `data/docker.conf`). This is why
earlier tuning attempts had no effect and why SRS defaulted to 10s
fragments / 60s window.
3. **SRS http_hooks `client_id` is a string** (connection id like
`5u9c4d30`), not a number. `streams.srs_client_id` is now TEXT
(migration 000004); `OnStreamStart`/`OnStreamEnd`/`disconnectSRSClient`
all use string IDs.
4. **Recording MP4 needs `-bsf:a aac_adtstoasc`.** SRS HLS segments carry
ADTS-framed AAC; muxing TS→MP4 fails with "Malformed AAC bitstream"
without the bitstream filter.
5. **SRS `on_unpublish` sends many extra JSON fields** — the handler must
not use `DisallowUnknownFields()` (same as `on_publish`).
6. **SRS does not send duration in `on_unpublish`** — `OnStreamEnd` computes
duration from `started_at` when the caller passes 0.
7. **Recording start is delayed until the first HLS segment exists**
(~2–5s after publish; the retry loop covers the 404 window). The first
few seconds of a stream are not in the recording. Acceptable for v1;
RTMP-pull recording (`rtmp://srs:1935/live/{key}`) would capture from
byte zero if this becomes a requirement.
8. **Recording robustness**: fragmented MP4 flags
(`frag_keyframe+empty_moov+default_base_moof`) keep the file playable when
ffmpeg is SIGKILLed at stream end, and ffmpeg stderr is captured for
logging (was silently discarded).
9. **SRS `hls_ctx` must be disabled.** By default SRS wraps every playlist
(including static VOD playlists) in a master playlist pointing at a
root-absolute child URL (`/vods/...?...hls_ctx=...`). hls.js resolves that
against the frontend origin, losing the `/hls` proxy prefix → 404. With
`hls_ctx off`, SRS serves the raw playlist with relative segment URIs and
both live and VOD playback work through the `/hls/*` rewrite.
