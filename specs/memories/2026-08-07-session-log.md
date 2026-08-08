# Session Log — 2026-08-07

> Running log of corrections, questions, and lessons from this session.
> Consolidated into a proper retro at end of session/feature.

---

## Corrections & Lessons

| # | What happened | Root cause | Fix / Rule change |
|---|--------------|------------|-------------------|
| 1 | SRS callbacks returned 403 | `srs.conf` missing `?secret=` in callback URLs | Fixed conf; should be documented in go-chi skill |
| 2 | SRS callback body consumed twice | `SRSCallback` read body then forwarded consumed `r` to sub-handlers | Body restore pattern (`io.NopCloser`) added to go-chi skill |
| 3 | `/api/channel/bad-id` returned 500 | No UUID validation, Postgres rejected at query level | `uuid.Parse()` in handler → 400; added to go-chi skill |
| 4 | Junk `ui 2/` directory | File system glitch during parallel agent work | Deleted |
| 5 | Frontend 500 on first load | `node_modules` installed with Node 18, `.next` cache stale, Tailwind native binary missing | Reinstall with correct Node version, delete `.next` |
| 6 | Node version pinning | No `.nvmrc` in frontend | Created `.nvmrc` + added rule to nextjs + expo skills |
| 7 | Pinned Node 22 → user wanted latest | User prefers latest LTS over conservative choice | Changed to Node 24 in `.nvmrc` + both skills |
| 8 | OAuth callback 404 | `BASE_URL` was `localhost:3000` so Google redirected to frontend, but backend handles callback | Changed architecture: Next.js now acts as API tunnel via rewrites |
| 9 | Next.js 16 middleware deprecation | `middleware.ts` renamed to `proxy.ts`, export changed | Ran `npx @next/codemod middleware-to-proxy` |
| 10 | Dashboard "Recoverable Error" SSR 401 | `use(fetchUser())` fired during SSR without auth cookies | Deferred promises to client-only via `useEffect` + `useState`; SSR shows skeleton |

## Questions / Follow-ups

- [ ] Should we add `@tailwindcss/oxide-darwin-arm64` as an explicit dependency to avoid the native binary issue on fresh installs?
- [ ] `docker-compose.yml` version key removed — verify no other services rely on it
- [ ] Next.js 16 deprecates `middleware.ts` → migrate to `proxy.ts`?

## To retro at end

- [ ] Trace each correction to a specific missing rule
- [ ] Update AGENTS.md if any guardrail was missing
- [ ] Update relevant skills with lessons learned
