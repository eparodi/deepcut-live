# US2 — Going Live

**Status:** Design
**Parent:** `_shared.md`
**Depends on:** US1 (streamer account + stream key)
**Blocks:** US3, US4, US5

## User Story

As a streamer, I want to start and stop my stream so that viewers know
when I'm live and can watch in real time.

## Acceptance Criteria

- Given OBS is streaming to the RTMP ingest, When the SRS server receives the stream, Then my channel status changes to "Live" and appears on the homepage live list
- Given I am live, When I stop streaming in OBS, Then my channel status changes to "Offline" within 30 seconds and the stream ends gracefully
- Given I am live, When a viewer navigates to my channel page, Then they see the live video player with HLS playback
- Given I am live, When the stream drops unexpectedly (OBS crash, network loss), Then my channel shows "Stream Interrupted" for up to 60 seconds before showing "Offline"

## Edge Cases

- What happens if SRS receives an RTMP stream with an invalid key? (Reject connection, log attempt)
- What happens if the streamer's key was regenerated while OBS is connected? (Existing connection stays; new key needed for next stream)
- What happens if SRS process crashes mid-stream? (Backend detects SRS is down, marks all channels as "Stream Interrupted")
- What happens during the 60-second interruption window if the streamer reconnects? (Status goes back to "Live", stream resumes, VOD recording continues from reconnect point)

---

## API Contract

### POST /api/srs/callback

**Purpose:** Webhook called by SRS on stream publish/unpublish events.

**Authentication:** Shared secret in `Authorization: Bearer <secret>` header (configured in SRS `http_hooks`).

**Request (on_publish):**
```json
{
  "action": "on_publish",
  "client_id": 12345,
  "ip": "192.168.1.100",
  "vhost": "__defaultVhost__",
  "app": "live",
  "stream": "sk-a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "param": ""
}
```

**Request (on_unpublish):**
```json
{
  "action": "on_unpublish",
  "client_id": 12345,
  "ip": "192.168.1.100",
  "vhost": "__defaultVhost__",
  "app": "live",
  "stream": "sk-a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "param": ""
}
```

**Success Response 200:** `0` (SRS expects integer 0 for success).

**Error Response (reject stream):** `1` (non-zero = SRS rejects the connection).

**Side effects on on_publish:**
- Look up user by stream key (hash the `stream` field, match against `users.stream_key_hash`)
- If match: create `streams` row with status=`live`, set `users.is_live=true`, update HLS path
- If no match: return 1 to reject

**Side effects on on_unpublish:**
- Set `streams.status='offline'`, set `ended_at=now()`, compute `duration_seconds`
- Set `users.is_live=false`
- Set `streams.recording_status='processing'` (VOD processing)

---

### GET /api/streams/live

**Purpose:** List all currently live channels for the homepage grid.

**Authentication:** None.

**Query params:**
```
?page=1&limit=20
```

**Success Response 200:**
```json
{
  "streams": [
    {
      "streamId": "uuid",
      "streamerId": "uuid",
      "streamerName": "Alice Streamer",
      "streamerAvatarUrl": "https://...",
      "title": "Late night coding session",
      "category": "Programming",
      "hlsUrl": "/hls/sk-a1b2c3d4-e5f6-7890-abcd-ef1234567890.m3u8",
      "viewerCount": 42,
      "startedAt": "2026-08-06T22:00:00Z"
    }
  ],
  "total": 15,
  "page": 1
}
```

**Error Responses:**
- None expected (returns empty array if no live streams)

---

### GET /api/channel/:username

**Purpose:** Get channel info including live status and HLS URL.

**Authentication:** None (viewers don't need to sign in to watch).

**Path params:**
```
username=<streamer's user ID (UUID)>
```

**Success Response 200:**
```json
{
  "streamerId": "uuid",
  "streamerName": "Alice Streamer",
  "streamerAvatarUrl": "https://...",
  "isLive": true,
  "title": "Late night coding",
  "category": "Programming",
  "hlsUrl": "/hls/sk-a1b2c3d4-e5f6-7890-abcd-ef1234567890.m3u8",
  "viewerCount": 42,
  "startedAt": "2026-08-06T22:00:00Z"
}
```

If offline:
```json
{
  "streamerId": "uuid",
  "streamerName": "Alice Streamer",
  "streamerAvatarUrl": "https://...",
  "isLive": false,
  "title": null,
  "category": null
}
```

**Error Responses:**
- 404: User not found

---

## SRS Configuration (reference)

```nginx
# srs.conf
listen 1935;
max_connections 1000;

http_api {
    enabled on;
    listen 8080;
}

http_server {
    enabled on;
    listen 8080;
    dir /data/hls;
}

vhost __defaultVhost__ {
    http_remux {
        enabled on;
        mount [vhost]/[app]/[stream].m3u8;
        hls_m3u8_file /data/hls/[vhost]/[app]/[stream].m3u8;
        hls_ts_file /data/hls/[vhost]/[app]/[stream]-[seq].ts;
        hls_window 60;
    }

    http_hooks {
        enabled on;
        on_publish http://backend:8081/api/srs/callback;
        on_unpublish http://backend:8081/api/srs/callback;
    }
}
```
