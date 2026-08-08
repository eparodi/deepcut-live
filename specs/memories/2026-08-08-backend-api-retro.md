# Retro — Backend API Endpoints Implementation

**Date:** 2026-08-08
**Spec:** `specs/backend-api-endpoints.md` (archived: `specs/memories/2026-08-08-backend-api-endpoints.md`)
**Session log:** `specs/memories/2026-08-08-session-log.md` (entries #11, #12)
**Result:** All 19 tasks complete. 6 endpoints live and stable. Full test suite passes (16 packages green).

---

## What Went Well

- **Spec-driven process worked.** PM → Architect → Backend Engineer handoff was clean. The Architect's Design section caught 10 issues in existing code before the Backend Engineer touched a file.
- **Phase ordering was correct.** Fixing response shapes first (Phase 1) created a stable contract baseline. Wiring services next (Phase 2) meant the hard logic had clean shapes to work against.
- **Skill updates happened during the session, not after.** Rules derived from architectural findings were added to `go-chi` and `backend-engineer` skills immediately, so future agents in the same session benefit.

---

## Architectural Findings → Rule Trace

Each of the 10 findings from the Architect's code inspection is traced to a rule that was missing or didn't exist:

| # | Finding | Root cause | Rule added |
|---|---|---|---|
| 1 | `ListLiveStreams` returned bare array, not `{streams, total}` | No rule to cross-reference frontend wrapper types | `go-chi` skill: "Match response shapes to frontend contract" (5-point checklist) |
| 2 | `LiveStream.HlsPath` → `thumbnailUrl` semantically wrong | No pattern for documenting known hacks | Deferred to v2 (thumbnail pipeline). Documented in code comment per Architectural Finding #2. |
| 3 | `Analytics` missing `startDate`/`endDate` | No rule to verify all frontend fields are present | `go-chi` skill: pre-deploy checklist item "all fields present, no extra fields" |
| 4 | `ForceEndStream` returned 404 instead of 409 | No rule for semantic HTTP status codes | `go-chi` skill: error anti-patterns table — `errs.Conflict` row |
| 5 | `ForceEndStream` had no SRS integration | Stub was treated as "done" | `backend-engineer` skill: "Stub implementations are NOT stable" rule |
| 6 | `GetMe` included `createdAt` (not in frontend contract) | No rule to verify no extra fields | `go-chi` skill: pre-deploy checklist + response shape verification |
| 7 | `GetMe` missing `streamCategory` | Same as #3 | Same rule as #3 |
| 8 | `GetAnalytics` didn't validate `period` | No rule for handler-level input validation | `go-chi` skill: pre-deploy checklist "all handler inputs validated" |
| 9 | `UpdateSettings` returned `{"status":"ok"}` | No rule to echo fields back | `backend-engineer` skill: anti-pattern "returning generic success when frontend expects echoed fields" |
| 10 | Viewer count lacked 60-second recency filter | Business logic gap — spec said 60s, code didn't enforce | `backend-engineer` skill: "If Something Seems Off" — flag when spec constraints aren't in query |

### Skills updated

| Skill | Sections modified | New rules |
|---|---|---|
| `go-chi/SKILL.md` | Error anti-patterns table, new "Match response shapes to frontend contract" section, Pre-Deploy Checklist | 8 new rules across 3 sections |
| `backend-engineer/SKILL.md` | Contract-First Rule, "If Something Seems Off" section | 6 new anti-patterns + 2 process rules |

---

## What Could Be Better

### 1. Frontend-type verification should be automated

Right now the rule is "grep the frontend type before writing the handler." This is a human process that can be skipped. A better approach:

- **Option A:** Generate Go response structs from `frontend/src/types/index.ts` at build time (codegen).
- **Option B:** Add an integration test that calls each endpoint and validates the JSON schema against the frontend type definitions.

**Recommendation:** Add this as a backlog item. Not blocking v1.

### 2. Stub detection should be automated

The `ForceEndStream` handler called `streamOps.ForceEndStream` but the service wasn't wired — it returned `errs.NotFound("not implemented")` or similar. There's no compile-time check that an interface method is implemented with real logic vs. a panic.

**Recommendation:** Any interface method that returns a `NotImplemented` error should log a `slog.Warn` at startup. The Pre-Deploy Checklist already covers this ("Dev-default secrets log a `slog.Warn`") — extend it to stubs.

### 3. Pre-existing test environment dependency

The Backend Engineer reported `testutil/db.go` breakage that turned out to be a Docker availability issue. testcontainers requires Docker — if Docker isn't running, all postgres adapter tests fail with a connection error, not a clear "Docker required" message.

**Recommendation:** Add a Docker availability check to `SetupDB` or the TestMain that skips postgres-dependent tests with a clear message when Docker is unavailable.

---

## Cross-reference

- Session log: `specs/memories/2026-08-08-session-log.md` (entries #11, #12)
- Archived spec: `specs/memories/2026-08-08-backend-api-endpoints.md`
- Live spec: `specs/backend-api-endpoints.md` (status: Implemented)
- Skills updated: `.agents/skills/go-chi/SKILL.md`, `.agents/skills/backend-engineer/SKILL.md`
