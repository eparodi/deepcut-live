# Session Log — 2026-08-09

## Session Summary
Created the Security Engineer role and performed initial security audit of DeepCut Live.

## Corrections & Root Causes

| # | Event | Root Cause | Fix |
|---|-------|-----------|-----|
| 1 | No security role existed in the workflow | Security auditing was a gap in the pipeline | Created `security-engineer` skill with comprehensive knowledge base |
| 2 | WebSocket chat has no auth | User identity from untrusted query params | Flagged as VULN-01 (Critical) — needs JWT auth on WS upgrade |
| 3 | WebSocket origin verification disabled | `InsecureSkipVerify: true` | Flagged as VULN-02 (Critical) — needs origin checking |
| 4 | No rate limiting anywhere | Missing middleware | Flagged as VULN-03 (High) |
| 5 | Ephemeral JWT keys on restart | Auto-generated keys with no persistence | Flagged as VULN-04 (High) |
| 6 | Predictable SRS secret default | `dev-srs-secret` hardcoded fallback | Flagged as VULN-05 (High) |
| 7 | Navbar `useEffect` calling `setState` flagged as lint error in CI | React Compiler's `react-hooks/set-state-in-effect` rule treats fetch-on-mount as a violation. The same pattern is already used in `dashboard/page.tsx` with an eslint-disable comment, but the new Navbar didn't include it. | Added `eslint-disable-next-line react-hooks/set-state-in-effect` to Navbar's `useEffect`. Also cleaned up 6 pre-existing unused import warnings across `loading.tsx`, `api.ts`, `VideoPlayer.test.tsx`, `proxy.test.ts` that were now failing CI due to `--max-warnings 0`. |
| 8 | `_streams` underscore prefix didn't suppress `@typescript-eslint/no-unused-vars` | ESLint's `no-unused-vars` rule doesn't treat `_` prefix as "used" by default (unlike some other linters). The destructured mock parameter `_streams` still triggered the warning. | Changed to `props.streams` pattern — avoid destructuring unused parameters in mock components. |
| 9 | Double-fetch on first dashboard load (review feedback) | `autoGeneratingKey` was in `useCallback` deps array. When it flipped `false → true`, `fetchData` recreated, `useEffect` re-ran, causing a redundant second `getMe()` + `getAnalytics()` call. | Changed `autoGeneratingKey` from `useState` to `useRef` — refs don't trigger re-renders or effect re-runs. |
| 10 | `package-lock.json` installed with Node v18 caused rolldown native binding mismatch | Ran `npm install` with system default Node v18 (rolldown/vitest 4.x require ≥v20). The `.nvmrc` specifies v24.19.0 but the agent's shell doesn't auto-load nvm. | Reinstalled with explicit `PATH="$HOME/.nvm/versions/node/v24.19.0/bin:$PATH" npm install`. This should be automated in the agent setup. |

## Questions / Follow-Ups

- [ ] Should rate limiting be implemented via chi middleware or a reverse proxy?
- [ ] Should the security audit run on every PR or only on release branches?
- [ ] Do we need a dependency vulnerability scanner in CI (e.g., nancy, govulncheck)?
- [ ] Should we add a `.env.example` file with documented security requirements?
- [ ] Should the orchestrator skill mention that `/tmp/` may not be writable in all environments?
- [ ] Should we add a pre-commit hook to validate the CI polling script against `gh pr checks --help` output?
- [ ] The `spec-driven` skill's CI polling script was fixed — need to port the fix back to `skills-test` repo too.
- [ ] Should AGENTS.md or skill files mandate `PATH=...nvm... npm install` for frontend operations to avoid Node version mismatches?
- [ ] Should we add a rule to always run `npm run lint` locally before pushing (not just `tsc --noEmit`)?

## Files Changed

- **Created:** `.agents/skills/security-engineer/SKILL.md` — New security engineer skill
- **Modified:** `.agents/skills/spec-driven/SKILL.md` — Updated Phase 5 to run QA + Security in parallel
- **Modified:** `HOW_WE_WORK.md` — Added Security Engineer to role table and parallel execution docs

## To retro at end

- [x] Trace correction #1 to missing rule (test skill scripts against real CLI output)
- [x] Trace correction #2 to missing rule (account for all check states)
- [x] Update spec-driven skill with fixes
- [ ] Port CI polling fix back to skills-test repo
- [x] Trace correction #7 (lint in CI) → retro: [2026-08-09-ux-fixes-retro.md](./2026-08-09-ux-fixes-retro.md)
- [x] Trace correction #9 (double-fetch) → retro: [2026-08-09-ux-fixes-retro.md](./2026-08-09-ux-fixes-retro.md)
- [x] Trace correction #10 (Node version) → retro: [2026-08-09-ux-fixes-retro.md](./2026-08-09-ux-fixes-retro.md)

---

*Retros: [orchestrator-pipeline](./2026-08-09-orchestrator-pipeline-retro.md), [ux-fixes](./2026-08-09-ux-fixes-retro.md), [security-engineer-role](./2026-08-09-security-engineer-role-retro.md)*
