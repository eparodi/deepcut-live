# Session Log — 2026-08-10

## Corrections & Root Causes

| # | Issue | Root Cause | Fix |
|---|-------|-----------|-----|
| 1 | WS integration test hanging | `readPump` concurrent writes with `writePump` | Routed all writes through `client.Send` channel via `sendToClient()` |
| 2 | WS integration test failing (client.UserID empty) | WS route moved out of auth group; test didn't pass auth | Added `HTTPHeader` with Cookie to `websocket.DialOptions` |
| 3 | Server test build failure | `chathttp.NewChatHandler` signature changed | Added `chatAuthAdapter` to server test |
| 4 | `nil` logger panic in handler test | Test passed `nil`; error path called `h.logger.Error()` | Used `testLogger()` instead of `nil` |
| 5 | Chat panel never rendered | Backend never returned `streamId` in channel response | Added `StreamID` to `ChannelInfo`, repo query, and population |
| 6 | WebSocket connection failed | `getWsUrl` used Next.js proxy (port 3000); cant proxy WS | Added separate `WS_HOST` pointing to backend directly |
| 7 | Stream offline → infinite reconnect | HTTP 400 returned before WS upgrade; browser got no close code | Accept WS first, then close with code 4001 |
| 8 | Messages appearing twice | React StrictMode double-invoke re-receives `sendInitialBatch` | Dedup by message ID in `onmessage` |
| 9 | Sending messages twice | `<button>` without type defaults to submit; Enter fires both handlers | Added `type="button"` |
| 10 | CI: Auth repo integration tests failing | Pre-existing: `CreateUser` param order mismatch after #18 | Reordered all 10 seed calls |
| 11 | CI: Chat repo cursor test failing | Messages inserted in same microsecond; second-precision cursor filters all | Added 10ms sleeps + `RFC3339Nano` format |
| 12 | CI: sqlc generate failing | Pre-existing: `stream_key` column added without regenerating sqlc output | Ran `sqlc generate` |

## Questions / Follow-Ups

- [ ] Extract `chatAuthAdapter` to shared package (deferred: Go visibility constraint)
- [ ] Make origin patterns configurable via env var
- [ ] Simplify whitespace check with `strings.TrimSpace`
- [ ] Consider per-client deadline timer instead of O(n) `ExpireIdle` scan

## Related Retros

- [2026-08-10-chat-review.md](./2026-08-10-chat-review.md)
- [2026-08-10-chat-retro.md](./2026-08-10-chat-retro.md)
- [2026-08-10-nextjs-websocket-proxy.md](./2026-08-10-nextjs-websocket-proxy.md)
