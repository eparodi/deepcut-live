# Retro — UI/UX enhancement session (frontend)

Date: 2026-08-13
Feature: `feat/ux-ui-enhance` — `specs/ui-ux-audit-v1.md` tasks 1–12.
Session log: `specs/memories/2026-08-13-session-log.md` (second section).

## Corrections → missing rules

| Correction | Missing rule | Where it now lives |
|---|---|---|
| `useMemo` rejected by `react-hooks/preserve-manual-memoization` | The repo uses the React Compiler; manual memoization conflicts | Added to `nextjs` SKILL.md → "React Compiler — no manual memoization" |
| WS test stub captured the inner object instead of the instance | Stubs must capture what the production code actually receives (`this`/instance), not internal helper objects | Session log; too test-specific for a skill rule |
| Flaky identity-fetch timing in chat test | Prefer deterministic tests: drive the failure with an event you control (socket close) instead of racing an async fetch | Session log |
| Accessible-name assertion mismatch on `<Link>` with `<img>` + text | Assert by `href`/`within()`, don't guess `getByRole` name computation | Session log |
| `onSend` contract change (`void` → `boolean`) broke `vi.fn()` semantics | When a callback contract changes, update mock defaults and pin both return paths | Session log |

## What worked

- Audit-first flow: sub-agent review with file:line evidence → spec → orchestrator loop.
- Text-safe color tokens (`--color-primary-text`, `--color-danger-text`)
  computed against real contrast math before committing values.
- Small subtasks with per-subtask verification kept every iteration green.
