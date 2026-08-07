# US4 — Real-Time Chat

**Status:** Design
**Parent:** `_shared.md`
**Depends on:** US2 (streams must exist), US1 (auth for sending messages)

## User Story

As a viewer, I want to send and read chat messages alongside the stream
so that I can participate in the conversation.

## Acceptance Criteria

- Given I am watching a live stream, When the streamer is live, Then I see a chat panel alongside the video player showing real-time messages
- Given I am signed in with Google, When I type a message and press Enter, Then my message appears in the chat within 1 second
- Given I am NOT signed in, When I try to send a chat message, Then I see "Sign in to chat" linking to Google OAuth
- Given I am watching a past stream (VOD), When I view the video page, Then the chat replay scrolls alongside the video timeline

## Edge Cases

- What happens if the WebSocket connection drops? (Auto-reconnect with exponential backoff, show "Reconnecting..." indicator)
- What happens if a user sends a message very rapidly? (Rate limit: 1 message per 2 seconds per user)
- What happens with 1,000 concurrent chatters? (WebSocket broadcast scales to all viewers of the same stream)
- What happens to chat messages when a stream ends? (Persisted; available in VOD replay)
- What happens if someone sends an empty message? (Client-side validation: reject empty or whitespace-only messages)

---

## API Contract

### WS /ws/chat/:streamId

**Purpose:** WebSocket connection for real-time chat on a specific stream.

**Authentication:** JWT cookie required to send messages; optional to read.

**Protocol:** JSON frames over WebSocket.

**Client → Server messages:**

```json
// Send a chat message
{
  "type": "message",
  "payload": {
    "message": "Hey, great stream!"
  }
}

// Ping (keep-alive)
{
  "type": "ping"
}
```

**Server → Client messages:**

```json
// Chat message from another user
{
  "type": "message",
  "payload": {
    "id": "uuid",
    "userId": "uuid",
    "userName": "Alice Viewer",
    "userAvatarUrl": "https://...",
    "message": "Hey, great stream!",
    "sentAt": "2026-08-06T22:01:30Z"
  }
}

// Error (rate limit, auth, validation)
{
  "type": "error",
  "payload": {
    "code": "rate_limited",
    "message": "Please wait before sending another message"
  }
}

// Pong
{
  "type": "pong"
}
```

**Error codes:**
- `rate_limited` — exceeded 1 msg / 2 seconds
- `unauthorized` — not authenticated (must sign in to send)
- `invalid_message` — empty or whitespace-only message
- `stream_ended` — stream is no longer live

**Connection lifecycle:**
1. Client opens `wss://<host>/ws/chat/<streamId>`
2. Server validates stream exists and is live
3. If stream is offline, server closes with code 4001 "stream offline"
4. Server sends last 50 messages as initial batch
5. Server broadcasts new messages to all connected clients
6. Client sends `ping` every 30s; server responds `pong`
7. Server closes idle connections after 2 minutes of no pings

---

### GET /api/chat/:streamId/messages

**Purpose:** Retrieve chat history for VOD replay.

**Authentication:** None.

**Query params:**
```
?before=<ISO8601 timestamp>&limit=100
```

**Success Response 200:**
```json
{
  "messages": [
    {
      "id": "uuid",
      "userId": "uuid",
      "userName": "Alice Viewer",
      "userAvatarUrl": "https://...",
      "message": "Hey, great stream!",
      "sentAt": "2026-08-06T22:01:30Z"
    }
  ],
  "hasMore": true
}
```

**Error Responses:**
- 404: Stream not found

---

## Implementation Notes

- One WebSocket hub per active stream. Hub starts when first viewer connects, shuts down when last viewer disconnects + stream ends.
- Messages are persisted to PostgreSQL AND broadcast in-memory. Persistence is for VOD replay.
- Rate limiting: per-user, token-bucket with 1 token per 2 seconds, burst of 3. Enforced server-side (never trust client-side rate limiting).
- `nhooyr.io/websocket` handles context cancellation and deadline propagation natively.

---

## UI Design

### Component: ChatPanel (Live)

**Purpose:** Real-time chat alongside the video player.

**Layout:**
```
┌─────────────────────┐
│  💬 CHAT            │
│                     │
│  ┌─────────────────┐│
│  │ Alice: great!   ││  ← auto-scrolls to bottom
│  │ Bob: lol        ││
│  │ Carol: 🔥       ││
│  │ ...             ││
│  └─────────────────┘│
│                     │
│  ┌───────────────┐  │
│  │ Type message...│  │
│  │          [Send]│  │
│  └───────────────┘  │
└─────────────────────┘
```

**States:**

| State | What renders |
|-------|-------------|
| Loading | "Connecting to chat..." + spinner |
| Connected (signed out) | Messages visible, input shows "Sign in to chat" button linking to Google OAuth |
| Connected (signed in) | Messages visible, input active with character count |
| Reconnecting | Yellow banner "Reconnecting..." at top, input disabled |
| Stream ended | Messages visible, input hidden, "Chat closed" label |
| Stream offline (VOD mode) | Messages visible, scrollable timeline synced to video position |

**Message Bubble:**
```
┌─ Alice ─────── 22:01 ─┐
│ great stream!          │
└────────────────────────┘
```

- Avatar (24px round) + name + timestamp
- Consecutive messages from same user: only show avatar/name on first
- Message text: `--text-base`, wrap at 100%

**Accessibility:**
- Message list: `role="log"`, `aria-live="polite"`, `aria-label="Chat messages"`
- Input: `aria-label="Type a chat message"`
- Sign-in prompt: focusable link to Google OAuth

**Component: ChatInput**
- Text input + Send button (primary variant)
- Enter = send, Shift+Enter = newline
- Max 500 chars, counter appears at 400+
- Disabled: not signed in, reconnecting, stream ended, rate-limited (shows "Wait 2s...")
