# Live Streaming Platform — Shared Context

**Status:** Approved
**Created:** 2026-08-06
**Platform:** Web only (v1), Google OAuth, SRS RTMP, HLS playback

---

## Architecture

```
[OBS / Streamer] ──RTMP──▶ [SRS :1935]
                               │
                    on_publish │  │ HLS segments → disk
                    callback   │  │
                               ▼  ▼
                          [Go/chi Backend :8080]
                               │
                    ┌──────────┼──────────┐
                    │          │          │
               REST API    WebSocket   Session Auth
               (chi mux)   /ws/chat    (JWT cookie)
                    │          │          │
                    ▼          ▼          ▼
              [PostgreSQL 16]         [Next.js Frontend :3000]
                                          │
                                          ▼
                                     [Browser]
```

### Component Summary

| Component | Role | Port |
|-----------|------|------|
| SRS | RTMP ingest, HLS output, HTTP callbacks | 1935 (RTMP), 8080 (HTTP API), serves HLS on 8080 |
| Go/chi Backend | REST API, WebSocket chat hub, auth, stream management | 8080 (or 8081 to avoid SRS conflict — decide at deploy) |
| PostgreSQL | Users, streams, chat, analytics, viewer tracking | 5432 |
| Next.js Frontend | SSR, HLS video player, chat UI | 3000 |

### Technology Choices

| Choice | Technology | Rationale |
|--------|-----------|-----------|
| RTMP server | SRS v5 in Docker | https://github.com/ossrs/srs — written in C++, battle-tested, native HTTP callbacks |
| WebSocket | `nhooyr.io/websocket` | https://pkg.go.dev/nhooyr.io/websocket — context-aware, no global state, modern API |
| Google OAuth | `golang.org/x/oauth2` | https://pkg.go.dev/golang.org/x/oauth2 — stdlib-adjacent, maintained by Go team |
| Sessions | JWT in httpOnly cookie | XSS-resistant (httpOnly), CSRF-protected via SameSite=Lax |
| Stream keys | crypto/rand UUID → bcrypt hash stored | Never store plaintext; OBS sends plaintext → backend hashes for lookup |
| SRS integration | SRS HTTP callback on `on_publish`/`on_unpublish` | SRS hits our backend; we validate the stream key and update status |
| HLS storage | Local disk, served by SRS | No CDN in v1; SRS serves HLS directly to browsers |
| Chat broadcasting | In-memory hub per stream + PostgreSQL persistence | Fan-out to all WebSocket clients on the same stream |

### Non-Goals (applies to all stories)

- ❌ Monetization (subscriptions, bits, ads, tipping) — v2
- ❌ Native mobile apps (iOS/Android) — v2, web-only for v1
- ❌ Email/password auth — Google OAuth only for v1
- ❌ Streamer categories/tags beyond a single text field — no curated taxonomy
- ❌ Moderation tools (banning users, deleting messages) — v2
- ❌ Following/subscribing to channels — no notification system
- ❌ Clips/highlights from VODs — v2
- ❌ Multiple stream keys per user — one key per account
- ❌ Transcoding/adaptive bitrate — single quality output for v1
- ❌ Embedded player for external sites — watch on our platform only
- ❌ CDN — single origin server for v1
- ❌ Streamer profile customization — bio, banner in v2 (avatar auto-imported from Google)

---

## Data Model (Complete DDL)

