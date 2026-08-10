# Retro — Chat Implementation, 2026-08-10

**Session Log:** [2026-08-10-session-log.md](./2026-08-10-session-log.md)
**Spec:** [us4-real-time-chat.md](../live-streaming-platform/us4-real-time-chat.md)
**PR:** #22
**Branch:** `feat/chat-implementation`

---

## Correction → Rule Mapping

### 1. WebSocket route was in auth group, preventing anonymous reads

**What happened:** Originally the WS route was inside `AuthMiddleware`, requiring JWT to even open the connection. Spec says auth is optional for reads.

**Root cause:** The spec's auth requirement wasn't reflected in the route wiring.

**Fix:** Moved WS to root router at `/ws/chat/{streamID}`, handling auth internally with `extractToken()` and optional fallback to anonymous.

**Rule to add to `go-chi` skill:**
> When a WebSocket endpoint has optional auth, register it outside the auth middleware group and extract credentials manually in the handler. Use `extractToken()` to check cookies/headers without rejecting the connection.

---

### 2. WebSocket writes happening from two goroutines

**What happened:** `readPump` was calling `wsjson.Write` for errors and pongs while `writePump` was also writing. `nhooyr.io/websocket` doesn't support concurrent writes.

**Root cause:** The `nhooyr.io/websocket` constraint wasn't documented in project standards.

**Fix:** Routed all writes through `client.Send` channel via `sendToClient()`. Single `writePump` goroutine handles all `conn.Write` calls.

**Rule to add to `go-chi` skill:**
> When using `nhooyr.io/websocket`, route ALL writes through a single goroutine (writePump). Never call `wsjson.Write` or `conn.Write` from `readPump`. Use a channel (`client.Send`) with non-blocking `select/default` to avoid head-of-line blocking.

---

### 3. Stream validation before WebSocket upgrade

**What happened:** The handler returned HTTP 400 before the WebSocket upgrade when the stream wasn't live. The browser couldn't receive the close code and fell into infinite reconnect.

**Root cause:** Misunderstanding of the WebSocket lifecycle — the spec says "close with code 4001", not "reject with HTTP 400".

**Fix:** Reordered to accept the WebSocket FIRST, then validate stream liveness. If offline, close with `websocket.StatusCode(4001)`.

**Rule to add to `go-chi` skill:**
> Accept the WebSocket connection before performing application-level validation. This lets the browser receive meaningful close codes (like 4001 "stream offline") instead of opaque HTTP errors that trigger infinite reconnect loops.

---

### 4. Next.js cannot proxy WebSocket connections

**What happened:** `getWsUrl` used `API_HOST` (Next.js server at port 3000). Next.js's built-in proxy rewrites HTTP but doesn't support WebSocket upgrades, causing connection failures.

**Root cause:** Assumption that Next.js's `rewrites` config handles WebSocket — it doesn't.

**Fix:** Added separate `NEXT_PUBLIC_WS_URL` env var defaulting to the backend URL directly (`http://localhost:8081`). Documented in `specs/memories/2026-08-10-nextjs-websocket-proxy.md`.

**Rule to add to `nextjs` skill:**
> WebSocket connections must bypass the Next.js proxy. Use a separate `NEXT_PUBLIC_WS_URL` environment variable pointing directly to the backend. In local dev this is the backend port; in production behind a reverse proxy that supports WS upgrades, it can match the API URL.

---

### 5. Chat not rendering — missing `streamId` in channel API

**What happened:** The frontend's `ChannelView` checks `channel.streamId` to decide whether to render `ChatPanel`. The backend's `GET /api/channel/:userId` never returned `streamId`, so chat never appeared even for live streams.

**Root cause:** The `StreamID` field was defined in the frontend TypeScript type (`ChannelResponse`) but was never added to the backend's `ChannelInfo` domain struct or repo query. The API contract wasn't verified bidirectionally.

**Fix:** Added `StreamID` to `ChannelInfo`, updated the SQL query to include `s.id`, populated the field when the stream is live.

