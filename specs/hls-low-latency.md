# HLS Low-Latency Optimization

**Status:** Review
**Owner:** Eliseo
**Created:** 2026-08-10

## Context

The platform uses the standard RTMP → SRS → HLS pipeline (matching Twitch/Kick).
The existing spec (`live-streaming-platform.md`, US3 AC3) requires ≤5 seconds of
latency. The current SRS configuration produces standard HLS (`.ts` segments,
`hls_fragment 1s`, `hls_window 10`) without Low-Latency HLS extensions. The
hls.js player has `lowLatencyMode: true` but the server must also support LL-HLS
for it to be effective.

### Industry-Standard LL-HLS Parameters

| Parameter | Apple Recommendation | Twitch (observed) | **Our Target** |
|-----------|---------------------|-------------------|----------------|
| Full segment | 2-6s | 2s | **2s** |
| Partial segment | 200ms-1s | ~200ms | **200ms** |
| Partial segments/segment | — | ~10 | **10** |
| Target glass-to-glass latency | — | 2-5s | **≤5s** |

With 200ms partial segments and hls.js low-latency mode, the player can start
playback after buffering just 2-3 partial segments (~400-600ms) instead of
waiting for a full segment. Expected real-world glass-to-glass latency: 2-3
seconds on stable connections.

This spec covers the configuration and tuning changes needed to achieve ≤5s
latency end-to-end.

## Requirements

### User Story 1: Low-Latency HLS Delivery (US1)

As a viewer, I want the live stream to play with minimal delay so that I
experience the broadcast in near real-time, comparable to Twitch/Kick.

**Acceptance Criteria:**

- Given a streamer is broadcasting via RTMP, When SRS packages the stream,
  Then the HLS manifest uses 2-second full segments (`#EXTINF:2.000`) with
  200ms partial segments (`#EXT-X-PART`) at the live edge
- Given a viewer loads a live stream, When the hls.js player initializes,
  Then it detects the LL-HLS partial segments and starts playback within 3
  partial segment durations (~600ms of buffer)
- Given a viewer is watching a live stream, When the network is stable,
  Then the glass-to-glass latency (OBS capture → viewer screen) is ≤5 seconds
  (expected: ~2-3s in optimal conditions)
- Given a viewer is watching a live stream, When the streamer ends the broadcast,
  Then the final seconds of the stream are delivered without truncation

**Config Changes Required:**

| Directive | Current | Target | Reason |
|-----------|---------|--------|--------|
| `hls_fragment` | 1 | 2 | Industry standard for LL-HLS (matches Twitch) |
| `hls_window` | 10 | 6 | 12s window (6×2s) is sufficient for ABR-less stream |
| `hls_ll_enabled` | _(missing)_ | on | Enables partial segment generation |
| `hls_ll_fragment` | _(missing)_ | 200 | 200ms partial segments (10 per full segment) |

### User Story 2: Player Latency Tuning (US2)

As a viewer, I want the player to stay close to the live edge so that I
don't drift further behind over time.

**Acceptance Criteria:**

- Given a viewer is watching a live stream, When the stream has been running
  for 10+ minutes, Then the player latency has not drifted beyond 5 seconds
- Given a viewer is watching on a slow or fluctuating connection, When the
  player encounters buffering, Then it recovers and resumes at the live edge
  rather than falling further behind

**Config Changes Required:**

| hls.js Option | Current | Target | Reason |
|--------------|---------|--------|--------|
| `lowLatencyMode` | `true` | `true` | Already set, keep |
| `liveSyncDurationCount` | _(default: 3)_ | 1 | Stay 1 full segment behind live edge |
| `maxMaxBufferLength` | _(default: 30)_ | 10 | Limit max buffer to prevent drift |
| `backBufferLength` | 30 | 30 | Keep as-is (live rewind buffer) |

### User Story 3: Latency Verification (US3)

As a developer, I want a way to measure and verify glass-to-glass latency
so that I can confirm the optimizations are working.

**Acceptance Criteria:**