```sql
-- ============================================================
-- US1: Streamer Onboarding
-- ============================================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    google_id TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL,
    name TEXT NOT NULL,
    avatar_url TEXT,                              -- auto-imported from Google
    stream_key_hash TEXT NOT NULL,                -- bcrypt hash of stream key
    stream_title TEXT CHECK (
        stream_title IS NULL OR char_length(stream_title) BETWEEN 1 AND 100
    ),
    stream_category TEXT,
    is_live BOOLEAN NOT NULL DEFAULT false,
    live_since TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_google_id ON users(google_id);
CREATE INDEX idx_users_is_live ON users(is_live) WHERE is_live = true;

-- ============================================================
-- US2 + US5: Streams (each broadcast session)
-- ============================================================
CREATE TABLE streams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    title TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'live'
        CHECK (status IN ('live', 'interrupted', 'offline')),
    hls_path TEXT,                                -- SRS HLS output path
    recording_path TEXT,                          -- VOD file path (after stream ends)
    recording_status TEXT NOT NULL DEFAULT 'pending'
        CHECK (recording_status IN ('pending', 'processing', 'ready', 'failed')),
    peak_viewers INTEGER NOT NULL DEFAULT 0,
    total_viewers INTEGER NOT NULL DEFAULT 0,
    duration_seconds INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_streams_user_id ON streams(user_id);
CREATE INDEX idx_streams_status ON streams(status);
CREATE INDEX idx_streams_started_at ON streams(started_at DESC);
CREATE INDEX idx_streams_recording_status ON streams(recording_status)
    WHERE recording_status = 'ready';

-- ============================================================
-- US4: Chat Messages
-- ============================================================
CREATE TABLE chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stream_id UUID NOT NULL REFERENCES streams(id),
    user_id UUID NOT NULL REFERENCES users(id),
    message TEXT NOT NULL CHECK (char_length(message) BETWEEN 1 AND 500),
    sent_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_chat_messages_stream_sent
    ON chat_messages(stream_id, sent_at);

-- ============================================================
-- US3 + US6: Viewer tracking (for viewer count + analytics)
-- ============================================================
CREATE TABLE stream_viewers (
    stream_id UUID NOT NULL REFERENCES streams(id),
    user_id UUID REFERENCES users(id),            -- NULL = anonymous viewer
    client_id TEXT NOT NULL,                       -- anonymous session ID or user ID
    first_seen TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (stream_id, client_id)
);

CREATE INDEX idx_stream_viewers_active
    ON stream_viewers(stream_id) WHERE last_seen > now() - interval '2 minutes';

-- ============================================================
-- US6: Stream Analytics (pre-computed daily)
-- ============================================================
CREATE TABLE stream_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    date DATE NOT NULL,
    total_seconds INTEGER NOT NULL DEFAULT 0,
    peak_viewers INTEGER NOT NULL DEFAULT 0,
    unique_viewers INTEGER NOT NULL DEFAULT 0,
    UNIQUE (user_id, date)
);

CREATE INDEX idx_stream_analytics_user_date
    ON stream_analytics(user_id, date DESC);
```

### Architecture Decisions

| Decision | Choice | Rejected |
|----------|--------|----------|
| Primary keys | UUID v4 (`gen_random_uuid()`) | Auto-increment (leaks count in URLs), ULID (no native PG support) |
| Stream key storage | bcrypt hash (never plaintext) | Plaintext in DB (leak = all accounts compromised) |
| Sessions | JWT in httpOnly, SameSite=Lax cookie | localStorage JWT (XSS-vulnerable), server-side sessions (stateful, harder to scale) |
| SRS integration | HTTP callback (`on_publish`/`on_unpublish`) | Polling SRS API (adds latency), reading SRS logs (fragile) |
| Chat persistence | PostgreSQL with index on (stream_id, sent_at) | Redis only (no VOD replay), in-memory only (lost on restart) |
| HLS storage | Local disk, served by SRS | Object storage (S3) — v2 when CDN is added |
| Go project layout | Following `go-chi` skill conventions | Flat structure (harder to navigate at scale) |

### Story Dependencies

```
US1 (Onboarding) ──┬──▶ US2 (Going Live) ──▶ US3 (Viewing)
                   │                          │
                   │                          ├──▶ US4 (Chat)
                   │                          │
                   │                          └──▶ US5 (VOD)
                   │
                   └──▶ US6 (Dashboard)
```

---

## Task Checklist

### Phase 4a — Backend Foundation (US1 + US2 backend only)

> **Goal:** Get auth working, stream keys generated, SRS integration live.
> Frontend starts after these are stable.

1. [ ] **(Backend)** Scaffold Go/chi project structure
   - `backend/cmd/server/main.go`, `internal/{handler,service,store,middleware,model,errs,config}`
   - chi router with middleware chain (RequestID, RealIP, Logger, Recoverer, Timeout)
   - Docker Compose with Go backend + PostgreSQL + SRS
   → Satisfies: project setup
   → Files: `backend/go.mod`, `backend/cmd/server/main.go`, `docker-compose.yml`

2. [ ] **(Backend)** Create database migrations
   - `users`, `streams`, `chat_messages`, `stream_viewers`, `stream_analytics` tables
   - All indexes from DDL
   → Satisfies: data model (all stories)
   → Files: `backend/db/migrations/000001_*.sql`

3. [ ] **(Backend)** Implement Google OAuth flow
   - `GET /api/auth/google` — redirect to Google
   - `GET /api/auth/google/callback` — exchange code, create/lookup user, set JWT cookie
   - JWT middleware for protected routes
   → Satisfies: US1 AC1, US1 AC2
   → Files: `backend/internal/handler/auth.go`, `backend/internal/middleware/auth.go`

4. [ ] **(Backend)** Implement `GET /api/me` and `POST /api/me/stream-key/regenerate`
   - Return user profile + stream key
   - Regenerate: generate new UUID, bcrypt hash, store, return plaintext
   → Satisfies: US1 AC3, US1 AC4
   → Files: `backend/internal/handler/users.go`

