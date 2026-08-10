# Retro — Chat Implementation, 2026-08-10

**Session Log:** [2026-08-10-session-log.md](./2026-08-10-session-log.md)
**Spec:** [us4-real-time-chat.md](../live-streaming-platform/us4-real-time-chat.md)
**Branch:** `feat/chat-implementation`

---

## Correction → Rule Mapping

### 1. Concurrent WebSocket writes caused integration test hang

**What happened:** `readPump` was calling `wsjson.Write` directly for errors and pongs while `writePump` was also writing. `nhooyr.io/websocket` requires all writes from a single goroutine.

**Missing rule:** None — this is a library-specific constraint that wasn't obvious from the spec. The spec didn't specify write-pump architecture.

**Action:** Add to `go-chi` skill:
> When using `nhooyr.io/websocket`, route ALL writes through a single goroutine (typically `writePump`). Never call `wsjson.Write` or `conn.Write` from `readPump`. Use a channel (`client.Send`) to pass messages from `readPump` to `writePump`.

### 2. Auth extraction mismatch in WS handler

**What happened:** WS route moved out of auth middleware group, but handler now uses `extractToken()` + `chatAuth` adapter instead of context-based `UserIDFromCtx()`. Test didn't pass auth credentials.

**Missing rule:** The AGENTS.md rule about "API contract fidelity" should explicitly mention that route-level auth changes require test updates for auth credential plumbing.

**Action:** Add to AGENTS.md Section 2.1:
> When moving a route into or out of an auth middleware group, update all integration tests to pass auth credentials through the mechanism the handler now uses (cookies, headers, or context injection).

### 3. Signature changes cascaded to all consumers

**What happened:** Adding `chatAuth` parameter to `NewChatHandler` broke server test. Updating repository interface (`GetMessages` returning 3 values instead of 2, cursor-based params) broke all mocks and tests.

**Missing rule:** None — this is expected. The rule already exists: "After any code change, run the build."

**Action:** Reinforce existing rule in AGENTS.md Section 5.1:
> After changing a function signature or interface, run the FULL build (`go build ./...`) not just the changed package. Go's compiler will catch all call sites, but only if you compile everything.

### 4. Nil logger panic in test error path

**What happened:** Test passed `nil` for logger. When an error path was triggered (the `service_error` test case), `h.logger.Error()` panicked on nil receiver.

**Missing rule:** Tests should use a proper (non-nil) logger. The `testLogger()` helper exists but wasn't used in the handler test.

**Action:** Add to `go-chi` skill:
> In handler tests, always provide a non-nil `*slog.Logger`. Use `slog.New(slog.NewTextHandler(io.Discard, nil))` for silent tests, or `slog.Default()` when debugging. Never pass `nil` as a logger — error paths will panic.

---

## Skill/AGENTS Updates Needed

| File | Rule to Add |
|------|------------|
| `.agents/skills/go-chi/SKILL.md` | WebSocket write-pump pattern for `nhooyr.io/websocket` |
| `.agents/skills/go-chi/SKILL.md` | Test logger: never pass nil `*slog.Logger` |
| `AGENTS.md` Section 2.1 | Auth group changes require test credential updates |
| `AGENTS.md` Section 5.1 | Reinforce: run full build after signature changes |

---

## What Went Well

1. **Spec-first approach caught 12 gaps** — The detailed spec (API contract, protocol envelope, connection lifecycle) made it easy to identify exactly what the implementation was missing. Without the spec, we would have shipped a broken chat.

2. **Frontend already followed the spec** — `ChatPanel` and `ChatInput` were built to the spec from the start. The only issue was the backend not matching. This validates the spec-as-contract approach.

3. **`writePump` single-writer refactor** — Moving all writes to a single goroutine via `sendToClient()` solved the concurrency issue cleanly and is now the canonical pattern for all WebSocket handlers.

4. **Optional auth pattern** — Moving the WS route out of the auth group and handling auth internally with graceful anonymous fallback is the right architecture for read-only + authenticated-write resources.

5. **Token bucket rate limiting** — Lock-free burst checking with proper refill math. Simple, correct, testable.

---

## Open Items (from review)

1. Replace `err.Error() == "rate limited"` string comparison with sentinel error
2. Add `context.Context` parameter to `chatAuth.ValidateToken`
3. Extract duplicated `chatAuthAdapter` to shared package
4. Remove or use `wsPingInterval` constant
5. Simplify whitespace check with `strings.TrimSpace`
