# Session Log — 2026-08-12

Feature: VOD processing pipeline debugging (PR #23 branch `feat/browse-vod-discovery`).

**Retro:** [2026-08-12-vod-pipeline-debugging-retro.md](./2026-08-12-vod-pipeline-debugging-retro.md)

## Corrections & Root Causes

| # | Symptom | Root Cause | Fix |
|---|---------|-----------|-----|
| 1 | VOD stuck in "processing" forever | Jobs never enqueued — River requires the job kind registered in the Workers bundle even on insert-only clients | `noopWorker` registration in `vods/adapter/river/queue.go` (commit 542ccdca); verified after redeploy |
| 2 | Recording ffmpeg `exit status 8` (404) then `exit status 255` | (a) SRS runs with default 10s-fragment HLS so the playlist 404s for ~16s; (b) ADTS AAC in TS segments can't be muxed to MP4 without `-bsf:a aac_adtstoasc` | Mounted tuned config at `conf/docker.conf` (image ignores `srs.conf`), added `-bsf:a aac_adtstoasc` |
| 3 | SRS config tuning had no effect (10s fragments, no callbacks) | `ossrs/srs:5` entrypoint loads `conf/docker.conf`, NOT `conf/srs.conf` | `data/docker.conf` mounted over `/usr/local/srs/conf/docker.conf` |
| 4 | `illegal vhost.hls.hls_ll_enabled` — SRS failed to start | SRS 5.0 removed LL-HLS entirely (`hls_ll_enabled` no longer a valid directive) | Dropped LL directives; 2s fragments only |
| 5 | `on_publish` rejected: `cannot unmarshal string into Go struct field .client_id of type int` | SRS 5 http_hooks send `client_id` as a string connection id | `srs_client_id` column → TEXT (migration 000004); service/handler/repo signatures → string |
| 6 | `on_unpublish` returned 400 | Handler used `DisallowUnknownFields()` but SRS sends many extra fields | Removed `DisallowUnknownFields` (matches on_publish handler) |
| 7 | `duration_seconds` always 0 | SRS doesn't send duration in the callback | `OnStreamEnd` computes from `started_at` when caller passes 0 |
| 8 | ffmpeg errors invisible (stderr discarded) | `cmd.Stderr = nil` in recording goroutine | Capture stderr into buffer, log last 500 chars |
| 9 | Corrupt MP4s after SIGKILL (moov atom missing) | Plain MP4 loses moov on kill | Fragmented MP4 flags (`frag_keyframe+empty_moov+default_base_moof`) |
| 10 | Stale dev DB rows stuck in "processing" (25+ streams) | Enqueue failure during earlier broken runs | Marked `failed` with explanatory `recording_error`; removed junk recordings |
| 11 | psql multi-statement `-c` rolled back silently | psql wraps multiple statements in one implicit transaction | Run one statement per invocation for destructive cleanup |
| 12 | VOD player 404: browser requested `/vods/{id}/index.m3u8?hls_ctx=...` (no `/hls` prefix) | SRS `hls_ctx` (default on) wraps every playlist in a master playlist pointing at a **root-absolute** child URL `/vods/...?...hls_ctx=...`; hls.js resolves it against `localhost:3000`, losing the `/hls` proxy prefix | `hls_ctx off;` in `data/docker.conf` — SRS now serves the raw playlist with relative segment URIs. VOD **and** live playback verified through the `/hls/*` proxy (200) |
| 13 | Random-key RTMP test push rejected with I/O error | Backend `on_publish` callback correctly rejects unknown stream keys (auth working as intended) | Use a DB-seeded test user key for simulated streams |
| 14 | VOD player: "Media error — attempting to recover" loop, garbage audio | Recording MP4 loses its moov atom on SIGKILL → AAC track extradata gone → worker's `-c:a copy` to TS muxes corrupt ADTS ("AAC bitstream not in ADTS format and extradata missing") | Record to **MPEG-TS** instead of MP4 (streaming container, no moov, ADTS passthrough). Repaired the affected VOD (1507795f) by re-muxing with `-c:a aac` |

## Verified End-to-End (simulated OBS via host ffmpeg RTMP push)

- on_publish → stream created immediately with string `srs_client_id` ✓
- Recording starts after first HLS segment (~2–5s), valid fragmented MP4 ✓
- on_unpublish → 200, `duration_seconds` computed ✓
- River job enqueued & processed by `cmd/worker` ✓
- `recording_status` → `ready`, `vod_hls_path` + `vod_thumbnail_path` set ✓
- VOD HLS served by SRS at `http://localhost:8080/vods/{id}/index.m3u8` (200) ✓
- VOD thumbnail served (200) ✓

## Follow-ups / Questions

- [x] First 2–5s of every stream is missing from the recording (HLS availability delay). RTMP-pull recording would capture from t=0 — worth a follow-up spec?
- [x] Worker transcode output prints image-sequence warning; `-update 1` would silence it (cosmetic).
- [x] `TestFailedVODsExcluded` + other integration tests passed before these changes — re-run full `cmd/server` integration suite to confirm (ran `go test ./...` — all packages pass).
- [x] River retention: completed/discarded jobs accumulate in `river_job` (8 rows now) — add a retention policy later.

## Round 2 — Review Follow-ups (TDD demo)

| # | Item | How resolved |
|---|------|--------------|
| 1 | VOD cards don't render `thumbnailUrl` | Test-first: 3 red tests → implementation. Commits `1c65d3ab` (red) + `6d265649` (green) |
| 2 | Failed VODs on channel pages | Test-first: repo test (red) → `ListVODs` filter (green). Product decision: hide failed everywhere public |
| 3 | `VodView` shows `vod.message` instead of `recordingError` | Test-first: test moved to `recordingError` (red) → VodView fixed; `message`/`viewerCount` removed from `VodDetail` |
| 4 | Stale `viewerCount` type field | Removed (compile-checked via fixtures) |
| 5 | VOD page hardcodes HLS path | Now prefers `vod.hlsUrl` with convention fallback |
| 6 | Avatar `<img>` without `onError` | Test-first fallback tests → shared `lib/fallbacks.ts` constants |
| 7 | `fmt.Println` in worker | → slog logger |
| 8 | Duplicate SRS URL derivation | Extracted `srsHTTPURL()` helper |
| 9 | Category param unfiltered | No change — documented non-goal (accepted for future use) |
| 10 | Spec said "remove `thumbnailUrl` until backend implements" | Condition satisfied — kept, added Implementation Notes to `specs/browse-vod-discovery.md` |

**New learnings for the retro:**
- Red test commits intentionally fail CI — always push them together with (or after) the green commit so the branch HEAD stays green.
- Spec conditions like "remove field X until backend implements it" go stale when a parallel feature implements it — when the condition flips, keep the contract and log an Implementation Note instead of blindly following the stale instruction.
