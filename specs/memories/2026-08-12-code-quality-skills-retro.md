# Retro — Reference-repo audit, code-quality refactor, and generic skills rewrite

**Date:** 2026-08-12
**Branch:** `refactor/code-quality-and-generic-skills`
**Goal:** Audit the codebase against high-quality references (Uber Go
Style Guide, Go Code Review Comments, bulletproof-react), fix what the
audit found, then fold the *generic* learnings into the skills so a
cheaper model can maintain quality — and strip project-specific content
out of the skills.

## What was done

1. **References studied:** Uber Go Style Guide, go.dev CodeReviewComments,
   bulletproof-react (project structure, components, API layer, error
   handling).
2. **Two audits** (backend, frontend) produced ~145 findings
   (15 P0, 57 P1, 74 P2).
3. **Code fixes applied** (commit `refactor: apply Go/Next.js
   style-guide quality fixes across both stacks`): see commit message
   for the full list. All builds/tests/lint/`next build` green;
   backend integration tests (testcontainers) green.
4. **Skills rewritten:**
   - `go-chi`: generic rewrite embedding the audit learnings (mutable
     globals, exit-once, handle-once, errors.Is, typed enums, goroutine
     lifetime, WS pump invariant, comma-ok, unified teardown, webhooks,
     subprocess hygiene). SRS/River/project-layout specifics removed.
   - `nextjs`: added API-layer discipline, error≠empty, disposed-flag WS
     cleanup, AbortController races, executed-query pagination, Suspense
     for useSearchParams, route-file completeness, img onError, derived
     state, vi.mock importOriginal trap.
   - All skills de-model-branded (`DeepSeek says` → `Hallucination`)
     and project examples generalized (backend-engineer, spec-driven,
     security-engineer).
   - **New `deepcut-platform` skill** (this repo only): SRS/River/
     paths/env-vars/known-deferred-issues moved here from go-chi.
5. **Skills mirrored** to the sibling `skills-test` repo (all except
   `deepcut-platform`).

## Corrections → rules (trace table)

| Finding class (audit) | Rule destination |
|---|---|
| 3 package-level `sync.Map` globals | go-chi "State & Mutable Globals" |
| 7 global `slog` calls in a service with injected logger | go-chi "Logging" (grep-before-finish rule) |
| 4 unchecked type assertions | go-chi comma-ok rule |
| ForceEnd/interrupt paths leaked recording goroutines | go-chi "every end path stops side effects" (unified teardown) |
| Chat WS pump leak + idle-expiry zombies | go-chi "Connection lifecycle" invariant + remote-close callback |
| OAuth any-error→create-user conflation | go-chi "Match error kinds before fallback behavior" |
| `err == pgx.ErrNoRows` (8 sites) | go-chi errors.Is rule |
| `log.Fatal`/`os.Exit` skipping defers in mains | go-chi "Exit once (run() pattern)" |
| Magic status strings (~15 sites) | go-chi "Enums & Magic Strings" |
| Missing `rows.Err()` in one repo of four | go-chi (audit EVERY loop rule) |
| Constructor panic on bad PEM | go-chi constructors-return-errors |
| WS reconnect after unmount (2 components) | nextjs "disposed-flag" pattern |
| Search race + load-more used live input | nextjs AbortController + executed-query rules |
| Home page raw fetch duplicating unused api fns | nextjs "API Layer Discipline" |
| 8 duplicated format helpers, drifting output | nextjs extract-helpers rule (grep first) |
| Missing error/loading/not-found route files | nextjs route-segment completeness rule |
| Backend-down rendered as "nothing live" | nextjs error≠empty rule |
| `useSearchParams` without Suspense | nextjs Suspense section |
| vi.mock factory dropped ApiError class → instanceof broke | nextjs "vi.mock and class exports" |
| go-chi skill described a layout the repo doesn't use | go-chi "discover the layout" rule; actual layout documented in deepcut-platform |

## Explicitly NOT fixed (need product/schema decisions — tracked in deepcut-platform skill)

- Stream key embedded in public HLS URLs (security)
- VOD heartbeat view-count inflation (needs dedup schema)
- Analytics peak/unique always 0 (needs aggregation step)
- VOD chat replay not implemented (dead UI props + unused API fn)
- Search `category` filter accepted but unimplemented
- WS `OriginPatterns` hardcoded to localhost (needs config threading)
- Dead exports kept (RemoveViewer, GetLiveUsers, etc.) — removal needs
  owner sign-off

## Process notes

- Sub-agent delegation failed mid-session (API credits) — all fixes
  applied directly. The two audit reports were produced before the
  failure and drove the whole session.
- One audit recommendation **rejected**: removing handler logs before
  `render.Error`. Our render layer hides internals from clients
  (generic 500 bodies), so the pre-render log is the only evidence —
  that's handle-once plus observability, not double handling. Encoded
  as an explicit exception in the go-chi skill.
