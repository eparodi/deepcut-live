# Session Log — 2026-08-13

Feature: Single-thread orchestrator skill (PR #27, branch `feat/orchestrator-skill`).

## Corrections & Root Causes (from PR #27 self-review)

| # | Symptom | Root Cause | Fix |
|---|---------|-----------|-----|
| 1 | Orchestrator skill had no session-log obligation, so DEBUGGER learnings would be lost (`LOOP_LOG.md` is gitignored) | New skill written without cross-checking AGENTS.md §9 | Added session-log + retro obligations to the skill (Progress Tracking item 3, Definition of Done, ALWAYS-CHECKS #6) |
| 2 | Skill claimed "CI enforces `--max-warnings 0`" in its VERIFIED FACTS section | Claim copied from AGENTS.md §5.3 without verifying `.github/workflows/` — actual frontend-ci.yml runs plain `npm run lint`, eslint config has no max-warnings | Removed the claim; added ALWAYS-CHECKS #1 "facts stay verified" |
| 3 | `.gitignore` patterns `PLAN.md` / `LOOP_LOG.md` unanchored | Bare gitignore patterns match the filename at ANY depth | Anchored to `/PLAN.md`, `/LOOP_LOG.md`; added ALWAYS-CHECKS #3 "gitignore anchoring" |
| 4 | "NEVER ask the user" autonomy rule could override AGENTS.md §3.1 stop-and-ask for business rules | Autonomy phrasing too broad | Carved out a mandatory stop-and-ask exception for business-rule ambiguity; added ALWAYS-CHECKS #5 |
| 5 | No Node-version guard before npm/npx commands in the skill | Missed AGENTS.md §5.1 requirement (agent shell doesn't auto-load nvm) | Added guard to verification commands + REVIEWER role; ALWAYS-CHECKS #4 |
| 6 | README structure tree comment misaligned for the new long filename | New line not aligned to the tree's long-name style (2-space like `mobile-engineer`) | Realigned; ALWAYS-CHECKS #2 "docs edits stay consistent" |

## Follow-ups / Questions

- [ ] **AGENTS.md §5.3 claims CI enforces `--max-warnings 0`, but frontend-ci.yml runs plain `npm run lint`.** Either (a) enforce it in CI/eslint config, or (b) correct AGENTS.md. Out of scope for PR #27 — needs a decision.
- [ ] Run the §9.2 retro after PR #27 merges: trace each correction above to the rule that caught it, and verify ALWAYS-CHECKS items 1–6 are the permanent rules (they were added directly to the orchestrator skill).

---

## Session: UX/UI expert role + UI/UX enhancement (feat/ux-ui-expert-role, feat/ux-ui-enhance)

| # | Symptom | Root Cause | Fix |
|---|---------|-----------|-----|
| 1 | eslint `react-hooks/preserve-manual-memoization` rejected a manual `useMemo` in ChatPanel | This repo runs the React Compiler; manual memoization conflicts with its output | Dropped the `useMemo` (compiler memoizes); new rule added to the `nextjs` skill |
| 2 | ChatPanel WS test fired handlers that never ran | The test stub captured the constructor's inner plain object, but the component assigns `onmessage` on the WebSocket instance | Capture `this` (the instance) in the stub constructor |
| 3 | "send while CONNECTING" chat test was flaky by design (depended on identity-fetch timing) | Test raced `getMe` resolution before the send | Replaced with a deterministic "send then socket drops" test that exercises the same failed-delivery UI |
| 4 | Navbar test failed on accessible-name matching (`img alt` + text) | Accessible-name computation differs from naive concatenation | Assert the link by `href` + `within()` the avatar `alt` |
| 5 | `vi.fn()` as `onSend` returned undefined once the contract became `(message) => boolean` | Signature change without updating default mock semantics | ChatInput keeps text on falsy return; tests now pin both true/false paths |
