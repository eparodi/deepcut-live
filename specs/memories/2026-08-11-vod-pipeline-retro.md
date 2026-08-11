# Retro: VOD Processing Pipeline

**Date:** 2026-08-11  
**Spec:** `specs/vod-processing-pipeline.md`  
**PR:** [#24](https://github.com/eparodi/deepcut-live/pull/24)

## Corrections Made During Session

| # | Issue | Root Cause | Fix | Rule to Add |
|---|-------|-----------|-----|-------------|
| 1 | `recording_status` stuck at `'pending'` | DB default is `'pending'`, never transitioned to `'ready'`. No post-processing pipeline existed. | Added River queue + ffmpeg worker to set status after processing. | When a DB column has a finite state machine, verify every transition is implemented before declaring the feature done. |
| 2 | Frontend called `/api/search` but backend had `/api/vods` | Frontend API client was out of sync with backend routes. | Updated `searchVods()` to call `/api/vods` with correct params. | Always grep backend route registrations and frontend API calls to verify they match before writing code. |
| 3 | `HlsPath` mapped to `thumbnailUrl` JSON tag | Stopgap hack: HLS playlist URL was serialized as thumbnail, causing broken `<img>` tags in browser. | Added proper `ThumbnailUrl` field, hid `HlsPath` via `json:"-"`, fixed frontend hacks in `ChannelView` and `GoLivePreview`. | Never use a field for two semantically different purposes via JSON tags. Add a proper field. |
| 4 | Worker exited immediately after start | River v0.43 `client.Start()` is non-blocking — launches goroutines and returns. Process exited because nothing kept it alive. | Added `<-ctx.Done()` after `Start()` to block until shutdown signal. | Always read the exact version's API docs for third-party libraries. Non-blocking `Start()` patterns vary between versions. |
| 5 | River migration failed with "relation river_queue does not exist" | River v0.43 requires explicit `rivermigrate.Migrate()` call before `NewClient()`. Auto-migration doesn't happen on `Start()`. | Added `rivermigrate` migration step before client creation. | Check if a job queue library requires explicit schema migration before use. |
| 6 | Migration failed with "duplicate key" on restart | Previous crash left partially-created tables. Migration tried to recreate them. | Check `ExistingVersions()` before migrating; if tables exist despite error, continue. | Handle idempotent startup: check if schema already exists before migrating. |
| 7 | Live thumbnail goroutine never started | SRS poller creates streams directly without calling `OnStreamStart`, bypassing the thumbnail startup hook. | Added `startLiveThumbnail`/`stopLiveThumbnail` calls in the poller's stream create/end paths. | When there are multiple code paths that create the same entity, verify all paths trigger the same side effects. |
| 8 | ffmpeg exit code 8 (can't open input) | HLS URL used `__defaultVhost__/live/` path but SRS Docker image writes to `live/` directly (ignores `hls_path` config). | Changed to `http://srs:8080/live/{key}.m3u8`. | Always verify file paths inside the actual Docker container, not just from config files. Docker images may override config defaults. |
| 9 | Thumbnail file existed but SRS returned 404 | SRS serves from `/usr/local/srs/objs/nginx/html/`, not `/data/hls/` (config `dir` directive ignored by Docker image). | Added bind mount: `./data/hls/thumbnails:/usr/local/srs/objs/nginx/html/thumbnails`. | Verify HTTP serving paths inside the container with actual requests. Config directives may be overridden by the base image. |
| 10 | Broken thumbnail with no fallback | `<img>` rendered with non-null `src` pointing to non-existent file; no `onError` handler. | Added `onError` that replaces `src` with inline SVG placeholder. | Every `<img>` with a potentially-missing `src` must have an `onError` fallback. |
| 11 | "Untitled stream" for all poller-created streams | Poller passed `nil` title to `CreateStream` instead of fetching user's configured `stream_title`. | Extended `AuthRepo` with `GetStreamSettings()`, poller fetches title before creating stream. | When creating an entity with user-configured fields, always fetch the user's current settings. |

## What Went Well

- **Interface abstraction**: `VODQueue` interface made River integration clean and swappable
- **Nil-safe dependencies**: Queue, hub, and logger all nil-checked before use
- **Graceful degradation**: Queue enqueue failure logs but doesn't crash the callback
- **Separate worker binary**: ffmpeg crashes isolated from HTTP server
- **Entrypoint migrations**: Both server and worker auto-run DB + River migrations on startup
- **Real-time debugging**: curl + docker exec enabled fast diagnosis of path/network issues

## Rules to Add to AGENTS.md / Skills

1. **Docker path verification**: When config files specify paths, verify the actual paths inside the running container with `docker compose exec <svc> find ...`. Docker base images may override config defaults.
2. **All code paths for entity creation**: When multiple code paths create the same entity (callbacks vs pollers), audit all paths for side effects (thumbnails, notifications, status updates).
3. **`<img>` error handling**: Every `<img>` with a potentially-missing `src` must have an `onError` fallback that either hides the image or shows a placeholder.
4. **Third-party library version checking**: Always verify the exact API of the installed version (not the latest docs) for breaking changes like non-blocking `Start()`.
5. **Schema migration idempotency**: Startup migration code must handle partially-applied states from previous crashes (check if tables exist before failing).
