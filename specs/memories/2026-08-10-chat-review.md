# Review: feat/chat-implementation — Real-Time Chat

**Branch:** `feat/chat-implementation`
**Spec:** `specs/live-streaming-platform/us4-real-time-chat.md`
**Diff:** 12 files, +844 / -300

---

## 🔴 Critical (must fix before merge)

*None identified.*

---

## 🟡 Warning (should fix)

*All resolved in commit 2.*

1. ~~**String comparison on error**~~ → Fixed: replaced `err.Error() == "rate limited"` with `errors.Is(err, application.ErrRateLimited)` sentinel error.

2. ~~**`context.Background()` in `chatAuthAdapter.ValidateToken`**~~ → Fixed: added `context.Context` parameter to `chatAuth.ValidateToken` interface; adapter forwards `r.Context()` to `GetByID`.

3. ~~**Unused constant `wsPingInterval`**~~ → Fixed: removed.

### 🔵 Style (deferred)

4. **`chatAuthAdapter` duplicated** — The adapter type is defined identically in `main.go` and `server_test.go`. Deferred: extracting to a shared package would require either a new package or exporting from main (not possible). Low risk — both copies are small and the test copy validates the same behavior.
   → `backend/cmd/server/main.go:199-226`, `backend/cmd/server/server_test.go:583-610`

---

## 🔵 Style (nice to fix)

1. **Verbose whitespace check** — `handleChatMessage` (lines 263-270) manually iterates over runes to check for whitespace. Could be simplified to `strings.TrimSpace(p.Message) == ""`. Minor, but clearer.
   → `backend/internal/modules/chat/adapter/http/handler.go:263-270`

2. **`sendToClient` uses `interface{}` for payload** — Could be more type-safe with generics (`[P any]`), but acceptable for a JSON-marshaling helper.
   → `backend/internal/modules/chat/adapter/http/handler.go:242`

---

## ✅ What's Good

- **Single-writer goroutine pattern** — All WebSocket writes route through `writePump` via `client.Send` channel. This is the correct pattern for `nhooyr.io/websocket` which requires serialized writes. The `sendToClient()` helper with non-blocking `select/default` prevents readPump from blocking on a slow client.

- **Rate limiting with token bucket** — `AllowMessage()` in the hub properly implements token bucket (burst=3, refill=1/2s) with thread-safe locking. Anonymous users (empty userID) are always rate-limited.

- **Stream validation before WS accept** — `IsStreamLive()` checks the `streams` table before accepting the WebSocket connection. Prevents connections to non-existent or offline streams.

- **Idle timeout** — `idleMonitor` with `ExpireIdle()` closes connections after 2 minutes of no activity. `LastActive` is touched on every client message (`h.hub.Touch()`). Prevents resource leaks from abandoned tabs.

- **Optional auth** — WS moved out of auth middleware group. `extractToken()` tries cookie first, then Authorization header. Failed auth silently proceeds as anonymous (read-only). Sending messages without auth returns `unauthorized` error code over WS.

- **Cursor-based pagination** — `GET /api/chat/{streamID}/messages` uses `?before=` cursor instead of `?offset=`. Fetches `limit+1` to determine `hasMore`. Returns wrapped `{messages, hasMore}` response with non-null empty array guard.

- **Domain entity completeness** — `ChatMessage` now includes `userAvatarUrl`, matching the frontend `ChatMessage` type and spec contract. `ChatClient` includes `LastActive` for idle tracking.

- **Test coverage maintained** — All existing test files updated for new signatures. New tests added for rate limiting and cursor pagination. Integration tests updated for spec protocol envelope and auth headers.

- **Frontend unchanged** — `ChatPanel.tsx` and `ChatInput.tsx` already followed the spec protocol (`{type, payload}` envelope, `/ws/chat/{streamId}` URL, all UI states). No frontend changes needed.

- **All tests pass** — 20 backend packages (short mode), 25 frontend test files (152 tests), `go vet` clean, `npx tsc --noEmit` clean.

---

## Verdict

**[REVIEW_PASS]** — All warnings resolved. One style item deferred (adapter duplication, blocked by Go package visibility). Implementation faithfully follows the spec's API contract, implements all edge cases, and maintains full test coverage.

### Resolution Summary

| Warning | Resolution |
|---------|-----------|
| String error comparison | `errors.Is(err, application.ErrRateLimited)` |
| `context.Background()` leak | `context.Context` added to `chatAuth.ValidateToken` |
| Unused constant | Removed |
| Adapter duplication | Deferred (Go package visibility constraint) |
