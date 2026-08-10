# 2026-08-10 Session Log

## Session: Reduce SRS log noise in Docker console

### Corrections & Learnings

| # | Event | Root Cause | Fix |
|---|-------|------------|-----|
| 1 | SRS poller logged WARN every 5s when SRS unreachable | Poller used global `slog.Warn` without respecting a configured log level | Demoted to DEBUG via instance logger; added LOG_LEVEL env var |
| 2 | Poller used global `slog` package while rest of codebase uses injected `*slog.Logger` | Pattern inconsistency — StreamService lacked a logger field | Added `*slog.Logger` field to StreamService, updated constructor, added nil-safe log helpers |

### Rules to Add/Update

- ⬜ go-chi skill: add rule about services always using injected loggers, never global slog
- ⬜ AGENTS.md: add LOG_LEVEL env var as a standard pattern
- ⬜ docker-compose: add log rotation as a standard for all services

### Follow-ups

- [ ] Consolidate remaining global `slog.Error` calls in `service.go` (lines 113, 220) — `OnStreamEnd` and `ForceEndStream` still use global slog for analytics errors