5. [ ] **(Backend)** Implement SRS webhook handler
   - `POST /api/srs/callback` — validate auth header, handle `on_publish`/`on_unpublish`
   - Stream key lookup: hash incoming key, match against `users.stream_key_hash`
   - On publish: create `streams` row, set `users.is_live=true`
   - On unpublish: set `streams.status='offline'`, trigger VOD processing goroutine
   → Satisfies: US2 AC1, US2 AC2, US2 AC4
   → Files: `backend/internal/handler/srs.go`, `backend/internal/service/streams.go`

6. [ ] **(Backend)** Implement `GET /api/streams/live` and `GET /api/channel/:username`
   - Live streams: query `users` WHERE `is_live=true`, join with live `streams`
   - Channel: return user + live stream info + viewer count
   → Satisfies: US2 AC3, US3 (provides data for homepage + channel page)
   → Files: `backend/internal/handler/streams.go`, `backend/internal/handler/channels.go`

7. [ ] **(Backend)** Write table-driven tests for all US1 + US2 endpoints
   - Happy path + error path for each handler
   - Integration test: `httptest.NewServer` + real SRS callback simulation
   → Satisfies: test coverage for US1, US2

### Phase 4b — Frontend Onboarding + Dashboard (US1 + US6 frontend)

> **Goal:** Users can sign in, see their dashboard, manage stream settings.
> Can start once task 4 (GET /api/me) is stable.

8. [ ] [P] **(Frontend)** Scaffold Next.js App Router project
   - `frontend/src/app/layout.tsx`, `frontend/src/app/page.tsx`
   - Tailwind CSS with design tokens (CSS custom properties)
   - API client utility (`frontend/src/lib/api.ts`)
   → Satisfies: project setup
   → Files: `frontend/package.json`, `frontend/src/app/layout.tsx`

9. [ ] [P] **(Frontend)** Build Landing Page
   - Hero section: title, subtitle, "Start Streaming with Google" button
   - Live stats: count of live streams + past streams
   - Sign-in redirects to `/api/auth/google`
   → Satisfies: US1 AC1
   → Files: `frontend/src/app/page.tsx`

10. [ ] [P] **(Frontend)** Build Dashboard page
    - StreamKeyDisplay component: masked key, copy button, toast on copy
    - RegenerateKeyButton with confirmation dialog
    - StreamSettingsForm: title input + category input + save button → `PATCH /api/me/settings`
    - AnalyticsCards: 4 cards calling `GET /api/me/analytics`
    - ForceEndButton: only when live, dual-confirm → `POST /api/me/stream/end`
    → Satisfies: US1 AC3, US1 AC4, US6 AC1, US6 AC2, US6 AC3
    → Files: `frontend/src/app/dashboard/page.tsx`, `frontend/src/components/`

### Phase 4c — Frontend Viewing Experience (US3 + US4 + US5)

> **Goal:** Viewers can browse live streams, watch, chat, and search VODs.
> Can start once task 6 (GET /api/streams/live) is stable.

11. [ ] [P] **(Frontend)** Build Homepage Live Grid
    - `LiveStreamCard` component: thumbnail, live indicator, viewer count, streamer info
    - Grid layout with skeleton loading, empty state, error state
    - Calls `GET /api/streams/live`
    → Satisfies: US3 AC1, US3 AC4
    → Files: `frontend/src/app/page.tsx` (add live grid below hero)

12. [ ] [P] **(Frontend)** Build Channel Page (video player + chat)
    - `VideoPlayer` component: hls.js integration, controls, "Stream ended" overlay
    - `StreamInfo` component: avatar, title, viewer count, category
    - Layout: video left, chat right (desktop), stacked (mobile)
    → Satisfies: US3 AC2, US3 AC3
    → Files: `frontend/src/app/channel/[id]/page.tsx`

13. [ ] [P] **(Backend)** Implement WebSocket chat hub
    - `WS /ws/chat/:streamId` — upgrade, validate stream is live, hub pattern
    - Broadcast messages to all connected clients on the same stream
    - Persist messages to `chat_messages` table
    - Rate limiting: 1 msg / 2 seconds per user
    - Auto-close idle connections after 2 minutes
    → Satisfies: US4 AC1, US4 AC2, US4 AC3
    → Files: `backend/internal/handler/chat.go`, `backend/internal/service/chat.go`

14. [ ] [P] **(Backend)** Implement chat history endpoint
    - `GET /api/chat/:streamId/messages` — paginated, `?before=` cursor
    - For VOD chat replay
    → Satisfies: US4 AC4
    → Files: `backend/internal/handler/chat.go`

