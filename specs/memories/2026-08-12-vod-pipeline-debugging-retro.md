# Retro: VOD Pipeline Debugging — 2026-08-12

**Session log:** [2026-08-12-session-log.md](./2026-08-12-session-log.md)
**Branch:** `feat/browse-vod-discovery` (PR #23)
**Outcome:** VOD pipeline working end-to-end; review passed with non-blocking
warnings.

## Correction → Missing Rule Mapping

For each correction in the session log, the rule that would have prevented it
and where it now lives.

| # | Correction (root cause) | Missing rule | Where added |
|---|--------------------------|--------------|-------------|
| 1 | River `Insert` fails on unregistered job kind, even for insert-only clients | River requires kind registration in the Workers bundle | go-chi SKILL.md → "River (PostgreSQL Job Queue)" |
| 2/3 | SRS tuning + callbacks never applied — image loads `conf/docker.conf`, not the mounted `conf/srs.conf` | Verify which config file the container entrypoint actually loads | AGENTS.md §10.6 (both repos); go-chi SKILL.md → "SRS Integration" |
| 4 | `hls_ll_enabled` invalid — LL-HLS removed in SRS 5 | Verify feature availability in the ACTUAL external-service version | AGENTS.md §10.7; go-chi SKILL.md → "SRS Integration" |
| 5 | `client_id` unmarshal failure — SRS 5 sends a string conn id | Match handler structs to the actual payload of the installed version | AGENTS.md §10.7; go-chi SKILL.md → "SRS Integration" + webhook example fixed (`ClientID string`) |
| 6 | `on_unpublish` 400 — `DisallowUnknownFields` on a third-party webhook | Never disallow unknown fields on external webhooks | Already in go-chi SKILL.md ("Third-Party Webhooks") — example payload corrected |
| 7 | `duration_seconds` always 0 | SRS doesn't send duration — compute from `started_at` | go-chi SKILL.md → "SRS Integration" |
| 8 | ffmpeg errors invisible (`cmd.Stderr = nil`) | Never discard subprocess stderr | AGENTS.md §10.8; go-chi SKILL.md → "ffmpeg Subprocesses" |
| 9/14 | VOD audio corrupt — SIGKILL loses MP4 moov → AAC extradata gone → TS mux fails | Prefer streaming containers (TS) for kill-prone recordings; verify with a decode pass | AGENTS.md §10.8; go-chi SKILL.md → "ffmpeg Subprocesses" |
| 10 | 25+ stale "processing" rows from earlier failures | Clean up stuck rows when the pipeline that produces them is fixed | Ops note in session log (no generic rule — dev-data hygiene) |
| 11 | psql multi-statement rollback surprise | `psql -c` batches run in one implicit transaction | AGENTS.md §10.9 (both repos) |
| 12 | VOD 404 via proxy — SRS `hls_ctx` master-playlist child URL is root-absolute and loses the proxy prefix | Understand external-service URL rewriting behind reverse proxies | go-chi SKILL.md → "SRS Integration" (`hls_ctx off`) |
| 13 | RTMP test push rejected | Backend stream-key auth works as intended — not a bug | — (documented in session log) |
| — | Search page `useCallback` missing dep would fail CI lint (`--max-warnings 0`) | Run `npm run lint` (not just `tsc`) before claiming CI-green | Already in AGENTS.md §5.3 — reinforced in review |

## Rules Updated This Retro

- `deepcut-live/.agents/skills/go-chi/SKILL.md` — new sections: **SRS Integration**,
  **River**, **ffmpeg Subprocesses**; webhook example fixed to string `client_id`;
  restored missing `## chi Router Patterns` header.
- `skills-test/.agents/skills/go-chi/SKILL.md` — identical content (copied).
- `deepcut-live/AGENTS.md` — §10.6–10.9 added; §2.1 bidirectional verification
  and §5.1 call-site grep synced from skills-test; date bumped.
- `skills-test/AGENTS.md` — §10.6–10.9 added; restored the `## Section 6` header
  that a previous retro edit had accidentally deleted; repaired mangled §5.1.
- Both repos' `AGENTS.md` and `go-chi/SKILL.md` now verify as byte-identical.

## Skill-Drift Debt Discovered

The two repos had drifted before this retro (skills-test had newer WebSocket
rules; deepcut-live was missing the bidirectional-verification and call-site
grep rules; skills-test had lost the Section 6 header). This retro synced them.
**New rule going forward:** when a retro updates a shared skill/AGENTS.md,
apply the same edit to BOTH repos in the same session (see also the user's
standing instruction to keep the two repos in sync).

## Follow-ups (non-blocking, from the PR review)

- VOD cards don't render `thumbnailUrl` (worker thumbnail is invisible on browse/search).
- Failed VODs appear on channel pages but not search — decide the intended behavior.
- `VodView` shows `vod.message` but backend sends `recordingError`.
- `VodDetail.viewerCount` is a stale type field (backend sends `totalViewers`).
- VOD page hardcodes the HLS path instead of using `vod.hlsUrl` (spec task 14).
- `SearchParams.Category` parsed but never filtered.
