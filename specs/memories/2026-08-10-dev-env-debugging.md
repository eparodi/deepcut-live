# Retro — Dev Environment Debugging & Config Fixes

**Date:** 2026-08-10
**Session log:** `specs/memories/2026-08-10-session-log.md` (entries #5–#8)
**PR:** [#17](https://github.com/eparodi/deepcut-live/pull/17)
**Result:** Both servers running, OAuth working, proxy configurable.

---

## What Went Well

- **Systematic debugging** — Traced the hang from curl timeout → server component fetch → port 3000 squatter → Docker container env vars. Each layer was verified independently.
- **Docker was already running** — Backend, Postgres, and SRS were all up via Docker Compose. Only needed to recreate the backend container with updated env vars.
- **Minimal code change** — Only `next.config.ts` needed modification. No architectural changes, no new dependencies.

---

## Root Cause → Rule Trace

| # | Finding | Root cause | Rule to add/update |
|---|---|---|---|
| 5 | Frontend SSR hung because fetch targeted wrong port | Port 3000 was occupied by a worktree's Next.js server. No check for port conflicts from sibling worktrees. | **AGENTS.md §6.2**: Add note: "When using git worktrees, check for port conflicts from other worktrees before starting dev servers." |
| 6 | `docker compose restart` didn't pick up new `.env` | Docker Compose `restart` preserves old container config; `up -d` recreates it. | **nextjs skill** or **go-chi skill**: Add a "Docker Compose" section documenting that `up -d` (not `restart`) is needed to pick up `.env` changes. |
| 7 | Go backend didn't auto-load `.env` | Go uses `os.Getenv()`, no dotenv library. Docker was the intended runtime (docker-compose loads `.env`). | **go-chi skill**: Document that `.env` files are only auto-loaded by Docker Compose, not by `go run`. |
| 8 | Hardcoded backend URL in Next.js rewrites | No rule requiring config via env vars. | **nextjs skill**: Add rule: "Any external service URL in next.config.ts must be configurable via an env var, never hardcoded." |

---

## What Could Be Better

### 1. Port conflict detection

The worktree's Next.js server on port 3000 caused a confusing failure mode (fetch hangs instead of connection refused). A pre-start check could prevent this.

**Recommendation:** Add a `dev` script or Make target that checks if the expected ports are free before starting.

### 2. Env-file discoverability

New contributors need to know which env vars are required. The root `.env.example` exists but is private; `frontend/.env.example` and `backend/.env.example` are missing.

**Recommendation:** Create tracked `.env.example` files in both `frontend/` and `backend/` documenting required vars (without secrets).

### 3. Docker vs. local confusion

It wasn't immediately obvious that the backend was running in Docker (port 8081 was served by `com.docke`). `lsof` shows the Docker process, not the Go binary, which delayed diagnosis.

**Recommendation:** Add a `HOW_WE_WORK.md` or README section documenting the expected local dev setup (Docker Compose for backend, `npm run dev` for frontend).

---

## Skills Updated

None this session — the findings are documented here for future skill updates.

## Cross-reference

- Session log: `specs/memories/2026-08-10-session-log.md` (entries #5–#8)
- PR: https://github.com/eparodi/deepcut-live/pull/17