15. [ ] [P] **(Frontend)** Build ChatPanel component
    - WebSocket connection to `/ws/chat/:streamId`
    - Auto-reconnect with exponential backoff
    - Message list with auto-scroll, ChatInput component
    - States: loading, connected (signed out), connected (signed in), reconnecting, stream ended
    - VOD mode: read-only, synced to video timeline via `GET /api/chat/:streamId/messages`
    → Satisfies: US4 AC1, US4 AC2, US4 AC3, US4 AC4
    → Files: `frontend/src/components/ChatPanel.tsx`, `frontend/src/components/ChatInput.tsx`

16. [ ] [P] **(Backend)** Implement VOD endpoints + VOD processing
    - `GET /api/channel/:userId/vods` — paginated VOD list
    - `GET /api/vods/:vodId` — VOD detail + HLS URL
    - Background goroutine: on stream end, validate HLS segments, generate thumbnail (ffmpeg), mark ready
    → Satisfies: US5 AC1, US5 AC3, US5 AC4
    → Files: `backend/internal/handler/vods.go`, `backend/internal/service/vods.go`

17. [ ] [P] **(Backend)** Implement search endpoint
    - `GET /api/search?q=` — PostgreSQL full-text search across `streams.title` + `users.name`
    - `tsvector` column + GIN index
    → Satisfies: US5 AC2
    → Files: `backend/internal/handler/search.go`

18. [ ] [P] **(Frontend)** Build VOD page
    - `VodPlayer`: same HLS player, with seek + playback speed controls
    - `VodInfo`: title, streamer, date, duration, view count
    - `ChatReplay`: read-only chat panel synced to video timeline
    - States: loading, ready, processing, failed
    → Satisfies: US5 AC3, US5 AC4
    → Files: `frontend/src/app/vods/[id]/page.tsx`

19. [ ] [P] **(Frontend)** Build Search page
    - SearchInput with auto-focus, 300ms debounce
    - VodResultCard grid with thumbnail, title, streamer, duration, date
    - States: empty input (recent VODs), typing, loading, empty results, error
    → Satisfies: US5 AC2
    → Files: `frontend/src/app/search/page.tsx`

### Phase 4d — Analytics Backend (US6 backend)

> **Goal:** Streamer analytics are pre-computed and queryable.
> Can start after task 5 (SRS webhook) is stable.

20. [ ] [P] **(Backend)** Implement `GET /api/me/analytics` and `POST /api/me/stream/end`
    - Analytics: read from `stream_analytics` table (pre-computed)
    - Post-stream hook: after stream ends, aggregate viewer data → upsert `stream_analytics` row
    - Force-end: call SRS HTTP API to disconnect publisher
    → Satisfies: US6 AC2, US6 AC3
    → Files: `backend/internal/handler/analytics.go`, `backend/internal/service/analytics.go`

### Phase 4e — Viewer Heartbeat + Polish

21. [ ] [P] **(Backend)** Implement viewer heartbeat endpoint
    - `POST /api/streams/:streamId/viewer-heartbeat` — upsert `stream_viewers` row
    - No auth required (anonymous viewers tracked by `client_id`)
    → Satisfies: viewer count for US3
    → Files: `backend/internal/handler/viewers.go`

22. [ ] **(All)** End-to-end verification
    - Streamer signs up → gets key → streams from OBS
    - Viewer visits homepage → sees live stream → watches + chats
    - Stream ends → VOD appears → searchable
    - All acceptance criteria verified

---

### Parallel Execution Plan

```
Task  1 ──▶ Task  2 ──▶ Task  3 ──▶ Task  4 ──▶ Task  5 ──▶ Task  6
                                                       │
              ┌────────────────────────────────────────┤
              ▼                                        ▼
         Task  8 (frontend setup)              Task  7 (tests)
              │
         Task  9 (landing)
              │
         Task 10 (dashboard)                           
                                                       
         After Task 6 stable:                         
              │                                        
     ┌────────┼────────┬────────┬────────┐            
     ▼        ▼        ▼        ▼        ▼            
  T11      T13      T14      T16      T20            
  (grid)   (WS)     (chat    (VOD     (analytics     
                    history)  backend)  backend)      
     │        │        │        │        │            
     ▼        ▼        ▼        ▼        ▼            
  T12      T15      T15      T17      T10            
  (chan    (chat    (chat    (search  (dashboard      
   page)   panel)   panel)   backend)  updates)       
                         │        │                   
                         ▼        ▼                   
                      T18      T19                   
                      (VOD     (search               
                       page)   page)                 
                                                      
                              ▼                       
                         T21 (heartbeat)              
                              │                       
                              ▼                       
                         T22 (E2E verify)             
```

**[P]** = Can be parallelized with others in the same phase

**Role assignments per task:**
- Tasks 1-7, 13-14, 16-17, 20-21: Backend Engineer
- Tasks 8-12, 15, 18-19: Frontend Engineer
- Task 22: All engineers
