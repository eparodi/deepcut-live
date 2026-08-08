# Retro: CI Pipelines + Frontend Test Suite — 2026-08-08

## What we built

CI pipelines for both backend and frontend, plus a Vitest + Testing Library test
suite with 37 tests across 4 files. Went through a 4-round review cycle that
caught 9 issues.

## Session log → retro trace

| # | Correction | Missing rule | Where added |
|---|-----------|-------------|-------------|
| 1 | `sqlc@latest` non-deterministic | No CI tool version pinning rule | — (see below) |
| 2 | `curl\|tar` binary download without checksum | No CI binary integrity rule | — (see below) |
| 3 | `mockResolvedValue` leak in api.test.ts | Existing test isolation conventions not followed | N/A (review caught it) |
| 4 | `vitest.config.mts` `.pathname` breaks on Windows | No cross-platform path rule for Node.js | — (minor, review caught it) |
| 5 | `GO_VERSION` hardcoded instead of `go-version-file` | No rule for sourcing Go version from go.mod in CI | — (see below) |
| 6 | `localhost:3000` hardcoded in api.test.ts | Existing DRY conventions | N/A (review caught it) |
| 7 | `useToast` tested via mutable ref instead of `renderHook` | No specific hook testing rule | N/A (review caught it) |
| 8 | `migration-check` job missing `setup-go` | CI design oversight | N/A (review caught it) |
| 9 | Local Node v18 couldn't run vitest 4.x | `.nvmrc` exists but not auto-loaded by shell | N/A (tooling issue) |

## New rules added

### AGENTS.md — Section 4.4: CI/CD Conventions (new)

Three rules added from this retro:

```markdown
### 4.4 CI/CD Conventions

- **Pin tool versions.** Every CLI tool installed in CI must use an exact
  version, never `@latest`. This includes `go install`, `npm install -g`,
  and any direct binary downloads.
  - ❌ `go install example.com/cmd/tool@latest`
  - ✅ `go install example.com/cmd/tool@v1.2.3`

- **Verify binary integrity.** Downloaded binaries must be checksum-verified
  or installed via a package manager (`go install`, `npm ci`, `apt`). Avoid
  `curl | bash` and `curl | tar | sudo mv` patterns.
  - ❌ `curl -L url | tar xvz && sudo mv binary /usr/local/bin`
  - ✅ `go install example.com/cmd/tool@v1.2.3`

- **Source versions from project files.** Go version must be read from
  `go.mod` via `go-version-file`, not hardcoded. Node version must be read
  from `.nvmrc` via `node-version-file`.
  - ❌ `go-version: "1.22"` (hardcoded)
  - ✅ `go-version-file: backend/go.mod`
```

### nextjs/SKILL.md — Pre-deploy checklist addition

Added one item to the existing pre-deploy checklist:
```
- [ ] Vitest config uses `fileURLToPath` for cross-platform path aliases
```

## What worked well

- **4-round review cycle** caught all 9 issues with zero false positives.
  The reviewer skill's severity levels (🔴/🟡/🔵) made it easy to prioritize
  fixes before the retros.
- **Parallel agent delegation** for backend CI + frontend CI produced
  complete, working workflows in a single turn with no file conflicts.
- **Vitest + Testing Library** integration was smooth — jsdom environment,
  `@/` path aliases, and `renderHook` all worked on first try after fixing
  the Node version.
- **Test coverage matches the spec** — every component state from the
  nextjs pre-deploy checklist is tested (loading, empty, error, populated,
  edge cases like viewer count formatting).

## What to improve

- **CI rules didn't exist before the review.** The 3 CI-specific issues
  (#1, #2, #5) all trace back to the same gap: no CI conventions in any
  skill. The new AGENTS.md Section 4.4 fills this gap, but a dedicated
  CI/CD skill (like the existing `go-chi` and `nextjs` skills) would give
  agents more detailed pipeline patterns to follow from the start.
- **Local Node version mismatch** (#9) is a recurring issue (also happened
  in the 2026-08-07 session). The `.nvmrc` exists and CI reads it, but
  local shells don't auto-switch. Consider adding a `.node-version` file
  or a `pre-commit` hook that warns on version mismatch.
- **Frontend test suite is still small.** 37 tests cover the 4 most
  testable units (Spinner, Toast, LiveStreamCard, api). The remaining 20
  components and 6 pages have zero tests. This should grow over subsequent
  PRs.

## Metrics

| Metric | Count |
|--------|-------|
| Review rounds | 4 |
| Total findings | 9 |
| Findings fixed | 9 |
| Rules added to AGENTS.md | 3 |
| Rules added to nextjs SKILL.md | 1 |
| Merged PRs | 1 |
| Test files created | 4 |
| Total tests | 37 |
