# Frontend Component Catalog (v1)

- **Status:** Draft
- **Owner:** UX Designer (Frontend Engineer implements against it)
- **Created:** 2026-08-13
- **Purpose:** the catalog-first design contract from
  `.agents/skills/ux-designer/references/a2ui.md`. Every shared
  component is documented as a catalog entry (JSON-serializable props,
  data-bound states, tokens only, action semantics) so that (a)
  engineers never make design decisions and (b) any future
  agent-generated surface (A2UI) can compose from these entries
  without new design work.
- **Maintenance rule:** any component change that adds a prop, variant,
  state, or token MUST update its entry here in the same commit.

## Design tokens (source of truth: `frontend/src/app/globals.css`)

Catalog entries may only reference these roles, never raw values.

| Role | Token | Notes |
|------|-------|-------|
| primary fill | `--color-primary` | buttons, badges, fills with white text |
| primary text | `--color-primary-text` | text links/pills on dark surfaces (AA) |
| danger fill | `--color-danger` | destructive buttons, LIVE badge |
| danger text | `--color-danger-text` | error text on dark surfaces (AA) |
| live | `--color-live` | live-state red fills |
| surface / raised | `--color-surface`, `--color-surface-raised` | page vs card backgrounds |
| text / muted | `--color-text`, `--color-text-muted` | body vs secondary text |
| spacing | `--space-1` … `--space-16` (4px grid) | `--space-4` = 16px |
| type ramp | `--text-xs` … `--text-2xl` | sizes with line-heights |

Breakpoints: `sm` 640 / `md` 768 / `lg` 1024 / `xl` 1280 (Tailwind).

## Catalog entries

### Catalog: navbar

**Purpose:** global header — branding, primary navigation (Browse,
Search), auth state. **Behavior:** sticky top; auth state derived from
`initialSignedIn` (server-read cookie) + `GET /api/me`.

**Props:**
| Prop | Type | Required | Default | Source |
|------|------|----------|---------|--------|
| initialSignedIn | boolean | yes | — | token cookie (server-side) |

**States (data-bound):**
| State | Trigger | Renders |
|-------|---------|---------|
| Loading | `initialSignedIn=true`, `me` pending | 8×8 avatar skeleton |
| Signed in | `me != null` | Dashboard link + avatar |
| Auth unknown | `initialSignedIn=true`, fetch failed | Dashboard link, no avatar, NO sign-in button |
| Signed out | `initialSignedIn=false` | "Sign in with Google" button |

**Action semantics:** nav links = navigation only. Sign-in = OAuth
redirect (`/api/auth/google`). No destructive actions.

**Responsive:** wraps (`flex-wrap`); link targets ≥24px (padding
compensated by negative margins).

**Accessibility:** links have text labels; avatar is inside the
Dashboard link (`alt=name`); hover/focus colors pass AA.

**Tokens:** primary (brand), text, text-muted, google-blue (fill).

### Catalog: live-grid

**Purpose:** the live-streams section — heading, count, sort control,
grid/list toggle, cards. **Behavior:** sorts client-side; sort mode is
URL-driven (`?sort=`); view mode is local state.

**Props:**
| Prop | Type | Required | Default | Source |
|------|------|----------|---------|--------|
| streams | LiveStream[] | yes | — | `GET /api/streams` |
| total | number | yes | — | `GET /api/streams` |

**States:**
| State | Trigger | Renders |
|-------|---------|---------|
| Empty | `streams: []` | "No one is live right now" + Browse-past-streams CTA |
| Populated | `streams.length > 0` | LIVE NOW + count + controls + cards |

**Action semantics:** sort buttons rewrite `?sort=` (URL state, not
agent action); grid/list toggle is presentation-only local state.

**Responsive:** grid `2 / md:3 / lg:4` columns; list = full-width
compact rows; controls wrap.

**Accessibility:** sort group `role=radiogroup` + `aria-checked`; view
toggle `aria-pressed`; list `role=list`, cards `role=listitem`.

**Tokens:** danger (live dot), primary (active sort/toggle fills).

### Catalog: live-stream-card

**Purpose:** one live stream as a card (grid) or compact row (list).
**Behavior:** whole card links to `/channel/[id]`.

**Props:**
| Prop | Type | Required | Default | Source |
|------|------|----------|---------|--------|
| stream | LiveStream | yes | — | `GET /api/streams` item |
| isNew | boolean | no | false | derived (fade-in) |
| variant | "grid" \| "list" | no | "grid" | parent's view mode |

**States:** populated only; missing thumbnail and missing avatar are
inline fallbacks (data-URI), never separate screens.

**Action semantics:** click/tap navigates. No mutation.

**Responsive:** grid card = stacked; list row = 128px thumbnail + text.

