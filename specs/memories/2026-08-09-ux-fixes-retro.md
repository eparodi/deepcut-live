# Retro: UX Stream Key & Browse Fixes — 2026-08-09

> Traces corrections from the `fix/ux-stream-key-and-browse` PR (#14)
> to missing or insufficient rules. Cross-referenced with
> [session log](./2026-08-09-session-log.md).

---

## Correction #7 — Lint fails in CI but passes locally

**What happened:** The `npm run lint` step in Frontend CI runs with
`--max-warnings 0`. The Navbar's `useEffect` → `fetchUser()` pattern
triggered `react-hooks/set-state-in-effect`, plus 6 pre-existing
unused-import warnings across files not touched by this PR.

**Root cause:** We only ran `tsc --noEmit` and `vitest run` locally.
We never ran `eslint . --max-warnings 0`. The CI lint step is stricter
than our local verification.

**Rule to add:**

```
## AGENTS.md Section 5.3 — Linting

- Fix all lint errors before considering work done. Do not suppress
  warnings unless explicitly asked.
+ Run `npm run lint` (or `eslint . --max-warnings 0`) as part of
+ the pre-push verification checklist, not just `tsc --noEmit`.
```

---

## Correction #8 — Underscore-prefix doesn't suppress ESLint unused-vars

**What happened:** Renaming `streams` to `_streams` in a mock component
didn't suppress `@typescript-eslint/no-unused-vars`. The ESLint rule
doesn't treat `_` prefix as "intentionally unused" by default.

**Root cause:** Assumed underscore-prefix convention works in ESLint
like it does in Go, Python, and some other linters. It doesn't without
explicit `argsIgnorePattern` config.

**Rule to add:**

```
## nextjs skill — ESLint conventions

+ ESLint's `@typescript-eslint/no-unused-vars` does NOT treat `_` prefix
+ as "intentionally unused" by default. To suppress unused parameter
+ warnings, either:
+ - Use `props.param` instead of destructuring, or
+ - Configure `argsIgnorePattern: "^_"` in eslint.config
```

---

## Correction #9 — Double-fetch from useState in useCallback deps

**What happened:** `autoGeneratingKey` (boolean state) was in
`useCallback`'s dependency array. When the key generation started and
the state flipped `false → true`, `fetchData` was recreated, the
`useEffect` re-ran, and a redundant `getMe()` + `getAnalytics()` call
fired.

**Root cause:** Used `useState` for a flag whose sole purpose is to
guard against re-execution of a side effect, not to drive rendering.
`useRef` is the correct tool for this.

**Rule to add:**

```
## nextjs skill — React patterns

+ When a boolean flag exists only to prevent re-execution of a side
+ effect (not to drive rendering), use `useRef` instead of `useState`.
+ Refs don't trigger re-renders and don't need to be in dependency
+ arrays, avoiding cascading effect re-runs.
```

---

## Correction #10 — Node version mismatch causes native binary failures

**What happened:** `npm install` ran with system Node v18 (rolldown
4.x requires ≥v20). The `package-lock.json` and native bindings were
installed for the wrong platform/version, causing vitest to crash with
"Cannot find native binding."

**Root cause:** The `.nvmrc` specifies `v24.19.0` but the agent's shell
doesn't auto-load nvm. Every `npm`/`node` command must be prefixed with
the nvm path.

**Rule to add:**

```
## AGENTS.md Section 1.3 or new Section 2.x

+ Before any `npm` or `node` command in the frontend/ directory, verify
+ the active Node version matches `.nvmrc`:
+   node --version
+ If it doesn't match, use the full nvm path:
+   PATH="$HOME/.nvm/versions/node/v$(cat .nvmrc)/bin:$PATH"
```

---

## Updated Rules (apply these)

1. **AGENTS.md §5.3**: Add `npm run lint` to pre-push checklist
2. **nextjs skill**: Document ESLint `_` prefix behavior + `useRef` vs `useState` pattern
3. **nextjs skill** or **AGENTS.md**: Add Node version verification rule

---

*Cross-referenced: [2026-08-09 session log](./2026-08-09-session-log.md)*
