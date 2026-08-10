# Session Log — 2026-08-10

## Session Summary
Fixed agent-autonomy friction: Zed file-operation prompts, branching from stale branches, over-restrictive commit rules. Pulled `main` in both repos, applied AGENTS.md changes, created PRs.

## Corrections & Root Causes

| # | Event | Root Cause | Fix |
|---|-------|-----------|-----|
| 1 | Features created from old branches instead of `main` | No base-branch rule in AGENTS.md §6.2 | Added "always branch from latest main" rule with fetch-checkout-pull sequence |
| 2 | Zed prompting for every file create/edit/delete inside repo | `move_path` still set to `"confirm"` in Zed settings; AGENTS.md §7.2 was too timid | Changed `move_path` to `"allow"`; rewrote §7.2 to grant full write autonomy with explicit blocklist |
| 3 | Skills-test AGENTS.md merge conflict during stash pop | Upstream `main` had new `gh pr create` backtick tip; section numbering diverged between repos | Resolved by merging both: kept our new rules + retained upstream tip |
| 4 | `git checkout main` fails in git-worktree setup | `main` already checked out in primary worktree | Used `git -C <primary-worktree> pull` as workaround; flagged as edge case in review |

## Questions / Follow-Ups

- [ ] Should the two repos' AGENTS.md be kept in sync automatically (CI check)?
- [ ] Should the branching rule account for git-worktree setups (use `git fetch origin main:main` instead of `git checkout main`)?
- [ ] Should Zed's `default_profile` be enforced at project level via `.zed/settings.json`?
- [ ] Should we create `frontend/.env.example` so new contributors know required env vars?

## Files Changed

- **Modified:** `AGENTS.md` — Sections 6.1, 6.2, 7.2 rewritten (both repos)
- **Modified:** `~/.config/zed/settings.json` — `move_path` → `allow`
- **Modified:** `frontend/next.config.ts` — hardcoded backend URL → `BACKEND_URL` env var (PR #17)
- **Created:** `frontend/.env`, `backend/.env` (gitignored; documented required vars)

## To retro at end

- [x] Trace correction #1 → missing base-branch rule in AGENTS.md
- [x] Trace correction #2 → missing write-autonomy rule + Zed config gap
- [x] Trace correction #3 → repos drifted apart, no sync mechanism
- [ ] Port findings to skills-test

---

## Corrections & Lessons (2026-08-10 — Dev Environment Debugging)

| # | Event | Root Cause | Fix |
|---|-------|-----------|-----|
| 5 | Frontend on port 3001 hung indefinitely, never returned a response | Server component `fetch()` in `page.tsx` called `http://localhost:3000/api/streams/live`; port 3000 was occupied by a git-worktree Next.js server, not the Go backend. The Next.js-to-Next.js request deadlocked. | Killed worktree server on port 3000. Restarted frontend on port 3000 (intended port). |
| 6 | Google OAuth returned `Missing required parameter: client_id` | Docker Compose started the backend container before `GOOGLE_CLIENT_ID` was set in `.env`. `docker compose restart` reuses old container config; `docker compose up -d` is needed to pick up new env vars. | Ran `docker compose up -d backend` to recreate the container with current `.env` values. |
| 7 | Backend `.env` not auto-loaded by Go | Go reads `os.Getenv()`, doesn't auto-load `.env` files. The backend was running in Docker (which loads via docker-compose), not locally. | Confirmed Docker was already running the backend. The local `go run` was unnecessary. |
| 8 | `next.config.ts` had hardcoded `localhost:8081` rewrite destination | No rule requiring configurable backend URLs via env vars. | Changed to `BACKEND_URL` env var with `localhost:8081` default. PR #17. |

---

*Retros: [agent-autonomy](./2026-08-10-agent-autonomy-retro.md), [dev-env-debugging](./2026-08-10-dev-env-debugging.md)*