**Accessibility:** `role=listitem`, aria-label composes name + title +
viewer count; `img` alt + onError fallback (AGENTS.md 10.3); focus
ring visible; hover scale is motion-safe.

**Tokens:** surface-raised, danger (LIVE), text, text-muted,
primary-text (category pill), spacing scale.

### Catalog: vod-card

**Purpose:** one past stream as a card. **Behavior:** links to
`/vods/[id]`; shows duration and processing/failed status.

**Props:**
| Prop | Type | Required | Default | Source |
|------|------|----------|---------|--------|
| vod | VodItem | yes | — | `GET /api/vods` / search results |

**States:** populated; `recordingStatus: pending|processing` →
"Processing" label, `failed` → "Unavailable"; thumbnail fallback on
error.

**Action semantics:** navigation only.

**Accessibility:** alt + onError fallback; status conveyed by text
label, not color alone.

**Tokens:** surface-raised, text, text-muted.

### Catalog: video-player

**Purpose:** HLS playback (live + VOD) with custom controls — the hero
component. **Behavior:** auto-play, muted for live; controls auto-hide;
keyboard shortcuts (space/f/m/arrows).

**Props:**
| Prop | Type | Required | Default | Source |
|------|------|----------|---------|--------|
| hlsUrl | string | yes | — | `ChannelResponse.hlsUrl` / VOD page |
| isLive | boolean | yes | — | channel state |
| vodId | string | no | — | for "watch VOD" CTA |
| viewerCount | number | no | 0 | channel poll |
| onTheaterChange | (boolean) => void | no | — | layout callback |

**States:**
| State | Trigger | Renders |
|-------|---------|---------|
| Loading | init | spinner overlay |
| Live/paused | media events | controls + LIVE badge |
| Interrupted | fatal network/media error | reconnecting copy + retry |
| Error | unrecoverable | message + Retry |
| Unsupported | no HLS support | browser guidance |

**Action semantics:** play/pause/mute/volume/theater/fullscreen —
local media controls, no backend calls. Hidden control bar leaves the
a11y tree (`invisible`).

**Responsive:** volume slider + theater hidden on `sm`; controls 40px
touch targets.

**Accessibility:** every control has aria-label; paused state shows a
center Play button; motion respects reduced-motion.

**Tokens:** primary (spinner/accent), text (controls), live (LIVE
badge).

### Catalog: stream-info

**Purpose:** channel identity block under the player (avatar, title,
category, live/offline, viewer count).

**Props:**
| Prop | Type | Required | Default | Source |
|------|------|----------|---------|--------|
| channel | ChannelResponse | yes | — | `GET /api/channels/:id` |

**States:** live (`isLive=true` → pulsing dot + viewers) / offline
(plain text). Missing avatar → fallback.

**Action semantics:** none (display only); category pill is
non-interactive.

**Accessibility:** live state has text ("X viewers" vs "Offline")
beside the dot — never color alone.

**Tokens:** text, text-muted, danger (dot), primary-text (pill).

### Catalog: chat-panel

**Purpose:** chat column for a stream or VOD replay. **Behavior:**
WebSocket `ws://…/ws/chat/:streamId` (live) or HTTP batch (VOD);
optimistic echo with sending/failed states.

**Props:**
| Prop | Type | Required | Default | Source |
|------|------|----------|---------|--------|
| streamId | string | yes | — | `ChannelResponse.streamId` |
| isSignedIn | boolean | no | false | page-level auth |
| isStreamEnded | boolean | no | false | channel state |
| isVodReplay | boolean | no | false | route |
| initialMessages | ChatMessage[] | no | [] | VOD API |

**States:**
| State | Trigger | Renders |
|-------|---------|---------|
| Connecting | socket not open, no messages | spinner + "Connecting to chat…" |
| Connected+empty | open, `messages: []` | "No messages yet…" |
| Populated | messages present | grouped messages |
| Message sending | pending echo | 70% opacity + "sending…" |
| Message failed | socket drop / not open | "not delivered" + Retry |
| Disconnected | close code 4001 / attempts exhausted | "Chat unavailable" |

**Action semantics:** send → WS `{type:"message"}`; failed-message
Retry re-sends the same text.

**Accessibility:** `role=log` + `aria-live=polite`; usernames + times
labeled; avatar alt.

**Tokens:** surface-raised, text, text-muted, primary-text (username),
danger-text (failed).

### Catalog: chat-input

**Purpose:** message composer. **Behavior:** Enter sends, 500-char
cap, counter ≥400; keeps text when send fails.

**Props:**
| Prop | Type | Required | Default | Source |
|------|------|----------|---------|--------|
| isSignedIn / isReconnecting / isStreamEnded / isRateLimited | boolean | yes | — | chat-panel state |
| rateLimitSeconds | number | no | 0 | rate-limit event |
| signInUrl | string | yes | — | auth |
| onSend | (message) => boolean | yes | — | chat-panel |