**Rule to add to AGENTS.md Section 2.1:**
> When a frontend type includes a field (e.g., `streamId: string | null`), verify the backend actually populates it. The frontend type is the contract — every nullable field must have a corresponding backend implementation that sets it to non-null at the right time.

---

### 6. React StrictMode double-invoke causing duplicate messages

**What happened:** React's StrictMode double-invokes `useEffect`: the first WebSocket receives `sendInitialBatch`, cleanup closes it, the second WebSocket receives the same batch again — all appended to the same undrained `messages` array.

**Root cause:** No deduplication on message ID in the `onmessage` handler.

**Fix:** Added `if (prev.some(m => m.id === msg.id)) return prev` check before appending.

**Rule to add to `nextjs` skill:**
> WebSocket `onmessage` handlers that append to React state must deduplicate by message ID. React StrictMode double-invokes effects in development, causing `sendInitialBatch` (or equivalent server push) to fire twice with the same payload.

---

### 7. Double-send from `<button>` without explicit `type`

**What happened:** Pressing Enter in the chat input triggered both `handleKeyDown` (via `onKeyDown`) AND the browser's implicit form submission on the `<button>` (which defaults to `type="submit"`).

**Root cause:** `<button>` without `type` defaults to `type="submit"` in HTML. Pressing Enter in an input within the same container triggers submit behavior.

**Fix:** Added `type="button"` to the send button.

**Rule to add to `nextjs` skill:**
> Always set `type="button"` on `<button>` elements inside forms or input containers. Without it, the browser treats them as submit buttons, causing Enter key presses to fire both the explicit handler and implicit submission.

---

### 8. Pre-existing `CreateUser` parameter order (discovered during CI)

**What happened:** PR #18 added a `rawKey` parameter to `CreateUser` between `avatarURL` and `keyHash`, but didn't update the 10 test seed calls. All auth integration tests were silently broken — passing the stream key as google_id. CI didn't catch it because integration tests weren't running on that PR.

**Root cause:** Function signature change without updating all call sites. Go's compiler catches type mismatches but not positional argument swaps when all types are `string`.

**Fix:** Reordered all 10 calls to match the actual signature: `(ctx, googleID, email, name, avatarURL, rawKey, keyHash)`.

**Rule to reinforce in AGENTS.md:**
> After adding/inserting a parameter to a function, grep all call sites. When all parameters are the same type (e.g., all `string`), the compiler won't catch positional errors — only runtime tests will.

---

## Skill/AGENTS Updates Needed

| File | Rule |
|------|------|
| `.agents/skills/go-chi/SKILL.md` | WebSocket write-pump pattern for `nhooyr.io/websocket` |
| `.agents/skills/go-chi/SKILL.md` | Optional auth: register WS outside middleware, extract manually |
| `.agents/skills/go-chi/SKILL.md` | Accept WS before app-level validation for meaningful close codes |
| `.agents/skills/nextjs/SKILL.md` | WebSocket must bypass Next.js proxy — use separate `WS_URL` env var |
| `.agents/skills/nextjs/SKILL.md` | Deduplicate WebSocket messages by ID in React `onmessage` handlers |
| `.agents/skills/nextjs/SKILL.md` | Always set `type="button"` on buttons near inputs |
| `AGENTS.md` Section 2.1 | Verify bidirectional API contract fidelity — frontend types must match backend responses |
| `AGENTS.md` Section 5.1 | After inserting a parameter, grep all call sites — compiler won't catch string-positional errors |

---

## What Went Well

1. **Spec-first approach caught 12 gaps** — The detailed spec made it trivial to audit the implementation. Without it, we would have shipped broken chat (no `streamId`, wrong WS protocol, no rate limiting).

2. **Frontend already followed the spec** — `ChatPanel` and `ChatInput` were built to the spec from the start. Only a few fixes were needed (double-send, dedup, WS_URL).

3. **`writePump` single-writer refactor** — Moving all writes through `client.Send` solved the concurrency issue cleanly and is now the canonical pattern.

4. **Docker backend survives terminal** — The backend runs in Docker, independent of the agent's terminal lifecycle. Only the frontend needed `nohup`.

5. **`gh` CLI for CI polling** — Quick feedback loop for integration test failures.