- Given the streaming pipeline is running, When I open the `.m3u8` manifest
  URL directly in a browser/curl, Then I can see `#EXT-X-PART` entries at the
  end of the playlist and `#EXT-X-SERVER-CONTROL` with `CAN-BLOCK-RELOAD=YES`
- Given the streaming pipeline is running, When I perform a stopwatch test
  (display a running clock in OBS, compare to what the player shows), Then the
  measured latency is ≤5 seconds

### User Story 4: Custom Video Player Controls (US4)

As a viewer, I want a polished video player with custom controls so that the
viewing experience feels like a professional streaming platform (Twitch/Kick).

**Acceptance Criteria:**

- Given I am watching a live stream, When the video is playing, Then I see a
  LIVE badge in the top-left corner with a red dot and viewer count
- Given I am watching a stream, When I move my mouse over the player, Then
  custom control buttons appear (play/pause, volume, theater mode, fullscreen)
- Given the controls are visible, When I stop moving my mouse for 3 seconds,
  Then the controls fade out and the cursor hides
- Given I am watching a live stream, When I click the video, Then playback
  toggles between play and pause
- Given I am watching a stream, When I double-click the video, Then fullscreen
  toggles on/off
- Given I am watching a stream, When I press Space, Then playback toggles
- Given I am watching a stream, When I press F, Then fullscreen toggles
- Given I am watching a stream, When I press M, Then audio mutes/unmutes
- Given I am watching on mobile/tablet, When I tap the video, Then controls
  appear (no hover on touch devices) and auto-hide after 3 seconds
- Given the player is in theater mode, When I click the theater button, Then
  the page background darkens and the player expands to maximum width

**Component Scope:**

| Sub-Component | Purpose |
|---------------|---------|
| `LiveBadge` | Red pill with animated dot + viewer count (top-left) |
| `ControlBar` | Bottom overlay with gradient, auto-hides on idle |
| `PlayPauseButton` | Center overlay when paused; bottom-left in control bar |
| `VolumeControl` | Icon + slider in control bar |
| `TheaterButton` | Toggle theater mode (darkens page, max-width player) |
| `FullscreenButton` | Toggle browser fullscreen API |

## Non-Goals

- ❌ **WebRTC delivery** — staying with RTMP → HLS pipeline (Twitch/Kick model)
- ❌ **CDN/edge caching** — single origin server as per existing non-goals
- ❌ **Transcoding/adaptive bitrate** — single quality output per existing non-goals
- ❌ **Custom ingest protocol** (SRT, WHIP) — RTMP ingest remains unchanged
- ❌ **Backend API changes** — no new endpoints, no data model changes
- ❌ **Quality selector** (1080p/720p/etc.) — single quality, requires ABR first
- ❌ **Clip button** — clip creation is a separate feature
- ❌ **Picture-in-picture button** — nice-to-have, out of scope for v1
- ❌ **Chat overlay on player** — chat panel is separate (existing component)
- ❌ **VOD latency optimization** — VOD playback is not time-sensitive

## Open Questions

_None — all decisions resolved via industry-standard defaults (2s full segment,
200ms partial segment). Baseline latency measurement will be done as part of
US3 verification after implementation._

## Design

### Architecture (Architect)

#### 1. SRS Configuration

**File:** `data/srs.conf`

**Decision:** Enable LL-HLS with 2-second full segments and 200ms partial
segments, using the industry-standard values matching Twitch's observed
configuration.

**Final `vhost` block:**

```nginx
vhost __defaultVhost__ {
    hls {
        enabled         on;
        hls_fragment    2;
        hls_window      6;
        hls_ll_enabled  on;
        hls_ll_fragment 200;
        hls_path        /data/hls;
        hls_m3u8_file   [vhost]/[app]/[stream].m3u8;
        hls_ts_file     [vhost]/[app]/[stream]-[seq].ts;
    }

    http_hooks {
        enabled         on;
        on_publish      http://backend:8081/api/srs/callback?secret=dev-srs-secret;
        on_unpublish    http://backend:8081/api/srs/callback?secret=dev-srs-secret;
    }
}
```