**States:** signed-out → sign-in CTA; stream ended → closed message;
reconnecting → notice; rate-limited → countdown + disabled; default →
input + Send.

**Action semantics:** `onSend` returns delivery success; false keeps
the text (retry affordance).

**Accessibility:** input `aria-label`, visible focus ring (WCAG
2.4.7), Send button aria-label, counter reaches danger color at cap.

**Tokens:** surface, text, text-muted, primary, google-blue (sign-in),
danger-text (counter at cap).

### Catalog: stream-key-display

**Purpose:** "Server & Stream Key" section — server URL + stream key
with copy buttons. **Behavior:** clipboard copy + toast feedback.

**Props:**
| Prop | Type | Required | Default | Source |
|------|------|----------|---------|--------|
| streamKey | string | no | — | `GET /api/me` |

**States:** populated → key + copy buttons; missing key → copy
disabled. Toast success/error per copy attempt.

**Action semantics:** copy = read-only clipboard action, no backend
call; errors say "select and copy manually".

**Accessibility:** buttons aria-labeled; toast is `role=status`
(success) / `role=alert` (error) with icons.

**Tokens:** primary (button fills), text-muted (labels).

### Catalog: stream-settings-form

**Purpose:** title + category editor. **Behavior:** local state,
dirty-aware Save, inline validation errors.

**Props:**
| Prop | Type | Required | Default | Source |
|------|------|----------|---------|--------|
| initialTitle / initialCategory | string \| null | yes | — | `ChannelResponse` |
| onSave | (title, category) => void | yes | — | parent handler |
| onError | (message) => void | yes | — | parent handler |

**States:** default / dirty (Save enabled) / saving (inputs+button
disabled, spinner) / error (inline `role=alert` text).

**Action semantics:** Save → `onSave`; failures surface via `onError`,
not silently.

**Accessibility:** labels via `htmlFor`; visible focus ring on inputs.

**Tokens:** surface, surface-raised, text, danger-text (error),
primary (button).

### Catalog: analytics-cards

**Purpose:** dashboard metric cards (This Week). **Behavior:**
display-only.

**Props:**
| Prop | Type | Required | Default | Source |
|------|------|----------|---------|--------|
| analytics | Analytics \| null | yes | — | analytics API |
| loading | boolean | yes | — | parent |
| error | string \| null | yes | — | parent |
| onRetry | () => void | yes | — | parent |

**States:** loading → skeleton cards; error → "Analytics unavailable"
+ Retry; populated → cards.

**Action semantics:** Retry refetches.

**Accessibility:** heading labels the section; Retry target ≥24px.

**Tokens:** skeleton (surface-raised), primary-text (Retry), text.

### Catalog: go-live-preview

**Purpose:** streamer's preview panel. **Behavior:** fetches channel,
polls while live (10s), distinct error state.

**Props:** `userId: string`, `isLive: boolean`.

**States:** loading → skeleton; error → "Couldn't load your stream" +
Retry (never "Not streaming yet"); live → badge + player + channel
link; offline → guidance + channel link.

**Action semantics:** Retry refetches; links navigate.

**Accessibility:** error is `role=alert`; player inherits its labels.

**Tokens:** skeleton, danger (LIVE), primary (CTA), text, text-muted.

### Catalog: force-end-button / regenerate-key-button

**Purpose:** destructive confirmations (end stream / regenerate key).

**Props:** `onEnded` / `onRegenerated` + `onError` callbacks.

**States:** idle → dialog open (focus trapped, Escape/backdrop close,
focus restore) → loading (both buttons disabled, spinner).

**Action semantics:** destructive — consequence copy in-dialog
("Viewers will see 'Stream ended'"), confirm + disabled-while-pending.

**Accessibility:** `role=dialog`, `aria-modal`, `aria-labelledby`,
`useDialogFocus` (initial focus, Tab trap, Escape, restore).

**Tokens:** danger (confirm), primary/secondary (cancel), text.

### Catalog: toast

**Purpose:** transient feedback. **Behavior:** auto-dismiss 3s.

**Props:** `message`, `variant: "success"|"error"`, `durationMs?`,
`onDismiss`.

**States:** success (`role=status`, ✓) / error (`role=alert`, ⚠️) —
variant never color-only.

**Tokens:** primary (success bg), danger (error bg).

### Catalog: spinner

**Purpose:** inline loading indicator inside buttons. No props.

**Accessibility:** `aria-hidden` (decorative; the button's label
carries the state).

**Tokens:** inherits context color.

## Explicit non-goals

- No A2UI runtime/renderer dependency yet (protocol still in preview;
  React renderer not official — see `references/a2ui.md` §7).
- This catalog documents the DESIGN contract; it does not change
  component code.
- Page-level sections (hero, empty states owned by pages) are out of
  scope — they compose these entries.
