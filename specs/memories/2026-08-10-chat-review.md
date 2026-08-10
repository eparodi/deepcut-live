# Final Review: PR #22 — Real-Time Chat

**Branch:** `feat/chat-implementation`
**Spec:** `specs/live-streaming-platform/us4-real-time-chat.md`
**Diff:** 23 files, +1121 / -346

---

## 🔴 Critical (must fix before merge)

*None.*

---

## 🟡 Warning (should fix)

1. **`chatAuthAdapter` duplicated in `main.go` and `server_test.go`** — Both define identical `ValidateToken` methods. If the auth service API changes, only one copy will be updated. Deferred because Go package visibility makes extraction awkward (the adapter needs `auth/application` types, and `main` can't be imported).
   → `backend/cmd/server/main.go:199-226`, `backend/cmd/server/server_test.go:583-610`

2. **Origin patterns hardcoded** — `websocket.AcceptOptions.OriginPatterns` is `["localhost:3000", "localhost:8081", "127.0.0.1:3000", "127.0.0.1:8081"]`. In production, this needs to match the deployed domain. Should read from an env var or derive from the CORS configuration.
   → `backend/internal/modules/chat/adapter/http/handler.go:92-93`

---

## 🔵 Style (nice to fix)

1. **Verbose whitespace check** — Lines 249-255 manually iterate runes. `strings.TrimSpace(p.Message) == ""` is clearer and handles all Unicode whitespace.
   → `backend/internal/modules/chat/adapter/http/handler.go:249-255`

2. **`SendToClient` (hub) vs `sendToClient` (handler)** — Two methods with the same purpose but different implementations exist: `hub.SendToClient(client, msg)` and `h.sendToClient(client, msgType, payload)`. Consolidate.
   → `backend/internal/modules/chat/application/service.go:176`, `backend/internal/modules/chat/adapter/http/handler.go:283`

3. **`ExpireIdle` scans all clients** — Every 30s, `idleMonitor` calls `ExpireIdle` which iterates all clients in the room and returns expired ones. The caller then linear-scans the returned slice to find itself. O(n) per-client per-tick. Fine for hundreds of clients; consider a per-client deadline timer for thousands.
   → `backend/internal/modules/chat/adapter/http/handler.go:327-328`

---

## ✅ What's Good

- **Correct WebSocket lifecycle** — Upgrade before validation, close with spec code 4001, optional auth, single-writer goroutine pattern. Faithfully implements the spec's connection lifecycle.

- **Token-bucket rate limiting** — `AllowMessage` properly implements leaky bucket with burst=3 and refill=1/2s. Thread-safe via `sync.RWMutex`. Anonymous users (empty userID) are always rejected.

- **Cursor-based pagination** — `GET /api/chat/{streamID}/messages` uses `?before=` timestamp cursor with `hasMore`. Fetches `limit+1` to determine if more pages exist. Non-null empty array guard in response.

- **Idle timeout with activity tracking** — `Touch` on every client message + `ExpireIdle` polling every 30s. Closes connections idle for 2 minutes. Prevents resource leaks from abandoned tabs.

- **Frontend dedup** — `onmessage` checks `prev.some(m => m.id === msg.id)` before appending, handling React StrictMode double-invoke without clearing the message list on reconnect.

- **Double-send fix** — `<button type="button">` prevents browser's implicit form submission from firing alongside the `onKeyDown` Enter handler.

- **`streamId` in channel response** — `ChannelInfo.StreamID` added to domain struct, repo query, and returned in JSON. Enables `ChatPanel` to know which stream to connect to.

- **Separate `WS_HOST` for WebSocket** — bypasses Next.js proxy (which can't forward WS upgrades). Documented in `specs/memories/2026-08-10-nextjs-websocket-proxy.md`.

- **Pre-existing `CreateUser` param order fix** — All 10 test seed calls reordered to match the function signature that was changed in a prior PR.

- **Test coverage** — 20 backend + 25 frontend test packages pass. Integration tests for WebSocket (connect, send, receive, broadcast), cursor pagination, rate limiting, and empty/null guards.

---

## Verdict

**[REVIEW_PASS]** — No critical issues. Three warnings are deferred (adapter duplication, hardcoded origins, O(n) idle scan) and don't block merge. The implementation meets every acceptance criterion in the US4 spec.
