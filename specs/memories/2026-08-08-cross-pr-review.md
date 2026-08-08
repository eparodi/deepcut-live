# Retro: Cross-PR Code Review — 2026-08-08

## Context

Ran code reviews across all 6 open PRs in parallel. Below are the patterns
that appeared across multiple PRs — these are skill/standards gaps, not
one-off mistakes.

---

## Cross-Cutting Findings

### 🔴 Pattern 1: Zero test coverage (all 6 PRs)

Every PR has zero `*_test.go` or `*.test.*` files. AGENTS.md Section 4.2
requires table-driven tests for backend and render tests for UI. The rule
exists but agents aren't following it.

**Fix:** Add to go-chi and nextjs skills: "Before opening a PR, verify with
`find . -name '*_test.*'`. If zero test files exist, do NOT open the PR."

### 🔴 Pattern 2: Bare error returns (PR #1, #3, #6)

`RegenerateStreamKey` returns `err` without wrapping. The go-chi skill
documents `fmt.Errorf("context: %w", err)` but doesn't explicitly call
out bare returns as an anti-pattern.

**Fix:** Add to go-chi skill: "NEVER return bare errors upstream. Always
wrap with `fmt.Errorf("operation: %w", err)`."

### 🟡 Pattern 3: Missing `rows.Err()` after iteration (PR #2, #4)

Multiple postgres repos iterate `rows.Next()` without checking `rows.Err()`.
Silently swallows mid-iteration errors.

**Fix:** Add to go-chi skill under Database section: "After every `for
rows.Next()` loop, check `if err := rows.Err(); err != nil { return ... }`."

### 🟡 Pattern 4: Silently discarded errors with `_` (PR #2, #3, #6)

`OnStreamEnd` discards `UpdateStreamAnalytics` error. Multiple places.

**Fix:** Add to go-chi skill: "Never use `_ =` to discard errors. If the
error is non-critical, at minimum log it with `slog.Error`."

### 🟡 Pattern 5: Duplicated UI components (PR #1, #5, #6)

`Spinner` component duplicated 3 times. `formatViewerCount` duplicated.
`COALESCE` pattern duplicated across repos.

**Fix:** Add to nextjs skill: "Extract duplicated components/utilities to
shared files. If you see the same code in 3+ places, refactor."

### 🟡 Pattern 6: Hardcoded dev secrets with no production guard (PR #2)

`dev-srs-secret` is the default in both code and config. No warning at
startup.

**Fix:** Add to go-chi skill: "If a secret has a dev default, log a
prominent `slog.Warn` at startup when using the default."

### 🔵 Pattern 7: `defer r.Body.Close()` on GET handlers (PR #2, #4, #6)

Unnecessary but harmless. Creates noise.

**Fix:** Add to go-chi skill: "Only `defer r.Body.Close()` when you
actually read the body. Not needed on GET/HEAD handlers."

### 🔵 Pattern 8: No `DisallowUnknownFields()` on JSON decoders (PR #1, #6)

Already documented in go-chi skill, not being followed.

**Fix:** Move from "nice to have" to pre-deploy checklist item.

---

## Skills Updated

### go-chi/SKILL.md
- Added: "NEVER return bare errors" rule
- Added: `rows.Err()` requirement
- Added: "Never discard errors with `_`" rule
- Added: "Log warning on dev default secrets"
- Added: "Body.Close only when reading body"
- Moved: `DisallowUnknownFields` to pre-deploy checklist

### nextjs/SKILL.md
- Added: "Extract duplicated components" rule
- Added: "Module-level mutable state" warning

---

## PR-Specific Critical Issues

| PR | Critical | Description |
|----|----------|-------------|
| #1 | 🔴 regen key sends no body but backend requires `{confirm:true}` | Already fixed in session |
| #1 | 🔴 streamKey never populated in GET /api/me | Already fixed (omitempty) |
| #1 | 🔴 LiveStream type mismatch frontend/backend | Needs fix |
| #1 | 🔴 analytics/force-end routes not registered | Already registered in hex branch |
| #3 | 🔴 PR contains zero chat code (wrong branch?) | Needs investigation |
| #4 | 🔴 ViewerHeartbeat is a no-op stub | Needs implementation |
| #4 | 🔴 Missing `rows.Err()` in scanVODs | Needs fix |
| #5 | 🔴 hls.js not installed → build broken | `npm install hls.js` |
| #6 | 🔴 analytics/force-end backend routes missing | Needs routes registered |
