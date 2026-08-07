# US6 — Streamer Dashboard

**Status:** Design
**Parent:** `_shared.md`
**Depends on:** US1 (streamer account), US2 (for live analytics)

## User Story

As a streamer, I want to manage my stream settings and see basic
analytics so that I can control how my stream appears.

## Acceptance Criteria

- Given I am on my dashboard, When I set a stream title and category, Then it updates and appears on my channel page immediately
- Given I am on my dashboard, When I view my analytics panel, Then I see: total stream time (this week), peak concurrent viewers, total unique viewers
- Given my stream is live, When I click "End Stream" on my dashboard, Then the RTMP connection is terminated and the stream ends

## Edge Cases

- What happens if an empty title is set? (Client-side validation: title is required, 1-100 characters)
- What happens if "End Stream" is clicked but the stream has already ended? (Show "Stream already offline" and refresh the page)
- What happens if the analytics query is slow with many streams? (Cache analytics in a materialized view or pre-computed table; refresh every 5 minutes)

---

## API Contract

### PATCH /api/me/settings

**Purpose:** Update stream title and category.

**Authentication:** Bearer cookie (JWT).

**Request:**
```json
{
  "streamTitle": "Late night coding session",
  "streamCategory": "Programming"
}
```

**Validation:**
- `streamTitle`: required, 1-100 characters
- `streamCategory`: optional, max 100 characters

**Success Response 200:**
```json
{
  "streamTitle": "Late night coding session",
  "streamCategory": "Programming"
}
```

**Error Responses:**
- 400: Validation error (title empty or >100 chars)
- 401: Not authenticated

---

### GET /api/me/analytics

**Purpose:** Get streamer's analytics for this week.

**Authentication:** Bearer cookie (JWT).

**Query params:**
```
?period=week    (week | month | all)
```

**Success Response 200:**
```json
{
  "period": "week",
  "startDate": "2026-08-01",
  "endDate": "2026-08-07",
  "totalStreamTimeSeconds": 43200,
  "peakViewers": 142,
  "totalUniqueViewers": 1205,
  "totalStreams": 5
}
```

**Error Responses:**
- 401: Not authenticated

---

### POST /api/me/stream/end

**Purpose:** Force-end the current live stream.

**Authentication:** Bearer cookie (JWT).

**Request body:** Empty (no parameters).

**Success Response 200:**
```json
{
  "status": "offline",
  "message": "Stream ended"
}
```

**Error Responses:**
- 401: Not authenticated
- 409: No active stream to end

**Side effects:**
- Calls SRS API: `DELETE http://srs:8080/api/v1/clients/<client_id>` to disconnect the RTMP publisher
- Sets `users.is_live = false`
- Sets `streams.status = 'offline'`, `streams.ended_at = now()`
- Triggers VOD processing

---

## Implementation Notes

- Analytics are pre-computed in `stream_analytics` table. A cron job (or post-stream hook) updates them after each stream ends. The GET endpoint reads from this table, not from raw `streams` + `stream_viewers` joins.
- Force-end requires SRS HTTP API access. Backend calls SRS to disconnect the publisher by client ID. SRS client ID is stored in the `streams` row on `on_publish`.
- Analytics "this week" = Monday 00:00:00 UTC to Sunday 23:59:59 UTC of the current week.

---

## UI Design

Adds to the Dashboard screen designed in US1.

### Component: AnalyticsCards

**Purpose:** Show weekly streaming stats in a row of cards.

```
┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
│ 12h 30m  │ │   142    │ │  1,205   │ │    5     │
│  stream  │ │   peak   │ │  unique  │ │ streams  │
│   time   │ │ viewers  │ │ viewers  │ │ this wk  │
└──────────┘ └──────────┘ └──────────┘ └──────────┘
```

- 4 cards in a row on desktop, 2×2 grid on mobile
- Each card: large number (var(--text-2xl)), muted label below (var(--text-xs), var(--color-text-muted))
- Number formatted: 1205 → "1.2k", 14205 → "14.2k", 1234567 → "1.2M"
- Cards animate on load (fade up, stagger 50ms each)

**States:**
| State | What renders |
|-------|-------------|
| Loading | 4 skeleton cards (grey rectangles) |
| Empty (never streamed) | "Start streaming to see analytics" single card |
| Error | "Analytics unavailable — try again" |
| Populated | 4 cards with formatted numbers |

### Component: ForceEndButton

**Purpose:** Emergency stop for the current stream.

- Only visible when `isLive === true`
- Danger variant (red background: `var(--color-danger)`)
- Icon: ⏹ (stop square)
- Text: "End Stream"
- On click: confirmation dialog → call `POST /api/me/stream/end`

**Confirmation Dialog:**
- Title: "End Stream?"
- Body: "Your stream will end immediately. Viewers will see a 'Stream ended' screen. A recording will be saved."
- Cancel: "Keep Streaming" (primary)
- Confirm: "End Stream" (danger variant)

**Post-end state:**
- Button disappears
- Dashboard shows "Stream ended. Recording processing..."
- After recording is ready: link to VOD appears
