# Backend API Endpoints — Live Streaming Platform

**Feature:** Live Streaming Platform — Backend REST API
**Status:** Implemented
**Owner:** PM
**Created:** 2026-08-08
**Completed:** 2026-08-08
**Depends on:** `specs/live-streaming-platform.md` (Review), `specs/live-streaming-platform/_shared.md` (Approved)

---

## Context

The frontend already calls these six endpoints and defines TypeScript types for
every request and response in `frontend/src/types/index.ts`. The database schema
in `backend/db/migrations/000001_initial_schema.up.sql` has all required tables.
Auth middleware (JWT via `Authorization: Bearer` header or `token` cookie)
already works. The auth handler in
`backend/internal/modules/auth/adapter/http/handler.go` already registers five of
the six routes and has handler method stubs. The following gaps remain:

| Endpoint | Status | Gap |
|---|---|---|
| `GET /api/me` | Stub exists | Response shape doesn't match frontend `User` type (missing `streamCategory`, `streamKey`) |
| `POST /api/me/stream-key/regenerate` | Stub exists | Needs `RegenerateStreamKey` wired through service layer |
| `PATCH /api/me/settings` | Stub exists | Response shape should return updated fields, not `{"status":"ok"}` |
| `GET /api/me/analytics` | Stub exists | Needs `streamOps.GetAnalytics` implementation (streams service) |
| `POST /api/me/stream/end` | Stub exists | Needs `streamOps.ForceEndStream` implementation (streams service) |
| `GET /api/streams/live` | Missing entirely | No route, no handler — must be built from scratch |

---

## Requirements

### User Story 1 — Get Current User Profile (GET /api/me)

**As a** streamer or viewer,
**I want** to retrieve my user profile including my stream key and live status,
**so that** the frontend dashboard can display my account info and OBS configuration.

**Acceptance Criteria:**

- **Given** a valid JWT is present in the `Authorization: Bearer <token>` header or `token` cookie,
  **When** the frontend calls `GET /api/me`,
  **Then** the backend returns HTTP 200 with the `User` object matching the shape defined in `frontend/src/types/index.ts`:
  ```json
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Alice Streamer",
    "email": "alice@example.com",
    "avatarUrl": "https://lh3.googleusercontent.com/...",
    "streamKey": "sk-a1b2c3d4e5f6...",
    "streamTitle": "Late night coding",
    "streamCategory": "Programming",
    "isLive": true
  }
  ```
- **Given** the user's `streamTitle` or `streamCategory` is null in the database,
  **When** the endpoint is called,
  **Then** the response includes `null` for those fields (not omitted).
- **Given** no valid authentication is provided,
  **When** the endpoint is called,
  **Then** the backend returns HTTP 401 with `{"error": "unauthorized"}`.

**Non-Goals for this story:**
- ❌ Return `avatarUrl` as an empty string instead of null — the frontend expects `null` when no avatar is set.
- ❌ Return the raw stream key hash — always return `null` or omit the field if the stream key cannot be resolved to plaintext (the backend stores only the bcrypt/SHA-256 hash).

---

### User Story 2 — List Live Streams (GET /api/streams/live)

**As a** viewer (authenticated or anonymous),
**I want** to browse all currently live streams,
**so that** the homepage can display a grid of live channels.

**Acceptance Criteria:**

- **Given** one or more streams are live,
  **When** the frontend calls `GET /api/streams/live`,
  **Then** the backend returns HTTP 200 with the `LiveStreamsResponse` shape defined in `frontend/src/types/index.ts`:
  ```json
  {
    "streams": [
      {
        "userId": "550e8400-...",
        "streamerName": "Alice Streamer",
        "streamerAvatarUrl": "https://...",
        "streamId": "660e8400-...",
        "title": "Late night coding",
        "category": "Programming",
        "viewerCount": 42,
        "thumbnailUrl": "https://...",
        "startedAt": "2026-08-08T01:23:45Z"
      }
    ],
    "total": 1
  }
  ```
- **Given** the streamer has no avatar (`users.avatar_url IS NULL`),
  **When** the endpoint is called,
  **Then** `streamerAvatarUrl` is `null`.
- **Given** no streams are live,
  **When** the endpoint is called,
  **Then** returns HTTP 200 with `{"streams": [], "total": 0}`.
- **Given** the request is unauthenticated,
  **When** the endpoint is called,
  **Then** still returns HTTP 200 — this endpoint is public (no auth required).

**Viewer count semantics:**
- `viewerCount` is the count of distinct `client_id` values in `stream_viewers` where `last_seen` is within the last 60 seconds.
- This includes both authenticated and anonymous viewers.

**Non-Goals for this story:**
- ❌ Server-sent events or WebSocket for live viewer count updates — the frontend polls this endpoint.
- ❌ Filtering or pagination — the endpoint returns all live streams.
- ❌ Sorting beyond the natural database order — the frontend handles any custom sorting.

---

### User Story 3 — Streamer Analytics (GET /api/me/analytics)

**As a** streamer,
**I want** to view aggregated analytics for my streams over a selected period,
**so that** I can understand my viewership trends.

**Acceptance Criteria:**

- **Given** a valid JWT and the streamer has streamed this week,
  **When** the frontend calls `GET /api/me/analytics?period=week`,
  **Then** the backend returns HTTP 200 with the `Analytics` shape defined in `frontend/src/types/index.ts`:
  ```json
  {
    "period": "week",
    "startDate": "2026-08-02",
    "endDate": "2026-08-08",
    "totalStreamTimeSeconds": 43200,
    "peakViewers": 142,
    "totalUniqueViewers": 1205,
    "totalStreams": 5
  }
  ```
- **Given** the `period` query parameter is omitted,
  **When** the endpoint is called,
  **Then** defaults to `"week"`.
- **Given** the `period` is `"week"`,
  **When** the analytics are computed,
  **Then** `startDate` is Monday 00:00:00 UTC and `endDate` is Sunday 23:59:59 UTC of the current week.
- **Given** the `period` is `"month"`,
  **When** the analytics are computed,
  **Then** date range covers the current calendar month.
