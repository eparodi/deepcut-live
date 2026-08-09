# Retro: Parallel Agent Artifacts + Test Hygiene — 2026-08-08

## What happened

Three incidents during the test-coverage PR (#7):

### #11 — Junk files from parallel agents

Two sub-agents ran in parallel (backend tests + frontend tests) and produced
artifact files with ` 2` suffixes:

```
backend/Makefile 2
backend/db/sqlc 2.yaml
frontend/src/components/ui/Spinner.test 2.tsx
frontend/src/components/ui/Toast.test 2.tsx
frontend/src/lib/api.test 2.ts
frontend/src/test/setup 2.ts
frontend/vitest.config 2.mts
frontend/src/components/LiveStreamCard.test 2.tsx
```

These were not committed but polluted the working tree. Same root cause as
the `ui 2/` directory from the 2026-08-08 end-to-end retro (item #4).

### #12 — Missing `beforeEach` import

`StreamKeyDisplay.test.tsx` used `beforeEach()` without importing it from
vitest. The test passed locally because `vitest.config.mts` has `globals:
true`, but CI's `tsc --noEmit` step failed because TypeScript doesn't
inject vitest globals.

### #13 — Push protection blocked by fake Stripe key

Test data used `sk_live_abcdef...` which matches Stripe's secret key regex.
GitHub push protection blocked the push. Fixed by changing prefix to
`dc_live_`.

## Root cause

| # | Incident | Missing rule |
|---|----------|-------------|
| 11 | Junk files from parallel agents | No rule for cleaning up after sub-agents. Existing `ui 2/` retro didn't produce a concrete guard. |
| 12 | `beforeEach` not imported | `globals: true` in vitest config masks missing imports locally. No rule requiring explicit imports. |
| 13 | Fake Stripe key in test data | No rule for avoiding secret-scanner patterns in test fixtures. |

## Rule changes

### nextjs/SKILL.md — Pre-deploy checklist (new item)

```markdown
- [ ] All vitest functions (`describe`, `it`, `expect`, `vi`, `beforeEach`,
  `afterEach`, etc.) are explicitly imported — do not rely on `globals: true`
```

### AGENTS.md — Section 6.2 (new sub-item)

```markdown
### 6.2 Write Operations

...
- **Clean up after parallel agents.** After spawning sub-agents that write
  files, check for and remove junk artifacts (files with ` 2` suffix,
  duplicated directories) before committing.
```

### AGENTS.md — Section 4.2 (new sub-item)

```markdown
### 4.2 Test Coverage

...
- **Test data must avoid secret-scanner patterns.** Fake keys, tokens, and
  credentials in test fixtures must not match patterns that trigger GitHub
  push protection (Stripe `sk_live_`, `rk_live_`; AWS `AKIA*`; GitHub
  `ghp_*`; etc.). Use obviously-fake prefixes like `test_`, `fake_`, or
  project-specific prefixes like `dc_live_`.
```

## What worked well

- **CI caught #12** — `tsc --noEmit` found the missing import before merge.
  The type-check step proved its value exactly as designed.
- **Push protection caught #13** — GitHub's secret scanner prevented a
  false-positive Stripe key from entering the repo. The test data fix was
  trivial (prefix change).
- **Parallel agents delivered 230 tests** — despite the junk-file cleanup
  overhead, the parallel approach completed backend (86) + frontend (144)
  tests in a single turn.

## Metrics

| Metric | Count |
|--------|-------|
| New incidents | 3 |
| Rules added | 3 |
| Pre-existing issue surfaced | 1 (#11 = same as end-to-end retro #4) |
| PRs merged | 1 (#7) |
| New tests | 230 |

---

*Cross-reference: `2026-08-08-session-log.md` corrections #11-#13*
*Related: `2026-08-08-end-to-end-retro.md` item #4 (same root cause)*
