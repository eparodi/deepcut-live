# US3 — Viewing Live Streams

**Status:** Design
**Parent:** `_shared.md`
**Depends on:** US2 (streams must be live to view)

## User Story

As a viewer, I want to browse live streams and watch one so that I can
discover content in real time.

## Acceptance Criteria

- Given I am on the homepage, When there are active streams, Then I see a grid of live channels with thumbnail, streamer name, title, and viewer count
- Given I am on the homepage, When I click on a live channel, Then I navigate to `/channel/<username>` where the HLS player loads and starts playing within 3 seconds
- Given I am watching a stream, When the streamer is active, Then the video plays with no more than 5 seconds of latency from the live broadcast
- Given there are no live streams, When I visit the homepage, Then I see an empty state: "No one is live right now. Check out past streams below."

## Edge Cases

- What happens if the viewer's browser doesn't support HLS? (Show "Browser not supported" with link to Chrome/Firefox/Edge)
- What happens if the stream ends while a viewer is watching? (Player shows "Stream ended" overlay, VOD link appears)
- What happens if 1,000 viewers watch simultaneously? (HLS scales; SRS handles the segment distribution)
- What happens on mobile browsers? (HLS works in mobile Safari and Chrome; responsive layout adapts)

---

## API Contract

All endpoints already defined in US2 (`GET /api/streams/live`, `GET /api/channel/:username`).

No new API endpoints needed for this story — it consumes what US2 provides.

---

## Viewer Count Protocol

Viewer count is tracked client-side via periodic heartbeat:

```
Client: POST /api/streams/:streamId/viewer-heartbeat
         (no body, JWT cookie optional)
         → sent every 30 seconds while video is playing

Server: UPSERT stream_viewers SET last_seen = now()
         WHERE stream_id = <id> AND client_id = <session_or_user_id>

Server (GET /api/streams/live response): 
         SELECT COUNT(*) FROM stream_viewers
         WHERE stream_id = <id> AND last_seen > now() - interval '2 minutes'
```

---

## Implementation Notes

- HLS is served directly by SRS, not proxied through the Go backend. The frontend `<video>` tag points to the SRS origin.
- Frontend uses [hls.js](https://github.com/video-dev/hls.js) for HLS playback in browsers that don't natively support it (Chrome, Firefox).
- Safari has native HLS support — no library needed.
- The heartbeat endpoint is lightweight (single upsert). No auth required for anonymous viewers.

---

## UI Design

### Screen: Homepage (Live Grid)

**Purpose:** Browse currently live streams. First thing viewers see.

**Layout:**
```
┌─────────────────────────────────────────────────────┐
│  [🔍 Search past streams...]        [Sign In]       │
├─────────────────────────────────────────────────────┤
│                                                     │
│  🔴 LIVE NOW                          [Grid ▦] [List ≡] │
│                                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐         │
│  │ ░░░░░░░░░ │  │ ░░░░░░░░░ │  │ ░░░░░░░░░ │         │
│  │ ░ THUMB ░ │  │ ░ THUMB ░ │  │ ░ THUMB ░ │         │
│  │ ░░░░░░░░░ │  │ ░░░░░░░░░ │  │ ░░░░░░░░░ │         │
│  │           │  │           │  │           │         │
│  │ 🔴 42     │  │ 🔴 137    │  │ 🔴 9      │         │
│  │ Alice     │  │ Bob       │  │ Carol     │         │
│  │ Coding    │  │ Music     │  │ Podcast   │         │
│  └──────────┘  └──────────┘  └──────────┘         │
│                                                     │
│  Empty state (no live streams):                     │
│  ┌─────────────────────────────────────────────┐    │
│  │  🎬 No one is live right now                 │    │
│  │  Check out past streams below                │    │
│  └─────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────┘
```

**Components:**

`LiveStreamCard`
- Thumbnail (auto-generated from first HLS frame)
- Red "🔴 LIVE" indicator in top-left corner
- Viewer count badge: "👁 42"
- Streamer name + avatar
- Stream title (truncated to 2 lines)
- Category pill (e.g., "Programming")
- Entire card is clickable → `/channel/<streamerId>`

**States:**
| State | What renders |
|-------|-------------|
| Loading (first load) | 8 skeleton cards (pulsing grey rectangles) |
| Empty | "No one is live" + link to past streams |
| Error | "Could not load streams" + retry button |
| Populated | Grid of LiveStreamCards, 4 per row on desktop, 2 on mobile |
| Stream ends while viewing | Card fades out with 300ms animation |
| New stream starts | Card fades in at top of grid |

**Accessibility:**
- Each card: `role="link"`, `aria-label="Alice is live: Late night coding session. 42 viewers"`
- Grid: `role="list"`, cards: `role="listitem"`
- Live indicator: `aria-label="Live"` (not just the red dot)

---

### Screen: Channel Page (Live)

**Purpose:** Watch a live stream with chat.

**Layout:**
```
┌──────────────────────────────────────────────────────┐
│  [🔍 Search...]  [Back to Live]     [Sign In]        │
├────────────────────────────────┬─────────────────────┤
│                                │  💬 CHAT            │
│                                │                     │
│        VIDEO PLAYER            │  Alice: great!      │
│        (HLS via hls.js)        │  Bob: lol           │
│                                │  Carol: 🔥          │
│                                │                     │
│                                │  ┌───────────────┐  │
│  🔴 LIVE                       │  │ Type message...│  │
│  Late night coding session      │  │          [Send]│  │
│  Alice · 42 viewers             │  └───────────────┘  │
│  Programming                    │                     │
└────────────────────────────────┴─────────────────────┘
  Mobile: full-width video, chat as bottom sheet
```

**Components:**

`VideoPlayer`
- Uses hls.js (Chrome/Firefox) or native HLS (Safari)
- Full-width, 16:9 aspect ratio
- Standard controls: play/pause, volume, fullscreen
- "Stream ended" overlay when stream goes offline (with link to VOD)
- Auto-plays on page load (muted initially for autoplay policy)

`StreamInfo`
- Streamer avatar + name
- Stream title
- Viewer count (updates every 30s via heartbeat)
- Category pill

**States:**
| State | Video Player | Chat | StreamInfo |
|-------|-------------|------|------------|
| Loading | Spinner on dark background | "Loading chat..." | Skeleton text |
| Live | Playing HLS | Active chat (see US4) | All info populated |
| Stream Interrupted | Dark overlay "Stream Interrupted — reconnecting..." | Chat still active | "Stream Interrupted" |
| Stream Ended | Dark overlay "Stream ended — watch VOD" + link | Chat read-only | "Offline" |
| Error | "Could not load stream" + retry | Hidden | Error message |

**Accessibility:**
- Video: standard `<video>` controls (keyboard accessible)
- Chat: see US4
- Stream title: `<h1>`
- Layout resizes: video uses `aspect-ratio: 16/9`, chat panel is `minmax(300px, 1fr)` on desktop

**Responsive:**
- Desktop (≥1024px): video left (70%), chat right (30%)
- Mobile (<640px): video full-width, chat below as collapsible panel
