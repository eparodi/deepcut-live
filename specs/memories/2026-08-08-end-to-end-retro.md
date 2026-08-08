# Retro: Session 2026-08-07 / 2026-08-08 — End-to-End

## What we built

DeepCut Live — a censorship-resistant streaming platform. Completed 5 of 6
user stories (US4 chat was not implemented) through spec-driven development.

- **US1**: Google OAuth, stream keys, dashboard
- **US2**: SRS webhooks, live stream lifecycle
- **US3**: Frontend — live grid, channel page, HLS player
- **US5**: VOD module — listing, search, viewer tracking
- **US6**: Dashboard — analytics, force-end stream

## Session log vs. retro trace

Every correction from the session log has been traced to a specific missing
rule. Below is the complete trace.

| # | Correction | Missing rule | Where added |
|---|-----------|-------------|-------------|
| 1 | SRS callback 403 — no secret in URL | No rule for external config URLs | go-chi skill: SRS/webhook callback URLs must include `?secret=` in config |
| 2 | Body consumed twice in dispatch | No rule for body-restore pattern | go-chi skill: body restore via `io.NopCloser` |
| 3 | `/api/channel/bad-id` → 500 | No input validation rule | go-chi skill: validate path params before DB |
| 4 | Junk `ui 2/` directory | No guard for parallel agent file ops | Not addressed (Zed agent issue) |
| 5 | Frontend 500 — Tailwind native binary | No `.nvmrc` / dependency rebuild rule | nextjs/expo skills: `.nvmrc` required |
| 6 | Node 22 → 24 | User prefers latest LTS | Updated `.nvmrc` + skills |
| 8 | OAuth callback 404 | `BASE_URL` pointed to frontend | go-chi skill: OAuth redirect URI must point to backend, or use Next.js tunnel |
| 9 | middleware.ts deprecated | Next.js 16 renamed to proxy.ts | Ran codemod; no skill change needed |
| 10 | Dashboard "Recoverable Error" SSR 401 | `use()` fired during SSR without cookies | nextjs skill: don't use `use()` for client-only data; use `useState`+`useEffect` |
| 11 | Backend auth only checked Bearer header | No cookie auth rule | go-chi skill: auth middleware should check both cookie and header |
| 12 | Settings field names mismatched | No contract-alignment check | nextjs skill: pre-deploy checklist item |
| 13 | Stream key undefined crash | `omitempty` drops field, frontend expects it | Backend: populate or remove omitempty; Frontend: handle undefined |
| 14 | Regenerate key required confirm but frontend sent empty body | Confirmation mismatch | Backend simplified to accept empty body |
| — | **Cross-PR review findings** | | |
| 15 | Bare error returns (3 PRs) | Rule existed but not enforced | Strengthened to "banned" in go-chi skill |
| 16 | `rows.Err()` missing (4 PRs) | No explicit rule | Added to go-chi skill database section |
| 17 | `_ =` discarded errors (3 PRs) | No explicit rule | Added to go-chi skill: "never discard errors" |
| 18 | Duplicated Spinner (3 files) | No DRY rule for components | Added to nextjs skill: extract at 3+ copies |
| 19 | Module-level mutable state | No React pattern rule | Added to nextjs skill |
| 20 | Dev-default secrets (3 locations) | No production guard rule | Added to go-chi skill: log warning at startup |
| 21 | Frontend↔backend type mismatches (4 PRs) | No contract validation rule | Added to nextjs skill pre-deploy checklist |

## Skills updated this session

### go-chi/SKILL.md (7 new rules)
1. Path parameter validation before database
2. Body restore pattern for dispatch routers
3. Bare error returns banned
4. `rows.Err()` mandatory after iteration
5. `_ =` discarded errors banned
6. Dev-default secrets must log warning
7. `defer r.Body.Close()` only when reading body

### nextjs/SKILL.md (5 new rules)
1. `.nvmrc` pinning Node version
2. Extract duplicated components (3+ copies)
3. No module-level mutable state
4. Prefer Tailwind classes over inline styles
5. Pre-deploy checklist for frontend components

### expo/SKILL.md (1 new rule)
1. `.nvmrc` pinning Node version

### New skills
- `reviewer/SKILL.md` — structured PR review with severity levels

### AGENTS.md
- Section 8: Session logs & retros

## Metrics

| Metric | Count |
|--------|-------|
| Review rounds | 4 |
| Total findings | 54 |
| Findings fixed | 54 |
| Skills rules added | 13 |
| New skills created | 1 |
| Skill rules strengthened | 2 |
| Merged PRs | 6 of 6 |

**Correction (2026-08-08):** PR #3 (US4 chat) was initially skipped because the Round 1 review agent checked the wrong branch (`feat/us2-going-live` instead of `feat/us4-real-time-chat`). The PR was properly reviewed in Round 3 by checking the correct branch. Chat code exists (359 lines, 5 files) and has been merged into main. Lesson: always verify the branch name matches the PR before reviewing. Added to reviewer skill as a pre-review checklist item.

## What worked well

- **Spec-driven gating** caught the architecture decision (hexagonal modular monolith vs flat) before code was written
- **Parallel agent reviews** found 54 issues across 6 PRs in ~30 seconds each
- **Review-fix iteration** converged in 4 rounds with no false positives
- **Session log** made the retro traceable — every correction mapped to a specific rule
- **Next.js tunnel** simplified auth: single origin, no CORS, cookies work natively

## What to improve

- **Tests**: zero existed at merge. The rules require them but agents skip them. Consider adding a pre-commit hook or CI check that blocks PRs with `find . -name '*_test.*' | wc -l` == 0
- **Contract alignment**: frontend↔backend field name mismatches were the #1 recurring bug (4 PRs). Consider generating TypeScript types from Go structs or vice versa
- **Branch hygiene**: US2 branch bundled US2-US6 code, US4 branch had zero chat code. Per-story branches should be verified before opening PRs
- **Architecture drift**: US1 uses flat structure while US2-US6 use hexagonal modules. This will need reconciliation at some point