- **Given** the `period` is `"all"`,
  **When** the analytics are computed,
  **Then** date range covers all recorded data (from the user's first stream to today).
- **Given** the streamer has never streamed,
  **When** the endpoint is called,
  **Then** returns HTTP 200 with all numeric fields as `0` and `period`/dates reflecting the requested range.
- **Given** an invalid period is passed (e.g., `period=year`),
  **When** the endpoint is called,
  **Then** returns HTTP 400 with `{"error": "invalid period: year; expected week, month, or all"}`.
- **Given** no valid authentication,
  **When** the endpoint is called,
  **Then** returns HTTP 401.

**Data source:**
- Analytics MUST be read from the `stream_analytics` table (pre-computed per-user per-day), NOT from a raw `JOIN` of `streams` + `stream_viewers`.
- Aggregation: sum `total_seconds`, max of `peak_viewers`, sum of `unique_viewers`, count of rows for `totalStreams`.

**Non-Goals for this story:**
- ❌ Real-time analytics — data is only as fresh as the last batch update (post-stream hook or cron).
- ❌ Per-stream breakdown — aggregate-only for v1.
- ❌ Charts or graphs — the endpoint returns raw numbers; the frontend renders visualizations.

---

### User Story 4 — Regenerate Stream Key (POST /api/me/stream-key/regenerate)

**As a** streamer,
**I want** to regenerate my stream key,
**so that** I can revoke a compromised key and get a new one.

**Acceptance Criteria:**

- **Given** a valid JWT,
  **When** the frontend calls `POST /api/me/stream-key/regenerate` with an empty body,
  **Then** the backend generates a new 64-character hex stream key (32 random bytes), hashes it with SHA-256, updates `users.stream_key_hash`, and returns HTTP 200:
  ```json
  {
    "streamKey": "a1b2c3d4e5f6..."
  }
  ```
- **Given** the request body includes `{"confirm": true}` (from a client-side confirmation dialog),
  **When** the endpoint is called,
  **Then** the backend verifies confirmation and proceeds.
- **Given** the request body includes `{"confirm": false}` or an unrecognized field,
  **When** the endpoint is called,
  **Then** returns HTTP 400 with `{"error": "must confirm stream key regeneration"}`.
- **Given** the old stream key is in use by an active RTMP connection,
  **When** the key is regenerated,
  **Then** the active stream is NOT disconnected — the new key takes effect for the next OBS connection only.
- **Given** no valid authentication,
  **When** the endpoint is called,
  **Then** returns HTTP 401.

**Security:**
- The raw stream key MUST NOT be stored in the database — only the SHA-256 hash is persisted.
- The raw key is returned in the response exactly once. If the user navigates away, they must regenerate again.

**Non-Goals for this story:**
- ❌ Disconnecting the current RTMP stream on key regeneration — out of scope for v1.
- ❌ Stream key expiration or rotation policy — keys are permanent until manually regenerated.

---

### User Story 5 — Update Stream Settings (PATCH /api/me/settings)

**As a** streamer,
**I want** to update my stream title and category,
**so that** viewers can discover my stream and understand what I'm broadcasting.

**Acceptance Criteria:**

- **Given** a valid JWT and a valid request body,
  **When** the frontend calls `PATCH /api/me/settings` with:
  ```json
  {
    "streamTitle": "Late night coding session",
    "streamCategory": "Programming"
  }
  ```
  **Then** the backend updates `users.stream_title` and `users.stream_category`, and returns HTTP 200 echoing the updated values:
  ```json
  {
    "streamTitle": "Late night coding session",
    "streamCategory": "Programming"
  }
  ```
- **Given** `streamTitle` is missing, empty, or only whitespace,
  **When** the endpoint is called,
  **Then** returns HTTP 400 with `{"error": "streamTitle is required and must be 1-100 characters"}`.
- **Given** `streamTitle` exceeds 100 characters,
  **When** the endpoint is called,
  **Then** returns HTTP 400 (backed by the database CHECK constraint on `users.stream_title`).
- **Given** `streamCategory` is omitted,
  **When** the endpoint is called,
  **Then** the category is cleared (set to `null` in the database).
- **Given** `streamCategory` exceeds 100 characters,
  **When** the endpoint is called,
  **Then** returns HTTP 400.
- **Given** no valid authentication,
  **When** the endpoint is called,
  **Then** returns HTTP 401.
- **Given** the user is currently live,
  **When** settings are updated,
  **Then** the new title/category takes effect immediately on the channel page (the `GET /api/me` response reflects the change on next poll).

**Validation rules (summary):**

| Field | Required | Min | Max |
|-------|----------|-----|-----|
| `streamTitle` | Yes | 1 char | 100 chars |
| `streamCategory` | No | — | 100 chars |

**Non-Goals for this story:**
- ❌ Updating fields beyond `streamTitle` and `streamCategory` (avatar, name, email come from Google OAuth).
- ❌ Pushing live updates to viewers — viewers will see the new title on next page load or poll.

---

### User Story 6 — Force-End Current Stream (POST /api/me/stream/end)

**As a** streamer,
**I want** to force-end my current live stream from the dashboard,
**so that** I can stop broadcasting without using OBS controls.

**Acceptance Criteria:**

- **Given** a valid JWT and the user is currently live (`users.is_live = true`),
  **When** the frontend calls `POST /api/me/stream/end` with an empty body,
  **Then** the backend:
  1. Calls the SRS HTTP API to disconnect the RTMP publisher (`DELETE /api/v1/clients/<srs_client_id>`)
  2. Sets `users.is_live = false` and `users.live_since = NULL`
  3. Sets `streams.status = 'offline'`, `streams.ended_at = now()`, `streams.duration_seconds`
  4. Triggers VOD processing (recording status transition)
  5. Returns HTTP 200:
     ```json
     {
       "status": "offline",
       "message": "Stream ended"
     }
     ```
- **Given** the user is NOT currently live,
  **When** the endpoint is called,
  **Then** returns HTTP 409 with `{"error": "no active stream to end"}`.
- **Given** the SRS API call fails (SRS is unreachable or returns an error),
  **When** the endpoint is called,
  **Then** the backend still sets the user as offline and marks the stream as ended in the database, but returns HTTP 200 with `{"status": "offline", "message": "Stream ended (publisher disconnect may have failed)"}` and logs the SRS error.
- **Given** no valid authentication,
  **When** the endpoint is called,
  **Then** returns HTTP 401.
- **Given** the stream has already been force-ended and the request is repeated (idempotency),
  **When** the endpoint is called,
  **Then** returns HTTP 409 (same as "no active stream") because `is_live` is already `false`.

**Side effects:**
- The `streams` row is finalized: `status → 'offline'`, `ended_at` set, `duration_seconds` computed.
- A post-stream hook (or async job) computes daily analytics into `stream_analytics`.
- VOD processing begins if a recording exists.

**Non-Goals for this story:**
- ❌ Graceful shutdown notification to viewers — viewers see the player stop; no custom "stream ended" overlay.
- ❌ Undo or "resume" after force-ending — the stream cannot be restarted on the same `streams` row.

---

## Explicit Non-Goals (across all stories)

These are things we are explicitly NOT building in the scope of these six endpoints:

- ❌ **New database tables or schema migrations** — all required tables already exist.
- ❌ **New middleware** — auth middleware (`AuthMiddleware`) is reused; no new CORS, rate-limiting, or logging middleware for these endpoints.
- ❌ **New third-party dependencies** — all functionality uses existing deps (`go-chi`, `golang-jwt`, `pgx`, `crypto/rand`, `crypto/sha256`).
- ❌ **WebSocket or push notifications** for live viewer counts or settings changes — the frontend polls REST endpoints.
- ❌ **Stream key revocation notification** — no email or in-app notification when a key is regenerated.
- ❌ **Analytics export or CSV download** — the frontend renders the JSON response; no file download endpoint.
- ❌ **Stream categories as an enum or curated list** — `streamCategory` is a free-text field (matches existing schema).
- ❌ **Bulk operations** — no batch endpoint for ending multiple streams or updating multiple users.
- ❌ **Admin override endpoints** — no admin-facing API to force-end another user's stream or view other users' analytics.
- ❌ **Rate limiting** on stream key regeneration — a user can regenerate their key as often as they want.

---

## Design

### 1. Module Placement

| Endpoint | Module | Rationale |
|---|---|---|
| `GET /api/me` | `auth` | Authenticated user profile — natural fit alongside OAuth handlers. Already registered in `AuthHandler.RegisterRoutes`. |
| `POST /api/me/stream-key/regenerate` | `auth` | Operates on the authenticated user's `stream_key_hash` column. Already registered in `AuthHandler.RegisterRoutes`. |
| `PATCH /api/me/settings` | `auth` | Operates on `users.stream_title` and `users.stream_category`. Already registered in `AuthHandler.RegisterRoutes`. |
| `GET /api/me/analytics` | `auth` (handler) → `streams` (service) | The auth handler owns the route (authenticated, user-scoped) but delegates to `streamOps` (backed by `StreamService`) for analytics queries against `stream_analytics`. Already wired via the `streamOps` interface. |
| `POST /api/me/stream/end` | `auth` (handler) → `streams` (service) | Same pattern: auth handler owns the route, delegates to `streamOps.ForceEndStream`. Already wired via the `streamOps` interface. |
| `GET /api/streams/live` | `streams` | Public endpoint (no auth). Does NOT belong in the auth handler, which groups all routes under `AuthMiddleware`. Already registered in `StreamHandler.RegisterRoutes` outside any auth group. |

**Decision:** Keep all five `/api/me/*` routes in the `auth` module handler. They share authentication (`AuthMiddleware`) and operate on the authenticated user. `GET /api/me/analytics` and `POST /api/me/stream/end` delegate to `streamOps` (backed by `StreamService`) — this is already wired in `main.go` (`authhttp.NewAuthHandler(authSvc, streamSvc, ...)`).

**Rejected:** Moving analytics or force-end to the streams handler. Those routes ARE user-scoped ("my analytics", "my stream") and share the auth middleware group — keeping them in the auth handler avoids duplicating the `AuthMiddleware` import and keeps the route grouping consistent.

---

### 2. API Contracts

#### 2.1 `GET /api/me`

**Purpose:** Return the authenticated user's profile, including stream key (if resolvable), stream settings, and live status.

**Authentication:** Bearer token OR `token` cookie (via `AuthMiddleware`)

**Request:** No body. No query parameters.

**Success Response (200):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Alice Streamer",
  "email": "alice@example.com",
  "avatarUrl": "https://lh3.googleusercontent.com/...",
  "streamKey": null,
  "streamTitle": "Late night coding",
  "streamCategory": "Programming",
  "isLive": true
}
```

**Response shape contract:**
| Field | Type | Source | Notes |
|---|---|---|---|
| `id` | `string` | `users.id` | Always present |
| `name` | `string` | `users.name` | Always present |
| `email` | `string` | `users.email` | Always present |
| `avatarUrl` | `string \| null` | `users.avatar_url` | `null` when no avatar is set |
| `streamKey` | `string \| null` | N/A (hash only) | Always `null` — the raw key cannot be recovered from the SHA-256 hash. The only time the raw key is returned is during `POST /api/me/stream-key/regenerate`. |
| `streamTitle` | `string \| null` | `users.stream_title` | `null` when never set |
| `streamCategory` | `string \| null` | `users.stream_category` | `null` when never set |
| `isLive` | `boolean` | `users.is_live` | `true`/`false` |

> **Contract note:** The current handler response struct includes `createdAt` which is NOT in the frontend `User` type. This field must be removed from the handler response to match the contract.

**Error Responses:**
- **401:** No authentication provided, or token is invalid/expired. Body: `{"error": "unauthorized"}`
- **500:** Database query failure or unexpected error.

---

#### 2.2 `GET /api/streams/live`

**Purpose:** Return all currently live streams with viewer counts. Public — no authentication required.

**Authentication:** None

**Request:** No body. No query parameters.

**Success Response (200 — with streams):**
```json
{
  "streams": [
    {
      "userId": "550e8400-e29b-41d4-a716-446655440000",
      "streamerName": "Alice Streamer",
      "streamerAvatarUrl": "https://lh3.googleusercontent.com/...",
      "streamId": "660e8400-e29b-41d4-a716-446655440001",
      "title": "Late night coding",
      "category": "Programming",
      "viewerCount": 42,
      "thumbnailUrl": null,
      "startedAt": "2026-08-08T01:23:45Z"
    }
  ],
  "total": 1
}
```

**Success Response (200 — empty):**
```json
{
  "streams": [],
  "total": 0
}
```

**Response shape contract:**
| Field | Type | Source | Notes |
|---|---|---|---|
| `streams` | `LiveStream[]` | JOIN `streams` + `users` + subquery on `stream_viewers` | Empty array when no streams are live |
| `total` | `number` | `len(streams)` | Same as array length (no pagination in v1) |
| `streams[].userId` | `string` | `users.id` | |
| `streams[].streamerName` | `string` | `users.name` | |
| `streams[].streamerAvatarUrl` | `string \| null` | `users.avatar_url` | `null` when no avatar |
| `streams[].streamId` | `string` | `streams.id` | |
| `streams[].title` | `string` | `streams.title` | May differ from `users.stream_title` (set at stream start) |
| `streams[].category` | `string \| null` | `users.stream_category` | `null` when never set |
| `streams[].viewerCount` | `number` | `COUNT(DISTINCT client_id) WHERE last_seen >= now() - interval '60 seconds'` | Active viewers in last 60s |
| `streams[].thumbnailUrl` | `string \| null` | TBD | Currently mapped from `streams.hls_path` — see Architectural Finding #2 |
| `streams[].startedAt` | `string` (ISO 8601) | `streams.started_at` | |

> **⚠ Architectural Finding #1 — Response wrapper:** The current `ListLiveStreams` handler returns a bare `[]LiveStream` array. It must be changed to return `{"streams": [...], "total": N}` to match the frontend `LiveStreamsResponse` type.

> **⚠ Architectural Finding #2 — `thumbnailUrl` source:** The current `LiveStream` domain entity maps `HlsPath *string` to `json:"thumbnailUrl"`. This is semantically incorrect — `hls_path` is the HLS playlist URL for playback, not a thumbnail image. The correct data source would be a thumbnail generation pipeline (e.g., SRS's `http_hooks` capturing a frame, or a separate thumbnail service). **Decision:** Keep the current mapping as a pragmatic stopgap (the frontend expects the field to exist) but add a `ThumbnailUrl *string` field to the `LiveStream` domain entity that is explicitly sourced from a dedicated column or service. Defer implementation of actual thumbnail generation to a future spec. For v1, populate with `null` or the HLS path until a thumbnail pipeline exists.

**Error Responses:**
- **500:** Database query failure.

---

#### 2.3 `GET /api/me/analytics`

**Purpose:** Return aggregated streaming analytics for the authenticated user over a selected period.

**Authentication:** Bearer token OR `token` cookie (via `AuthMiddleware`)

**Request:** No body.

**Query Parameters:**
| Parameter | Type | Required | Default | Valid Values |
|---|---|---|---|---|
| `period` | `string` | No | `"week"` | `"week"`, `"month"`, `"all"` |

**Success Response (200 — with data):**
```json
{
  "period": "week",
  "startDate": "2026-08-02",
  "endDate": "2026-08-08",
  "totalStreamTimeSeconds": 43200,
  "peakViewers": 142,
  "totalUniqueViewers": 1205,
  "totalStreams": 5
}
```

**Success Response (200 — no data):**
```json
{
  "period": "week",
  "startDate": "2026-08-02",
  "endDate": "2026-08-08",
  "totalStreamTimeSeconds": 0,
  "peakViewers": 0,
  "totalUniqueViewers": 0,
  "totalStreams": 0
}
```

> **⚠ Architectural Finding #3 — `startDate`/`endDate` missing:** The current `Analytics` domain entity has no `startDate` or `endDate` fields, and the repository query does not compute them. These must be added to the domain entity and populated by the repository (or service layer) based on the `period` parameter. See Section 3.3 for the computation logic.

**Error Responses:**
- **400:** Invalid `period` value. Body: `{"error": "invalid period: year; expected week, month, or all"}`
- **401:** No authentication, or token is invalid/expired.
- **500:** Database query failure.

---

#### 2.4 `POST /api/me/stream-key/regenerate`

**Purpose:** Generate a new stream key, persist the SHA-256 hash, and return the raw key (one-time visibility).

**Authentication:** Bearer token OR `token` cookie (via `AuthMiddleware`)

**Request:**
```json
{
  "confirm": true
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `confirm` | `boolean` | Yes | Must be `true`. `false` or missing returns 400. |

An empty body (no JSON, or `Content-Length: 0`) is also accepted — the existing handler treats this as an implicit confirmation, since the frontend dialog already serves as the confirmation step.

**Success Response (200):**
```json
{
  "streamKey": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
}
```

**Error Responses:**
- **400:** Body present but `confirm` is `false` or unrecognized fields. Body: `{"error": "must confirm stream key regeneration"}`
- **400:** Malformed JSON.
- **401:** No authentication.
- **500:** Key generation or database update failure.

> **Design note:** This endpoint is already fully implemented in `AuthHandler.RegenerateStreamKey`. The service layer (`AuthService.GenerateStreamKey` + `AuthService.RegenerateStreamKey`) and repository (`UpdateStreamKey`) are all wired. No design changes needed — included here for contract completeness.

---

#### 2.5 `PATCH /api/me/settings`

**Purpose:** Update the authenticated user's stream title and category.

**Authentication:** Bearer token OR `token` cookie (via `AuthMiddleware`)

**Request:**
```json
{
  "streamTitle": "Late night coding session",
  "streamCategory": "Programming"
}
```

| Field | Type | Required | Constraints |
|---|---|---|---|
| `streamTitle` | `string` | Yes | 1–100 characters, non-blank |
| `streamCategory` | `string \| null` | No | Max 100 characters. Omit to set to `null`. |

**Success Response (200):**
```json
{
  "streamTitle": "Late night coding session",
  "streamCategory": "Programming"
}
```

When category is cleared:
```json
{
  "streamTitle": "Late night coding session",
  "streamCategory": null
}
```

> **Design note:** The current handler returns `{"status":"ok"}`. It must be changed to echo back the updated values to match the frontend `StreamSettings` type (which is also the response type — the frontend uses `StreamSettings` for both request and response).

**Error Responses:**
- **400:** `streamTitle` is missing, empty, or whitespace-only. Body: `{"error": "streamTitle is required and must be 1-100 characters"}`
- **400:** `streamTitle` exceeds 100 characters.
- **400:** `streamCategory` exceeds 100 characters.
- **400:** Malformed JSON or unrecognized fields.
- **401:** No authentication.
- **500:** Database update failure.

---

#### 2.6 `POST /api/me/stream/end`

**Purpose:** Force-end the authenticated user's current live stream. Calls SRS to disconnect the RTMP publisher, then updates database state.

**Authentication:** Bearer token OR `token` cookie (via `AuthMiddleware`)

**Request:** No body required (empty POST).

**Success Response (200 — normal):**
```json
{
  "status": "offline",
  "message": "Stream ended"
}
```

**Success Response (200 — SRS failure, graceful degradation):**
```json
{
  "status": "offline",
  "message": "Stream ended (publisher disconnect may have failed)"
}
```

**Error Responses:**
- **401:** No authentication.
- **409:** User is not currently live (no active stream to end). Body: `{"error": "no active stream to end"}`
- **500:** Database update failure after SRS call succeeded (partial state — see Section 5).

> **⚠ Architectural Finding #4 — Error code mismatch:** The current `StreamService.ForceEndStream` calls `repo.GetStreamByUserID()` which returns `errs.NotFound("no live stream for user %s", userID)` when there's no active stream. The `render.Error` helper maps `KindNotFound` → 404. The PM spec requires 409. The service must catch the not-found case and return `errs.Conflict("no active stream to end")` instead.

---

### 3. Data Flow

#### 3.1 `GET /api/me` — Data Flow

```
Handler (GetMe)
  → authService.GetByID(ctx, userID)
    → authRepo.GetByID(ctx, id)          — SELECT * FROM users WHERE id = $1
  ← *domain.User
  → Map to getMeResponse (see Section 2.1 contract)
  ← JSON 200
```

**Repository methods reused:** `auth.domain.Repository.GetByID` — no changes needed.

**Changes needed:**
- Handler response struct: add `StreamCategory *string \`json:\"streamCategory\"\``, remove `CreatedAt`.
- `streamKey` field: keep as `*string` with `omitempty` (nil → omitted). The frontend `User` type has `streamKey?: string` (optional), so omit is acceptable. The PM non-goal explicitly allows "null or omit."

#### 3.2 `GET /api/streams/live` — Data Flow

```
Handler (ListLiveStreams)
  → streamService.ListLive(ctx)
    → streamsRepo.ListLiveStreams(ctx)
      → SQL:
        SELECT s.id, u.id, u.name, u.avatar_url, s.title,
               COALESCE(u.stream_category, ''),
               s.started_at, s.hls_path,
               COALESCE(vc.viewer_count, 0)
        FROM streams s
        JOIN users u ON s.user_id = u.id
        LEFT JOIN (
          SELECT stream_id, COUNT(*) as viewer_count
          FROM stream_viewers
          WHERE last_seen >= now() - interval '60 seconds'   ← NEW FILTER
          GROUP BY stream_id
        ) vc ON vc.stream_id = s.id
        WHERE s.status = 'live'
        ORDER BY s.started_at DESC
      ← []domain.LiveStream
  ← Wrap in response struct { Streams []LiveStream, Total int }
  ← JSON 200
```

**Repository methods reused:** `streams.domain.StreamRepository.ListLiveStreams` — needs two changes:
1. Add `WHERE last_seen >= now() - interval '60 seconds'` to the viewer count subquery (PM spec: "viewerCount is the count of distinct client_id values in stream_viewers where last_seen is within the last 60 seconds").
2. The subquery currently uses `COUNT(*)` — this is correct because the `PRIMARY KEY (stream_id, client_id)` ensures uniqueness. No `DISTINCT` needed.

**Changes needed:**
- Handler: wrap response in `{"streams": streams, "total": len(streams)}`.
- Repository: add 60-second recency filter to viewer count subquery.
- Domain: consider adding `ThumbnailUrl` field to `LiveStream` (see Architectural Finding #2).

#### 3.3 `GET /api/me/analytics` — Data Flow

```
Handler (GetAnalytics)
  → Validate period param (default "week", reject invalid)
  → streamOps.GetAnalytics(ctx, userID, period)
    → StreamService.GetAnalytics(ctx, userID, period)
      → streamsRepo.GetAnalytics(ctx, userID, period)
        → SQL:
          SELECT COALESCE(SUM(total_seconds),0),
                 COALESCE(MAX(peak_viewers),0),
                 COALESCE(SUM(unique_viewers),0),
                 COUNT(*)
          FROM stream_analytics
          WHERE user_id = $1 AND <date_condition>
        ← Populate domain.Analytics with startDate/endDate
      ← *domain.Analytics
  ← JSON 200
```

**Date range computation (startDate/endDate):**
| Period | `startDate` | `endDate` | SQL `date_condition` |
|---|---|---|---|
| `"week"` | Monday 00:00:00 UTC of current week | Sunday 23:59:59 UTC of current week | `date >= date_trunc('week', now())::date` |
| `"month"` | 1st day of current month 00:00:00 UTC | Last day of current month 23:59:59 UTC | `date >= date_trunc('month', now())::date` |
| `"all"` | Date of user's first stream (`MIN(date)`) | Today's date | `true` (no filter) |

**Repository methods reused:** `streams.domain.StreamRepository.GetAnalytics` — needs to compute and return `StartDate`/`EndDate` in the `Analytics` struct. The `all` period requires an additional query for `MIN(date)` or can use a COALESCE in a single query.

**Changes needed:**
- `domain.Analytics`: add `StartDate string \`json:\"startDate\"\`` and `EndDate string \`json:\"endDate\"\``.
- Repository: compute start/end dates. For `"week"` and `"month"`, use `date_trunc` + Go `time` package. For `"all"`, query `SELECT COALESCE(MIN(date), CURRENT_DATE) FROM stream_analytics WHERE user_id = $1`.
- Handler: add `period` validation (reject anything not `week`/`month`/`all`). The current handler already defaults to `"week"` but does not validate.

#### 3.4 `POST /api/me/stream-key/regenerate` — Data Flow

Already implemented. No design changes.

```
Handler (RegenerateStreamKey)
  → Validate confirm body
  → authService.RegenerateStreamKey(ctx, userID)
    → authService.GenerateStreamKey()       — crypto/rand 32 bytes + SHA-256
    → authRepo.UpdateStreamKey(ctx, userID, hash)
      → UPDATE users SET stream_key_hash = $2, updated_at = now() WHERE id = $1
    ← rawKey (string)
  ← JSON 200 { streamKey: rawKey }
```

#### 3.5 `PATCH /api/me/settings` — Data Flow

```
Handler (UpdateSettings)
  → Validate streamTitle (required, 1-100 chars, non-blank)
  → Validate streamCategory (optional, max 100 chars)
  → authService.UpdateSettings(ctx, userID, title, category)
    → authRepo.UpdateSettings(ctx, userID, title, category)
      → UPDATE users SET stream_title = $2, stream_category = $3, updated_at = now() WHERE id = $1
  ← nil error
  ← JSON 200 { streamTitle, streamCategory }
```

**Changes needed:**
- Handler: change response from `{"status":"ok"}` to `{"streamTitle": title, "streamCategory": category}`. Note: when category is cleared, it should be `null` in JSON (use `*string` type, pass `nil`).

#### 3.6 `POST /api/me/stream/end` — Data Flow

```
Handler (ForceEndStream)
  → streamOps.ForceEndStream(ctx, userID)
    → StreamService.ForceEndStream(ctx, userID)
      1. stream, err := repo.GetStreamByUserID(ctx, userID)
         → SELECT ... FROM streams WHERE user_id = $1 AND status = 'live'
         → If no rows: return errs.Conflict("no active stream to end")
      2. [NEW] Call SRS HTTP API:
         → DELETE {SRS_API_URL}/api/v1/clients/{stream.SRSClientID}
         → If SRS fails: log error, continue (graceful degradation)
      3. duration := int(time.Since(stream.StartedAt).Seconds())
      4. repo.EndStream(ctx, stream.ID, "", "", duration)
         → UPDATE streams SET ended_at=now(), status='offline', duration_seconds=$4 WHERE id=$1
      5. authRepo.SetLiveStatus(ctx, userID, false)
         → UPDATE users SET is_live=false, live_since=NULL WHERE id=$1
      6. [DEFER] Post-stream analytics update (async or inline)
    ← nil error (or SRS-failure-tolerated)
  ← JSON 200 { status: "offline", message: "..." }
```

**Repository methods reused:**
- `streams.domain.StreamRepository.GetStreamByUserID` — existing, returns the active (status='live') stream
- `streams.domain.StreamRepository.EndStream` — existing
- `auth.domain.Repository.SetLiveStatus` — existing (via `AuthRepo` interface in streams module)

**New dependencies needed:**
- SRS HTTP client. The `StreamService` needs the SRS API base URL (e.g., `http://srs:1985`). See Section 5 for integration design.

---

### 4. Error Mapping

| Domain / Service Error | HTTP Status | Body |
|---|---|---|
| No JWT present, or JWT invalid/expired | 401 | `{"error": "unauthorized"}` |
| `period` parameter not in `{week, month, all}` | 400 | `{"error": "invalid period: <value>; expected week, month, or all"}` |
| `streamTitle` missing, empty, or whitespace | 400 | `{"error": "streamTitle is required and must be 1-100 characters"}` |
| `streamTitle` > 100 chars | 400 | `{"error": "streamTitle is required and must be 1-100 characters"}` |
| `streamCategory` > 100 chars | 400 | `{"error": "streamCategory must be 100 characters or fewer"}` |
| `confirm: false` or unrecognized fields in regenerate body | 400 | `{"error": "must confirm stream key regeneration"}` |
| Malformed JSON body | 400 | `{"error": "invalid JSON: <detail>"}` |
| User is not live when calling force-end | 409 | `{"error": "no active stream to end"}` |
| SRS API unreachable during force-end | 200 | `{"status": "offline", "message": "Stream ended (publisher disconnect may have failed)"}` |
| Database query failure | 500 | `{"error": "internal server error"}` |

**Error type → HTTP mapping (existing, from `render.Error`):**
```
errs.KindBadRequest   → 400
errs.KindUnauthorized → 401
errs.KindForbidden    → 403
errs.KindNotFound     → 404
errs.KindConflict     → 409
(other)               → 500
```

**Key design rule:** The `ForceEndStream` service must convert the "not found" from `GetStreamByUserID` (which returns `errs.NotFound`) into `errs.Conflict` before surfacing to the handler. The handler should never receive a `NotFound` for "no active stream" — that's a business rule violation (409 Conflict), not a resource-not-found (404).

---

### 5. SRS Integration for `POST /api/me/stream/end`

#### 5.1 SRS HTTP API

SRS exposes an HTTP API on port 1985 by default. The endpoint to disconnect a publisher is:

```
DELETE http://{SRS_HOST}:1985/api/v1/clients/{client_id}
```

The `client_id` corresponds to `streams.srs_client_id`, which is set during the `on_publish` callback (see `StreamService.OnStreamStart`).

#### 5.2 Integration Architecture

```
┌──────────────────────────────────────────┐
│ StreamService.ForceEndStream(ctx, userID)│
│                                           │
│ 1. Get active stream (repo)              │
│    ├─ Found → continue                   │
│    └─ Not found → errs.Conflict(409)     │
│                                           │
│ 2. Call SRS HTTP API                     │
│    ├─ Success → continue                 │
│    └─ Failure → log error, continue      │
│       (graceful degradation)             │
│                                           │
│ 3. Mark stream as ended (repo)           │
│                                           │
│ 4. Mark user as offline (authRepo)       │
│                                           │
│ 5. Return nil (SRS failure tolerated)     │
└──────────────────────────────────────────┘
```

#### 5.3 Configuration

New environment variable:

| Variable | Purpose | Default |
|---|---|---|
| `SRS_API_URL` | Base URL of the SRS HTTP API | `http://localhost:1985` |

This is separate from `SRS_CALLBACK_SECRET` (which is for inbound SRS callbacks). No authentication is needed for the SRS HTTP API in the default configuration (it's an internal network API).

#### 5.4 Dependency Injection

`StreamService` currently receives `(repo, authRepo, srsSecret)`. It must also receive `srsAPIURL`:

```go
func NewStreamService(
    repo domain.StreamRepository,
    authRepo domain.AuthRepo,
    srsSecret string,
    srsAPIURL string,  // NEW
) *StreamService
```

The `StreamService` will hold an `*http.Client` (or use `http.DefaultClient` with a reasonable timeout like 5 seconds). The SRS call is fire-and-forget in terms of request success (we log the error but don't block the database transaction).

#### 5.5 SRS Call Timeout

The SRS HTTP call should have a **5-second timeout**. This prevents a hanging SRS from blocking the force-end flow indefinitely. If SRS is unreachable after 5 seconds, we proceed with the database changes and return the degraded-success message.

#### 5.6 Post-Stream Analytics

The PM spec mentions triggering analytics computation after stream end. The current design has two options:

**Chosen: Inline analytics update in `ForceEndStream`** — after ending the stream, compute daily analytics from the just-ended stream's metrics and upsert into `stream_analytics`. This reuses the same `UpdateStreamAnalytics` call already used in `OnStreamEnd`.

**Rejected: Async job queue** — adds infrastructure complexity (Redis, worker process) not justified for v1. The analytics update is a single UPSERT and completes in <10ms.

**Rejected: Defer to `OnStreamEnd` callback** — when we force-end via the SRS API (`DELETE /api/v1/clients/{id}`), SRS fires the `on_unpublish` callback as a side effect, which triggers `OnStreamEnd`. Relying on this creates a fragile coupling. Instead, `ForceEndStream` explicitly calls `UpdateStreamAnalytics` so the analytics update is deterministic.

---

### 6. Technology Decisions

#### 6.1 SRS HTTP Client

**Chosen:** `net/http` standard library with a 5-second timeout via `context.WithTimeout`. The SRS API call is a single `DELETE` request — no need for a third-party HTTP client library.

```
Decision: Use stdlib net/http for SRS API calls
Rationale: Single endpoint (DELETE /api/v1/clients/{id}), no retry logic needed.
           The project already uses net/http in AuthService for Google OAuth.
Rejected: go-retryablehttp (unnecessary — we don't retry)
Rejected: resty (adds dependency for a single call)
```

#### 6.2 Response Wrapper for `GET /api/streams/live`

**Chosen:** Inline response struct in the handler (`liveStreamsResponse{ Streams []LiveStream, Total int }`). Matches the existing pattern used by `getMeResponse` in the auth handler.

```
Decision: Inline struct for the response wrapper
Rationale: Simple, follows existing pattern, no new domain types needed.
Rejected: Domain type for the wrapper (adds a domain entity that's just a JSON shape,
          and the wrapper is specific to this HTTP response, not a domain concept)
```

#### 6.3 Date Computation for Analytics

**Chosen:** Compute `startDate` and `endDate` in the repository layer, using Go's `time` package for week/month boundaries and a SQL `MIN(date)` for the "all" period. Return them as `string` fields (formatted as `"2006-01-02"`).

```
Decision: Repository computes startDate/endDate
Rationale: The repository already owns the date-condition SQL. Computing dates
           at the same layer keeps the service layer clean.
Rejected: Service layer computes dates (would require the service to know SQL
          date-trunc semantics — that's a repository concern)
Rejected: PostgreSQL-only computation via extra SELECT columns (adds query
          complexity; Go's time package is clearer for week/month boundaries)
```

#### 6.4 Force-End Error Handling (409 vs 404)

**Chosen:** `errs.Conflict` (HTTP 409) when the user has no active stream to force-end.

```
Decision: 409 Conflict for "no active stream to end"
Rationale: The request itself is valid (authenticated, well-formed), but the
           resource state prevents the operation. 409 is the correct semantic
           for "the request could not be completed due to a conflict with the
           current state of the resource" (RFC 7231, Section 6.5.8).
Rejected: 404 Not Found (implies the user or endpoint doesn't exist — misleading)
Rejected: 422 Unprocessable Entity (not in the project's error kind enum, and
           the existing render.Error mapper doesn't support it)
```

#### 6.5 Stream Key in GET /api/me

**Chosen:** Return `null` (omit via `omitempty`). The raw key cannot be recovered from the SHA-256 hash.

```
Decision: streamKey is null/omitted in GET /api/me
Rationale: The database stores only stream_key_hash (SHA-256). Hashing is
           one-way by design — the raw key is only available at creation and
           regeneration time. Returning the hash would be a security issue
           (exposes hash to brute-force). The frontend type marks it optional.
Rejected: Storing the raw key (defeats the purpose of hashing)
Rejected: Encrypting the raw key with a reversible cipher (adds key management
           complexity for a field the frontend only needs at regeneration time)
```

#### 6.6 Analytics `period` Validation

**Chosen:** Validate in the handler (HTTP layer), before calling the service. Reject with 400 if the value is not in `{week, month, all}`.

```
Decision: Handler validates period enum
Rationale: Input validation is an HTTP concern. The service layer should receive
           a pre-validated period value and not worry about invalid enums.
Rejected: Validate in the service layer (service would need to return validation
          errors alongside domain errors — muddies the service interface)
Rejected: Validate in the repository (SQL would need CASE/error handling)
```

#### 6.7 Route Registration

**Chosen:** `GET /api/streams/live` stays in `StreamHandler.RegisterRoutes` (public, no auth). The five `/api/me/*` routes stay in `AuthHandler.RegisterRoutes` (protected, under `AuthMiddleware` group).

```
Decision: Keep current module split — auth handler for /api/me/*, streams handler
          for /api/streams/live
Rationale: Already wired this way. Changing it would require duplicating
           AuthMiddleware in the streams handler or breaking the route grouping.
           The auth handler's streamOps interface already cleanly separates
           concerns (handler owns auth/routing, service owns business logic).
Rejected: Moving all six to a single handler (violates module boundaries)
Rejected: Creating a new "me" module (over-engineering — these are thin
          pass-through handlers with no shared domain logic beyond what
          auth + streams already provide)
```

---

### 7. Architectural Findings (from code inspection)

These are issues discovered while inspecting the existing code that the implementation must address:

| # | Finding | Severity | Resolution |
|---|---|---|---|
| 1 | `ListLiveStreams` returns bare array, not `{streams, total}` wrapper | High | Wrap in response struct in handler |
| 2 | `LiveStream.HlsPath` → `json:"thumbnailUrl"` is semantically wrong | Medium | Defer to v2. For v1, keep mapping but document the hack. Add `ThumbnailUrl` field to domain entity for future use. |
| 3 | `Analytics` domain entity missing `startDate`/`endDate` fields | High | Add fields + populate in repository |
| 4 | `ForceEndStream` returns 404 for "no active stream" (should be 409) | High | Convert `errs.NotFound` → `errs.Conflict` in service layer |
| 5 | `ForceEndStream` has no SRS integration | High | Add SRS HTTP call as designed in Section 5 |
| 6 | `GetMe` response includes `createdAt` (not in frontend contract) | Medium | Remove from response struct |
| 7 | `GetMe` response missing `streamCategory` field | High | Add to response struct |
| 8 | `GetAnalytics` handler does not validate `period` parameter | Medium | Add enum validation in handler |
| 9 | `UpdateSettings` handler returns `{"status":"ok"}` instead of echoing values | High | Change response to `{streamTitle, streamCategory}` |
| 10 | `ListLiveStreams` viewer count query lacks 60-second recency filter | High | Add `WHERE last_seen >= now() - interval '60 seconds'` to subquery |
| 11 | `GetMe` handler has no `streamCategory` in response struct | High | Duplicate of #7 — same fix |

---

### 8. Design Non-Goals

These are design decisions we explicitly chose NOT to make in this spec:

- ❌ **New database migrations.** The `users`, `streams`, `stream_viewers`, and `stream_analytics` tables already support all six endpoints. No DDL changes are needed.
- ❌ **New middleware.** The existing `AuthMiddleware` covers authentication. No rate limiting, CORS changes, or request logging middleware is added for these endpoints.
- ❌ **Thumbnail generation pipeline.** The `thumbnailUrl` field in `LiveStream` is acknowledged as architecturally incomplete but its implementation is deferred to a separate spec.
- ❌ **WebSocket or SSE for viewer counts.** The frontend polls `GET /api/streams/live` — this design does not introduce push-based updates.
- ❌ **Pagination for `GET /api/streams/live`.** The PM spec explicitly excludes it. The `total` field exists for forward-compatibility but is always `len(streams)`.
- ❌ **Async job queue for post-stream analytics.** The analytics upsert is synchronous in the force-end flow — fast enough (<10ms) to not justify infrastructure overhead.
- ❌ **SRS authentication for the HTTP API.** The default SRS configuration exposes the HTTP API without auth on the internal network. If this changes, a follow-up spec should add SRS API token support.
- ❌ **Stream key encryption at rest.** The current SHA-256 hashing is sufficient for v1. Reversible encryption would add key management complexity without a clear v1 use case.
- ❌ **Audit logging for stream-key regeneration or force-end.** These are security-sensitive operations but formal audit trails are out of scope for v1.
- ❌ **New Go dependencies.** All six endpoints are implemented with existing dependencies: `net/http`, `crypto/rand`, `crypto/sha256`, `encoding/json`, `github.com/go-chi/chi/v5`, `github.com/jackc/pgx/v5`.

---

## Task Checklist

### Phase 1 — Fix Handler Response Shapes (no new logic)

> **Goal:** Make existing handler stubs return the correct JSON shapes that match `frontend/src/types/index.ts`. No new service/repository logic required.
> **Can start immediately.**

1. [x] **(Backend)** Fix `GET /api/me` response struct
   - Add `StreamCategory *string \`json:"streamCategory"\`` to `getMeResponse`
   - Remove `CreatedAt` field (not in frontend `User` type)
   - Confirm `StreamKey *string \`json:"streamKey,omitempty"\`` returns `null`/omitted (raw key not recoverable from hash)
   → Satisfies: US1 AC — frontend `User` type match
   → Files: `backend/internal/modules/auth/adapter/http/handler.go`

2. [x] [P] **(Backend)** Fix `PATCH /api/me/settings` response
   - Change from `{"status":"ok"}` to `{"streamTitle": title, "streamCategory": category}`
   - When category is cleared, return `null` (use `*string` type, pass `nil`)
   → Satisfies: US5 AC — frontend expects echoed fields
   → Files: `backend/internal/modules/auth/adapter/http/handler.go`

3. [x] [P] **(Backend)** Fix `GET /api/streams/live` response wrapper (if handler exists)
   - Wrap bare array in `{"streams": streams, "total": len(streams)}` response struct
   - If handler doesn't exist yet, defer to Task 8
   → Satisfies: US2 AC — frontend `LiveStreamsResponse` type match
   → Files: `backend/internal/modules/streams/adapter/http/handler.go`

### Phase 2 — Wire StreamService (analytics + force-end)

> **Goal:** Implement the `streamOps` interface methods that the auth handler already calls. These currently panic or return stubs.
> **Can start after Phase 1 (need correct response shapes for test fixtures).**

4. [x] **(Backend)** Implement `StreamRepository.GetAnalytics`
   - SQL query aggregating `stream_analytics` table: `SUM(total_seconds)`, `MAX(peak_viewers)`, `SUM(unique_viewers)`, `COUNT(*)`
   - Compute `startDate`/`endDate` based on period (week: Monday-Sunday UTC; month: calendar month; all: MIN(date) to today)
   - Add `StartDate` and `EndDate` fields to `domain.Analytics`
   → Satisfies: US3 AC — period-based date ranges
   → Files: `backend/internal/modules/streams/adapter/postgres/repo.go`, `backend/internal/modules/streams/domain/entity.go`

5. [x] **(Backend)** Wire `GET /api/me/analytics` through service layer
   - Implement `StreamService.GetAnalytics(ctx, userID, period)` → delegates to repository
   - Verify default period `"week"` when query param is empty
   → Satisfies: US3 AC — analytics endpoint returns correct data
   → Files: `backend/internal/modules/streams/application/service.go`

6. [x] **(Backend)** Implement `StreamRepository.ForceEndStream` + SRS integration
   - `GetStreamByUserID(ctx, userID)` — return active stream (status='live') or error
   - SRS HTTP call: `DELETE {SRS_API_URL}/api/v1/clients/{srs_client_id}` with 5s timeout
   - On SRS failure: log error, continue (graceful degradation)
   - Set `streams.status='offline'`, `streams.ended_at=now()`, compute `duration_seconds`
   - Set `users.is_live=false`, `users.live_since=NULL`
   - Upsert daily analytics row in `stream_analytics`
   → Satisfies: US6 AC — force-end with SRS disconnect + DB state transitions
   → Files: `backend/internal/modules/streams/adapter/postgres/repo.go`

7. [x] **(Backend)** Wire `POST /api/me/stream/end` through service layer
   - Implement `StreamService.ForceEndStream(ctx, userID)` → delegates to repository
   - Convert `errs.NotFound` from `GetStreamByUserID` to `errs.Conflict` (409) when no active stream
   - Return appropriate `message` field: `"Stream ended"` on success, `"Stream ended (publisher disconnect may have failed)"` on SRS failure
   → Satisfies: US6 AC — 409 for no active stream, 200 with graceful degradation message
   → Files: `backend/internal/modules/streams/application/service.go`

### Phase 3 — New Endpoint: List Live Streams

> **Goal:** Build `GET /api/streams/live` from scratch. Public endpoint, no auth required.
> **Can start in parallel with Phase 2 (different files, no shared state).**

8. [x] [P] **(Backend)** Build `GET /api/streams/live` — handler + service + repository
   - Handler: register in `StreamHandler.RegisterRoutes` (public, no auth middleware)
   - Service: `StreamService.ListLive(ctx)`
   - Repository: SQL joining `streams` + `users` + viewer count subquery
   - Viewer count subquery: `COUNT(*)` from `stream_viewers` WHERE `last_seen >= now() - interval '60 seconds'` grouped by `stream_id`
   - Response: `{"streams": [...], "total": N}` (not bare array)
   - `thumbnailUrl`: for v1, map `streams.hls_path` (document this hack — see Architectural Finding #2)
   → Satisfies: US2 AC — homepage live grid data
   → Files: `backend/internal/modules/streams/adapter/http/handler.go`, `.../application/service.go`, `.../adapter/postgres/repo.go`

### Phase 4 — Validation & Hardening

> **Goal:** Add input validation, fix error codes, add missing query filters.
> **Can start after Phase 2 (validation sits in handlers that are now wired).**

9. [x] [P] **(Backend)** Add `period` enum validation to `GET /api/me/analytics` handler
   - Reject with 400 if value is not `week`, `month`, or `all`
   - Default to `"week"` if empty (already done)
   → Satisfies: US3 AC — invalid period returns 400
   → Files: `backend/internal/modules/auth/adapter/http/handler.go` (GetAnalytics)

10. [x] [P] **(Backend)** Add 60-second recency filter to viewer count query
    - In `GET /api/streams/live` viewer count subquery: `WHERE last_seen >= now() - interval '60 seconds'`
    - Covered by Task 8 if implemented correctly; verify here
    → Satisfies: US2 AC — viewerCount is within last 60 seconds
    → Files: `backend/internal/modules/streams/adapter/postgres/repo.go`

11. [x] [P] **(Backend)** Add max-length validation to `title` and `category` in `PATCH /api/me/settings`
    - `streamTitle`: reject empty/whitespace, enforce 1-100 chars
    - `streamCategory`: enforce max 100 chars (optional, null allowed)
    - Reject with descriptive 400 error messages
    → Satisfies: US5 AC — validation on settings update
    → Files: `backend/internal/modules/auth/adapter/http/handler.go` (UpdateSettings)

### Phase 5 — Tests

> **Goal:** Table-driven tests for all six endpoints, covering happy path + error paths + edge cases.
> **Can start incrementally as each handler is stabilized.**

12. [x] [P] **(Backend)** Write tests for `GET /api/me`
    - Happy path: authenticated user returns full profile
    - Error: no auth (401), user not found (404)
    - Edge: null streamTitle/streamCategory/avatarUrl
    → Files: `backend/internal/modules/auth/adapter/http/handler_test.go`

13. [x] [P] **(Backend)** Write tests for `GET /api/streams/live`
    - Happy path: one or more live streams
    - Edge: zero live streams (empty array, total=0)
    - Edge: unauthenticated request (still returns 200, public endpoint)
    - Verify wrapper shape: `{streams: [...], total: N}`
    → Files: `backend/internal/modules/streams/adapter/http/handler_test.go`

14. [x] [P] **(Backend)** Write tests for `GET /api/me/analytics`
    - Happy path: period=week, period=month, period=all
    - Edge: never streamed (all zeros)
    - Error: invalid period (400), no auth (401)
    → Files: `backend/internal/modules/auth/adapter/http/handler_test.go`

15. [x] [P] **(Backend)** Write tests for `POST /api/me/stream-key/regenerate`
    - Happy path: empty body, returns new key
    - Happy path: confirm:true body, returns new key
    - Error: confirm:false (400), malformed JSON (400), no auth (401)
    → Files: `backend/internal/modules/auth/adapter/http/handler_test.go`

16. [x] [P] **(Backend)** Write tests for `PATCH /api/me/settings`
    - Happy path: valid title + category
    - Happy path: only title (category omitted → cleared to null)
    - Error: empty title (400), title > 100 chars (400), category > 100 chars (400), no auth (401)
    → Files: `backend/internal/modules/auth/adapter/http/handler_test.go`

17. [x] [P] **(Backend)** Write tests for `POST /api/me/stream/end`
    - Happy path: user is live → stream ends, returns 200
    - Error: user not live → returns 409
    - Error: no auth (401)
    - Edge: SRS unreachable → still returns 200 with warning message
    → Files: `backend/internal/modules/auth/adapter/http/handler_test.go`

### Phase 6 — Build Verification & Handoff

> **Goal:** Confirm everything builds, all tests pass, announce stable contracts.

18. [x] **(Backend)** Run build + full test suite
    - `cd backend && go build ./...`
    - `cd backend && go test ./... -count=1`
    - Fix any compilation errors or test failures
    → Satisfies: build discipline (AGENTS.md Section 4.1)

19. [x] **(Backend)** Announce stable API contracts
    - Confirm all six endpoints match the API contract shapes in Section 2 of this spec
    - Confirm response shapes match `frontend/src/types/index.ts`
    - Announce: "All six endpoints are live and stable. Frontend can consume."

---

### Parallel Execution Plan

```
Phase 1 (fix shapes) ──▶ Phase 2 (wire services) ──▶ Phase 4 (validation) ──▶ Phase 5 (tests) ──▶ Phase 6 (verify)
         │                       │
         │                       └── Task 8 (GET /api/streams/live) runs in parallel with Phase 2
         │
         └── Tasks 1-3 are parallelizable with each other

Phase 5 (tests): Tasks 12-17 are all [P] — each endpoint's tests are independent.
                 Start writing tests for an endpoint as soon as its handler is stable
                 (don't wait for all endpoints to be done).
```

**[P]** = Can be parallelized with other [P] tasks in the same phase.

**Role assignments:**
- All tasks: Backend Engineer