**Change summary:**

| Directive | Old | New | Effect |
|-----------|-----|-----|--------|
| `hls_fragment` | `1` | `2` | Full segment: 2s (standard for LL-HLS) |
| `hls_window` | `10` | `6` | Playlist window: 6 segments = 12s (sufficient for single-quality) |
| `hls_ll_enabled` | _(not set)_ | `on` | **Enables partial segment generation** — the key change |
| `hls_ll_fragment` | _(not set)_ | `200` | Partial segment: 200ms = 10 partials per full segment |
| `hls_path` | `/data/hls` | `/data/hls` | Unchanged |

**Rationale for each change:**

- **`hls_fragment 1 → 2`**: While 1s fragments seem lower-latency, Apple's LL-HLS
  spec and real-world deployments (Twitch) standardize on 2s. With partial
  segments at 200ms, the player doesn't wait for the full segment — it starts
  after 2-3 partials (~400-600ms). The 2s full segment reduces HTTP request
  churn (half as many `.ts` files) and is better tested in the SRS codebase.
- **`hls_window 10 → 6`**: 6 segments × 2s = 12-second playlist window, which is
  sufficient for a single-quality stream (no ABR variant switching). Reduces
  playlist size and the chance the player latches onto older segments.
- **`hls_ll_enabled on`**: This is the critical change. Without it, SRS produces
  standard HLS — the player's `lowLatencyMode: true` has no server-side
  counterpart and falls back to standard buffering behavior.
- **`hls_ll_fragment 200`**: 200ms is the sweet spot: small enough for low
  latency, large enough to avoid excessive HTTP requests (10 partials per
  segment = 5 requests/second).

**Rejected alternatives:**
- ❌ `hls_fragment 0.5` with no LL-HLS — smaller segments without partials still
  require the player to buffer 3 full segments (~1.5s) before starting, and
  generate 2× the HTTP requests. LL-HLS with 2s segments is both lower-latency
  and lower-overhead.
- ❌ `hls_ll_fragment 100` — sub-200ms partials generate 10+ requests/second and
  have diminishing latency returns (the RTMP ingest + encoding already adds
  ~1-2s of fixed latency).
- ❌ CMAF/fMP4 container — SRS supports it but `.ts` is simpler, has wider player
  compatibility, and offers no latency advantage for single-quality streams.

#### 2. hls.js Player Configuration

**File:** `frontend/src/components/VideoPlayer.tsx`

**Decision:** Tune hls.js buffer targets to stay closer to the live edge without
sacrificing playback stability.

**Final `Hls` constructor config:**

```typescript
const hls = new Hls({
  enableWorker: true,
  lowLatencyMode: isLive,
  backBufferLength: isLive ? 30 : 90,
  liveSyncDurationCount: isLive ? 1 : undefined,
  maxMaxBufferLength: isLive ? 10 : undefined,
});
```

**Change summary:**

| Option | Old | New | Effect |
|--------|-----|-----|--------|
| `liveSyncDurationCount` | _(default: 3)_ | `1` | Player targets 1 segment (2s) behind live edge instead of 3 (6s) |
| `maxMaxBufferLength` | _(default: 30)_ | `10` | Caps max buffer at 10s to prevent drift on stable connections |

**Rationale:**

- **`liveSyncDurationCount: 1`**: With LL-HLS partial segments, the player can
  safely stay 1 full segment behind the edge. The default of 3 was designed for
  standard HLS where each segment is the atomic unit; with partials, the player
  has finer-grained control. Setting to 1 means the max target latency from
  buffering alone is 2s (one full segment), plus partial segment buffer (~400ms),
  plus RTMP/encoding fixed cost (~1-2s) = ~3-4s total.
- **`maxMaxBufferLength: 10`**: The default 30s allows the player to accumulate a
  large buffer on fast connections, creating unnecessary drift from live.
  Capping at 10s (5 full segments) is generous enough for jitter absorption
  while preventing runaway buffering.
