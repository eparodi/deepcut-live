# US5 — Past Streams / VOD

**Status:** Design
**Parent:** `_shared.md`
**Depends on:** US2 (streams must exist to have past streams)

## User Story

As a viewer, I want to search and watch past streams so that I can catch
up on content I missed or discover new streamers.

## Acceptance Criteria

- Given a stream has ended, When I visit the streamer's channel page, Then I see a list of their past streams with title, date, duration, and thumbnail
- Given I am on the homepage or search page, When I search by keyword, Then I see past streams matching the title or streamer name
- Given I click on a past stream, When the VOD page loads, Then the HLS recording plays back with standard video controls (play, pause, seek, volume)
- Given a stream has ended, When the recording is being processed, Then the VOD shows "Processing — available soon" for up to 5 minutes before becoming playable

## Edge Cases

- What happens if the VOD file is corrupted? (Show "This recording is unavailable" with a link back to the channel)
- What happens if SRS fails to write the recording file? (Log error, mark stream as "recording failed" in database)
- What happens if disk space runs out? (Monitor disk usage; alert before critical; oldest VODs become "unavailable" marked but not deleted)
- What happens with very long streams (24h+)? (HLS segments are fine; the VOD playlist file just references many segments)
- What happens if a streamer deletes their account? (VODs become orphaned and inaccessible within 30 days)

---

## API Contract

### GET /api/channel/:userId/vods

**Purpose:** List past streams (VODs) for a channel.

**Authentication:** None.

**Query params:**
```
?page=1&limit=20
```

**Success Response 200:**
```json
{
  "vods": [
    {
      "id": "uuid",
      "title": "Late night coding session",
      "category": "Programming",
           "startedAt": "2026-08-06T21:00:00Z",
      "durationSeconds": 5400,
      "thumbnailUrl": "/thumbnails/<vod-id>.jpg",
      "status": "ready"
    }
  ],
  "total": 50,
  "page": 1
}
```

**Error Responses:**
- 404: User not found

---

### GET /api/vods/:vodId

**Purpose:** Get VOD details including HLS playback URL.

**Authentication:** None.

**Success Response 200:**
```json
{
  "id": "uuid",
  "title": "Late night coding session",
  "category": "Programming",
  "streamerId": "uuid",
  "streamerName": "Alice Streamer",
  "streamerAvatarUrl": "https://...",
  "startedAt": "2026-08-06T21:00:00Z",
  "durationSeconds": 5400,
  "hlsUrl": "/hls/recordings/<vod-id>.m3u8",
  "viewerCount": 1337,
  "status": "ready"
}
```

**Processing state:**
```json
{
  "id": "uuid",
  "status": "processing",
  "message": "Processing — available soon"
}
```

**Error Responses:**
- 404: VOD not found
- 404: VOD recording failed (with `status: "failed"`)

---

### GET /api/search

**Purpose:** Search VODs by title or streamer name.

**Authentication:** None.

**Query params:**
```
?q=<search+terms>&page=1&limit=20
```

**Success Response 200:**
```json
{
  "results": [
    {
      "vodId": "uuid",
      "title": "Late night coding session",
      "streamerName": "Alice Streamer",
      "streamerAvatarUrl": "https://...",
      "startedAt": "2026-08-06T21:00:00Z",
      "durationSeconds": 5400,
      "thumbnailUrl": "/thumbnails/<vod-id>.jpg"
    }
  ],
  "total": 12,
  "page": 1
}
```

**Error Responses:**
- 400: Query parameter `q` is empty or missing

---

## Implementation Notes

- VOD processing: after `on_unpublish`, a background goroutine converts the HLS segments into a VOD `.m3u8` playlist (SRS already writes HLS fragments). Processing is just: validate all segments exist, generate a thumbnail from the first segment, update `recording_status='ready'`.
- Thumbnail generation: use `ffmpeg` to extract a frame at 10 seconds into the recording.
- Search: PostgreSQL `tsvector` with `GIN` index for full-text search across `streams.title` and `users.name`.
- HLS VODs are served directly by SRS from the recordings directory (not proxied through the Go backend).

---

## UI Design

### Screen: VOD Page

**Purpose:** Watch a past stream recording with synced chat replay.

**Layout:**
```
┌──────────────────────────────────────────────────────┐
│  [🔍 Search...]                    [Sign In]         │
├────────────────────────────────┬─────────────────────┤
│                                │  💬 CHAT REPLAY     │
│        VIDEO PLAYER            │                     │
│        (HLS VOD)               │  Alice: great!      │
│        [▶ play/pause]          │  Bob: lol           │
│        [═══●══════] seek        │  Carol: 🔥          │
│                                │                     │
│                                │  (scrolls with      │
│  📼 Late night coding session   │   video timeline)   │
│  Alice · Aug 6, 2026           │                     │
│  1h 30m · 1,337 views          │                     │
│  Programming                    │                     │
└────────────────────────────────┴─────────────────────┘
```

**Components:**

`VodPlayer`
- Same HLS player as live (hls.js / native)
- Standard controls: play/pause, seek bar, volume, fullscreen, playback speed (1x/1.5x/2x)
- Duration and current time displayed

`VodInfo`
- Stream title (h1)
- Streamer avatar + name (link to channel)
- Date + duration + view count
- Category pill

`ChatReplay`
- Chat panel (same design as US4 live chat)
- Synced to video: as video timeline advances, chat scrolls to matching timestamp
- Not interactive (read-only)

**States:**
| State | What renders |
|-------|-------------|
| Loading | Skeleton video player + "Loading recording..." |
| Ready | Full video player + chat replay |
| Processing | Large spinner + "Processing — available soon" message. No player controls. |
| Failed | "This recording is unavailable" + link back to channel |
| Not found | 404 page with "Stream not found" |

---

### Screen: Search

**Purpose:** Find past streams by keyword.

**Layout:**
```
┌──────────────────────────────────────────────────────┐
│  [🔍 coding session                ] [×]   [Sign In] │
├──────────────────────────────────────────────────────┤
│                                                      │
│  Results for "coding session" · 12 found             │
│                                                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │ ░░░░░░░░░ │  │ ░░░░░░░░░ │  │ ░░░░░░░░░ │          │
│  │ ░ THUMB ░ │  │ ░ THUMB ░ │  │ ░ THUMB ░ │          │
│  │ ░░░░░░░░░ │  │ ░░░░░░░░░ │  │ ░░░░░░░░░ │          │
│  │ Late night│  │ Morning   │  │ Weekend   │          │
│  │ Alice     │  │ Bob       │  │ Alice     │          │
│  │ 1h 30m    │  │ 2h 15m    │  │ 45m       │          │
│  │ Aug 6     │  │ Aug 5     │  │ Aug 4     │          │
│  └──────────┘  └──────────┘  └──────────┘          │
│                                                      │
│  Empty state:                                        │
│  ┌──────────────────────────────────────────────┐   │
│  │  🔍 No results for "xyz"                      │   │
│  │  Try different keywords or browse channels     │   │
│  └──────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────┘
```

**Components:**
- `SearchInput` — text input with magnifying glass icon, auto-focus on page load, clear button
- `VodResultCard` — thumbnail, title, streamer name + avatar, duration, date
- Click → `/vods/<vodId>`

**States:**
| State | What renders |
|-------|-------------|
| Input empty | List of recent VODs (default browse view) |
| Typing | Debounced search (300ms), loading indicator in input |
| Loading results | Skeleton cards × 8 |
| Empty | "No results" with keyword + suggestion to try different terms |
| Error | "Search unavailable — try again" |
| Populated | Grid of VodResultCards, 4 per row desktop, 2 mobile |
