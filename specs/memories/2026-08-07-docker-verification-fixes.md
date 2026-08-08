# Retro: Docker Verification Fixes — 2026-08-07

## Context

After running `docker compose up`, the containers all started healthy but
several bugs surfaced during endpoint verification. These were latent bugs
that wouldn't appear until the SRS config and actual callbacks were tested.

## Bug 1 — SRS callback URLs missing secret

**Symptom:** `POST /api/srs/callback` returned 403 "invalid srs secret" even
though `SRS_CALLBACK_SECRET=dev-srs-secret` was set.

**Root cause:** `data/srs.conf` had bare URLs:
```
on_publish http://backend:8081/api/srs/callback;
```
SRS appends the stream key via `?key=...` in the body's `param` field, but
it does NOT inject the secret. The secret must be in the conf URL as a
query parameter.

**Fix:** Changed conf URLs to include the secret:
```
on_publish http://backend:8081/api/srs/callback?secret=dev-srs-secret;
on_unpublish http://backend:8081/api/srs/callback?secret=dev-srs-secret;
```

## Bug 2 — SRS callback body consumed twice

**Symptom:** Would have caused `on_publish` to fail silently when SRS sent
a real callback. The dispatch method `SRSCallback` read `r.Body` to
determine the action, then forwarded `r` (with consumed body) to
`SRSOnPublish`/`SRSOnUnpublish` which tried to read it again.

**Root cause:** Go's `http.Request.Body` is a `io.ReadCloser` that is
consumed on first read. Passing the same request to a second handler means
it gets an empty body. This is a common Go HTTP pitfall.

**Fix:** `SRSCallback` now reads the full body into `[]byte`, checks only
the `action` field via `json.Unmarshal`, then restores the body via
`io.NopCloser(bytes.NewReader(bodyBytes))` before dispatching. This lets
each dispatch handler read the body independently.

**Rule to add:** The `go-chi` skill should warn about `r.Body` being
single-use and the `io.NopCloser` restore pattern for dispatch routers.

## Bug 3 — `/api/channel/{userID}` 500 on non-UUID input

**Symptom:** `GET /api/channel/dummy-id` returned 500 instead of 400.

**Root cause:** No input validation on the `userID` path parameter. The
UUID was passed directly to PostgreSQL, which rejected it at the query
level. The error propagated as a 500.

**Fix:** Added `uuid.Parse(userID)` validation in the handler before the
service call, returning 400 with a clear message.

**Rule to add:** The `go-chi` skill should require path parameter validation
at the handler layer before hitting the database. UUIDs, emails, enums,
etc. should be validated in HTTP handlers, not in SQL queries.

## Bug 4 — Junk directory `frontend/src/components/ui 2/`

**Symptom:** Empty directory with a space in the name appeared in the repo.

**Likely cause:** File system glitch during parallel agent file creation
or an accidental extra `mkdir`.

**Fix:** Deleted the empty directory.

## Skill Rules Updated

**File:** `.agents/skills/go-chi/SKILL.md`

Added two new sections:

### DO — Validate path parameters in the handler, not the database

- Parse UUIDs with `uuid.Parse()` before passing to service/repo
- Return 400 with a clear message for invalid formats
- Do not let malformed input reach the database query layer

### DO — Restore request body for dispatch routers

- `r.Body` is consumed on first read — do not pass the same `*http.Request`
  to a second handler that also reads the body
- Pattern: read full body into `[]byte`, check action, restore via
  `io.NopCloser(bytes.NewReader(bodyBytes))`
- This applies to any router that dispatches to sub-handlers based on body
  content (SRS callbacks, webhook routers, etc.)

## Files Changed

- `data/srs.conf` — added `?secret=dev-srs-secret` to callback URLs
- `backend/internal/modules/streams/adapter/http/handler.go` — body restore
  pattern in SRSCallback, UUID validation in GetChannelInfo
- `backend/go.mod` / `go.sum` — added `github.com/google/uuid`
- `docker-compose.yml` — removed obsolete `version` key
- `.agents/skills/go-chi/SKILL.md` — added two new rules (input validation,
  body restore)