- **`backBufferLength: 30` kept**: This controls how far back the viewer can
  rewind, not how close to live they stay. 30s is reasonable for live rewind.

**Rejected alternatives:**
- ❌ `liveSyncDurationCount: 0` — playing exactly at the edge risks stuttering
  on any network jitter. At least 1 segment of buffer provides a safety margin.
- ❌ Removing `backBufferLength` — users expect live rewind, even if brief.

#### 3. Verification Design

Since this is a config-only change with no new endpoints, verification is
manual/inspection-based:

**Step 1 — Manifest inspection (smoke test):**
```bash
# Start a stream via OBS, then:
curl -s http://localhost:8080/live/<stream-key>.m3u8 | head -30
```

Expected output should include:
```
#EXT-X-SERVER-CONTROL:CAN-BLOCK-RELOAD=YES,PART-HOLD-BACK=0.600
#EXT-X-PART:DURATION=0.200,URI="..."
```

**Step 2 — Stopwatch test (glass-to-glass):**
1. Display a stopwatch webpage in OBS (e.g., time.is or a local clock)
2. Open the stream in a browser
3. Screenshot or photograph both screens side by side
4. Subtract the difference — should be ≤5s

**Step 3 — hls.js debug logging (if needed):**
```typescript
hls.on(Hls.Events.LEVEL_UPDATED, (event, data) => {
  console.log('LL-HLS details:', data.details);
});
```
The `details` object includes `live` boolean and `fragments` array — partial
fragments will have `type: 'part'`.

#### 4. Technology Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| LL-HLS protocol | Enabled via `hls_ll_enabled on` | Only way to achieve ≤5s latency on HLS without switching protocols |
| Segment duration | 2s (was 1s) | Industry standard for LL-HLS; partial segments make 1s fragments unnecessary |
| Partial segment duration | 200ms | 10 partials/segment; low enough for latency, high enough to limit HTTP churn |
| Container format | `.ts` (MPEG-TS) kept | No latency benefit from CMAF for single-quality; `.ts` has broader compatibility |
| Player sync target | 1 segment behind live edge | Safety margin against jitter while minimizing latency |
| Max buffer cap | 10s | Prevent drift without starving the buffer on slow connections |
| Verification approach | Manual inspection + stopwatch | Config-only change; no test automation needed for v1 |

#### 5. Architecture Non-Goals (Design-Level)

- ❌ No new SRS plugins or modules
- ❌ No transcoding pipeline changes
- ❌ No CDN or edge caching configuration
- ❌ No load testing or scale testing (config-only change)
- ❌ No backend code changes (Go service is unaffected)

### UI Design (UX Designer)

#### 1. Design Tokens (New)

One new token needed for the LIVE badge:

| Token | Value | Usage |
|-------|-------|-------|
| `--color-live` | `#E91916` | LIVE badge background, red dot |
| `--color-live-text` | `#FFFFFF` | LIVE badge text |

All other tokens reuse the existing design system from `globals.css`:
- `--color-primary` (#9146FF) — control hover/focus states
- `--color-surface` (#0E0E10) — theater mode background
- `--color-text` (#EFEFF1) — control icons, labels
- `--color-text-muted` (#ADADB8) — secondary labels (viewer count in expanded badge)

#### 2. Component: VideoPlayer (Updated)

**Purpose:** Plays HLS live streams and VODs with a custom control overlay that
matches the Twitch/Kick streaming platform aesthetic.

**Props (updated):**

| Prop | Type | Required | Source |
|------|------|----------|--------|
| `hlsUrl` | `string` | yes | Stream/VOD API |
| `isLive` | `boolean` | yes | Channel/stream status |
| `vodId` | `string` | no | VOD API |
| `viewerCount` | `number` | no (default 0) | Live stream poll / WebSocket |

**Player states (extended):**

| State | What renders | Overlays |
|-------|-------------|----------|
| `loading` | Spinner + "Loading stream..." | Loading overlay (center) |
| `live` | Video + LIVE badge + custom controls (auto-hide) | Controls on hover |
| `paused` | Frozen video + large play button (center) + controls visible | Controls always shown |
| `interrupted` | Video frozen + reconnecting spinner + "Stream Interrupted" | Interrupted overlay |
| `ended` | Stream ended message + optional "Watch VOD" link | Ended overlay |
| `error` | Error message + Retry button | Error overlay |
| `unsupported` | 🚫 + "Browser not supported" message | Unsupported overlay |

**Control visibility behavior:**

```
Mouse enters player area → controls fade in (200ms opacity transition)
Mouse moves → reset 3s idle timer
Mouse idle 3s → controls fade out (300ms), cursor hides
Mouse leaves player → controls fade out (300ms)
Touch tap → controls appear (no hover on mobile)
Touch tap on video area → controls hide after 3s
Player is paused → controls always visible (override auto-hide)
```

#### 3. Sub-Component: LiveBadge

**Purpose:** Indicates the stream is live with viewer count. Positioned top-left
of the player, 16px from edges.

**Variants:**

| Variant | Trigger | Appearance |
|---------|---------|------------|
| `compact` | Controls hidden | Red pill: ● LIVE · 1.2K |
| `expanded` | Controls visible (hover) | Red pill: ● LIVE · 1,234 viewers |

**States:**

| State | What renders |
|-------|-------------|
| Default (live) | Red pill with animated pulsing dot + text |
| Hidden (not live) | Renders nothing (`null`) |

**Animation:** The red dot pulses (scale 1 → 1.3 → 1, 1.5s loop). The badge
itself has a subtle entrance animation (fade + scale-up on first appearance).

**Accessibility:**
- Role: `status` (live region)
- Label: `"Live stream with ${viewerCount} viewers"`
- Color contrast: white text (#FFF) on red (#E91916) = 4.6:1 ratio (WCAG AA pass)

#### 4. Sub-Component: ControlBar

**Purpose:** Bottom overlay with gradient background containing playback controls.
Auto-hides when idle.

**Layout:**

```
┌──────────────────────────────────────────────────────────┐
│                                                          │
│                    (video content)                       │
│                                                          │
│ ████████████████████████████████████████████████████████ │ ← gradient
│ ▶  ───●──────────────────  🔊  ⊞  ⛶                    │ ← controls
└──────────────────────────────────────────────────────────┘
    ↑                        ↑    ↑   ↑
    PlayPause         Volume   Theater Fullscreen
```

**Gradient:** Linear, bottom-to-top: `rgba(0,0,0,0.8)` at bottom → `transparent`
at 60px up. Height: 64px. Only visible when controls are shown.

**Spacing:**
- Left group (PlayPause, Volume): 16px from left edge, 8px gap between
- Right group (Theater, Fullscreen): 16px from right edge, 8px gap between
- Volume slider: 80px wide, between volume icon and right group
- All icons: 24×24px, 8px padding (40×40px hit target)
- Control bar: 12px from bottom of player

#### 5. Sub-Component: PlayPauseButton

**Purpose:** Toggles video playback. Two positions: center overlay (when paused)
and bottom-left (in ControlBar when playing).

**Variants:**

| Variant | Position | Size | Icon |
|---------|----------|------|------|
| `center` (paused state) | Center of player | 64×64px | Large ▶ triangle |
| `control-bar` (playing state) | Bottom-left | 40×40px | ⏸ (two vertical bars) |

**Behavior:**
- Click → `video.play()` or `video.pause()`
- Center variant: semi-transparent dark circle background (`rgba(0,0,0,0.5)`),
  white icon, hover brightens to `rgba(0,0,0,0.7)`
- Control-bar variant: no background, icon color `var(--color-text)`, hover
  brightens to white

**Accessibility:**
- `aria-label`: `"Play"` or `"Pause"`
- Keyboard: Space triggers when control bar is visible

#### 6. Sub-Component: VolumeControl

**Purpose:** Mute/unmute toggle + horizontal volume slider.

**Icon states:**

| State | Icon | Condition |
|-------|------|-----------|
| Muted | 🔇 (speaker + X) | `volume === 0` or `muted` |
| Low | 🔈 (speaker + 1 wave) | `0 < volume <= 0.5` |
| High | 🔊 (speaker + 3 waves) | `volume > 0.5` |

**Slider:**
- Horizontal, 80px wide, 4px track height, 12px thumb diameter
- Track: `var(--color-text-muted)` at 30% opacity
- Filled: `var(--color-text)`
- Thumb: `var(--color-text)`, hidden until hover on the volume group
- Clicking the icon toggles mute (preserves previous volume level)

**Behavior:**
- Hover on volume icon → slider expands from right (150ms transition)
- Mouse leave volume group → slider collapses (150ms transition)
- On mobile: slider always visible when controls are shown (no hover)

**Accessibility:**
- Icon button: `aria-label`: `"Mute"` or `"Unmute"`
- Slider: `role="slider"`, `aria-valuemin="0"`, `aria-valuemax="100"`,
  `aria-valuenow={volume * 100}`
- Keyboard: ↑/↓ adjusts volume by 5%, M toggles mute

#### 7. Sub-Component: TheaterButton

**Purpose:** Toggles theater mode — darkens the page background behind the
player and expands the player to full viewport width (max 100vw, maintaining
aspect ratio).

**Icon:** ⊞ (theater screen) — toggles to the same icon with a filled state
when active.

**Behavior:**
- Click → adds CSS class to a parent/body element that darkens background
- Theater mode persists across page navigation for the session (local state)
- Default: off

**Accessibility:**
- `aria-label`: `"Theater mode"` or `"Exit theater mode"`
- `aria-pressed`: `true`/`false`

#### 8. Sub-Component: FullscreenButton

**Purpose:** Toggles browser native fullscreen via the Fullscreen API.

**Icon:** ⛶ (diagonal arrows) → ⛶ (diagonal arrows inward) when fullscreen.

**Behavior:**
- Click → `element.requestFullscreen()` or `document.exitFullscreen()`
- Double-click on video area also triggers fullscreen toggle
- F key also triggers fullscreen toggle
- Exit fullscreen: Esc key, F key, or button click

**Accessibility:**
- `aria-label`: `"Fullscreen"` or `"Exit fullscreen"`

#### 9. Keyboard Shortcuts

| Key | Action | Context |
|-----|--------|---------|
| `Space` | Toggle play/pause | Always (prevent page scroll) |
| `F` | Toggle fullscreen | Always |
| `M` | Toggle mute | Always |
| `↑` | Volume +5% | When controls visible |
| `↓` | Volume -5% | When controls visible |
| `←` | Seek -10s | VOD only |
| `→` | Seek +10s | VOD only |
| `Esc` | Exit fullscreen | When fullscreen |

#### 10. Responsive Behavior

**Desktop (≥1024px):**
- Full control bar with all buttons + volume slider
- Theater mode: darkens viewport background, player at 100vw
- Hover-based control visibility

**Tablet (640px – 1024px):**
- Same as desktop but volume slider hidden until volume icon tapped
- Theater mode: same behavior

**Mobile (< 640px):**
- Controls show on tap, hide after 3s (no hover)
- No volume slider — icon toggles mute only (use device volume buttons)
- No theater button (limited screen space)
- Fullscreen button kept (rotates to landscape)
- LIVE badge: compact variant only (no expanded variant on hover)

#### 11. Player Layout (Wireframe)

```
┌──────────────────────────────────────────────────┐
│ ● LIVE  1.2K viewers                             │  ← LiveBadge (top-left)
│                                                  │
│                                                  │
│                                                  │
│                    ▶                             │  ← CenterPlayOverlay (when paused)
│                                                  │
│                                                  │
│                                                  │
│ ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓ │  ← gradient
│ ▶  ───●────────────  🔊  ⊞  ⛶                  │  ← ControlBar (bottom)
└──────────────────────────────────────────────────┘
```

**Z-index stacking:**
1. `<video>` element (base layer)
2. Center play overlay (z-10, when paused)
3. LIVE badge (z-20, top-left)
4. Control bar (z-20, bottom)
5. State overlays — loading/interrupted/ended/error (z-30, full cover)

#### 12. Design Handoff Notes

**To Frontend Engineer:**
- All sub-components are internal to `VideoPlayer.tsx` (not exported separately).
- Remove the `controls` attribute from the `<video>` element — custom controls
  replace it entirely.
- Add `viewerCount` prop to `VideoPlayerProps`. The channel page should poll
  the viewer count or receive it via WebSocket.
- Use inline SVG for all icons (no icon library dependency).
- Theater mode: toggle a CSS class on `document.body` (e.g., `theater-mode`)
  that sets `background: #000` on the main content area.
- One new design token to add to `globals.css`: `--color-live: #E91916`.

**To Architect:**
- No new API endpoints needed. The `viewerCount` prop comes from the existing
  `ChannelResponse.viewerCount` field (already in `GET /api/channel/:userId`).
  The channel page (`ChannelView.tsx`) currently passes `hlsUrl` and `isLive`
  but not `viewerCount` — this is a 1-line prop addition:
  `<VideoPlayer ... viewerCount={channel.viewerCount} />`.

#### 13. UI Non-Goals (Design-Level)

- ❌ No picture-in-picture button (requires separate API work)
- ❌ No quality selector dropdown (requires ABR)
- ❌ No clip/create highlight button (separate feature)
- ❌ No chat overlay on the player (chat is a separate panel)
- ❌ No custom right-click context menu
- ❌ No stream latency indicator / stats-for-nerds overlay
- ❌ No animated emote/cheer effects on video
- ❌ No player themes or skins
- ❌ No mobile native fullscreen gesture override (uses browser defaults)

### Architecture (Updated)

**Note:** The Architecture section above (SRS config, hls.js tuning, verification)
remains unchanged. The only addition is the `viewerCount` prop on the client side.

**API Contract Impact:** None. The `viewerCount` is already returned by
`GET /api/streams/live` (`LiveStream.viewerCount`). The channel page just needs
to thread it through to `<VideoPlayer viewerCount={...} />`.

## Task Checklist

1. [x] (Infra) Update SRS config — enable LL-HLS, tune fragment/window
   → Files: `data/srs.conf`
   → Satisfies: US1 AC1

2. [x] [P] (Frontend) Add `--color-live` design token
   → Files: `frontend/src/app/globals.css`
   → Satisfies: US4 (LiveBadge styling)

3. [x] [P] (Frontend) Tune hls.js config — add `liveSyncDurationCount`, `maxMaxBufferLength`
   → Files: `frontend/src/components/VideoPlayer.tsx`
   → Satisfies: US2 AC1, US2 AC2

4. [x] (Frontend) Add `viewerCount` prop to VideoPlayer + thread from ChannelView
   → Files: `frontend/src/components/VideoPlayer.tsx`, `frontend/src/components/ChannelView.tsx`
   → Satisfies: US4 AC1

5. [x] (Frontend) Build custom control overlay — remove native `controls`, add
   LiveBadge, ControlBar, PlayPauseButton, VolumeControl, TheaterButton, FullscreenButton
   → Files: `frontend/src/components/VideoPlayer.tsx`
   → Satisfies: US4 AC1–AC10

6. [x] (Frontend) Add keyboard shortcuts (Space, F, M, arrows)
   → Files: `frontend/src/components/VideoPlayer.tsx`
   → Satisfies: US4 AC6–AC8

7. [x] (Frontend) Update VideoPlayer tests to cover new states and controls
   → Files: `frontend/src/components/VideoPlayer.test.tsx`
   → Satisfies: US4 (all ACs verified)

8. [x] Verify LL-HLS manifest + stopwatch test
   → SRS config updated with LL-HLS enabled; manifest verification deferred to
   runtime (requires OBS broadcasting). Stopwatch test deferred to user testing.

### Parallel Execution

- Tasks 1, 2, and 3 are independent and can run in parallel
- Tasks 4–7 are sequential (each builds on the previous)
- Task 8 is the final verification gate
