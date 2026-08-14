# Agent Assistant Panel — Channel Page (v1 Pilot)

- **Status:** Draft
- **Owner:** PM (requirements) — Architect + UX Designer (design)
- **Created:** 2026-08-13
- **Related:** `specs/live-streaming-platform.md`, `specs/browse-vod-discovery.md`, `specs/component-catalog-v1.md`, `.agents/skills/ux-designer/references/a2ui.md`

## Requirements

The pilot delivers ONE agent-generated surface: an assistant panel on
the channel page that answers questions about the stream and channel
using ONLY data the platform already serves. It is the vehicle for the
A2UI catalog-first pattern (see Design D4) — small, read-only, and
nowhere near money or stream-critical controls.

### User stories

- US1: As a viewer on a channel page, I can open an assistant panel and
  ask questions about the stream ("How many people are watching?", "What
  does this channel usually stream?", "When did this stream start?"),
  and get an answer based on the platform's real data.
- US2: As a viewer, when the assistant wants to point me somewhere
  useful (a past stream, the channel's live status), it renders a
  familiar catalog component (status line, VOD suggestion) — never
  arbitrary markup or raw styling.
- US3: As a viewer, I always see what the assistant is doing: thinking
  (skeleton), answering, failed (with retry) — no silent failures.
- US4: As an operator, the assistant never fabricates platform data and
  never costs more than the configured budget per request.

### Acceptance criteria (GWT)

- US1 Given an open assistant panel on `/channel/[id]` When I send
  "how many viewers?" Then within the response-time budget I get a
  reply whose numbers match `GET /api/channels/:id`'s current data.
  And if the channel data is unavailable, the reply says so instead of
  inventing a number.
- US2 Given an assistant reply that recommends a past stream Then the
  panel renders a `vod_suggestion` component linking to the real
  `/vods/[id]`. And no reply may emit any component type outside the
  pilot allowlist (channel_status, vod_suggestion).
- US3 Given the LLM is slow or down When I send a message Then the
  panel shows a thinking state and, on failure, a plain-language error
  with Retry — never a blank panel.
- US4 Given a malformed or schema-violating LLM response Then the
  backend falls back to a text-only reply (validated), and no
  unvalidated data reaches the renderer.
- US4 Given N requests exceed the per-user rate limit Then further
  requests get 429 with a plain-language message in the panel.

## Explicit Non-Goals

- NO A2UI protocol runtime dependency (no `@a2ui/*`, no CopilotKit
  runtime) — the pilot implements the CATALOG PATTERN with an in-house
  renderer (Design D4 rationale).
- No agent write actions: the assistant is read-only. It cannot modify
  stream settings, moderate, follow, or perform any mutation.
- No general-purpose chat or personality — this is a
  stream-context Q&A assistant, not a companion bot.
- No conversation memory/persistence in the pilot (stateless
  per-message; history is client-side only).
- No token streaming (SSE) in the pilot — single response with a
  thinking state.
- No assistant on VOD/dashboard pages in this spec (pilot = channel
  page only).
- No backend data-model changes; the assistant reads existing
  endpoints/services.

## Design

### D1. Architecture (Architect)

New hexagonal module `backend/internal/modules/assistant/` mirroring
the existing module layout:

```
assistant/
├── domain/            # AssistantComponent types, allowlist registry, reply model
├── application/       # usecase: assemble context → build prompt → call provider → validate → map
├── adapter/http/      # POST /api/assistant/query handler (chi, existing JWT auth)
└── adapter/llm/       # OpenAI-compatible provider client (config-driven, see D3)
```

Flow: handler (auth + validation + rate limit) → usecase → context
assembly from the existing streams/channels/vods services (same data
the public endpoints serve — the LLM sees NO secrets, NO stream keys,
NO viewer PII beyond public counts/names) → prompt build → LLM call
(timeout, cost cap) → strict JSON validation → component allowlist
check + per-type data schema validation → response. Any validation
failure degrades to the text-only reply (never 500 unless the
provider itself is unreachable — then D2's error contract applies).

The catalog registry lives in `domain/`: the ONLY place component
types are declared. Adding a type is a deliberate domain change + UI
change + catalog-spec update, not something a prompt can do.

### D2. API contract (Architect)

`POST /api/assistant/query` — auth: existing JWT (cookie), same as
dashboard endpoints.

Request:
```json
{
  "channelId": "string (required)",
  "message": "string (required, 1-500 chars)"
}
```

Success (200):
```json
{
  "reply": "plain-text answer, 1-4 sentences, no markdown",
  "components": [
    {
      "type": "channel_status | vod_suggestion",
      "data": { "...": "per-type schema below" }
    }
  ]
}
```

Per-type data schemas (exact — the frontend types are the contract):
- `channel_status`: `{ "isLive": boolean, "viewerCount": number, "startedAt": string | null }`
- `vod_suggestion`: `{ "id": string, "title": string | null, "durationSeconds": number | null }`

Errors: `400` invalid body, `401` unauthenticated, `404` unknown
channel, `429` rate limited, `502` provider unreachable (frontend
shows Retry), `504` provider timeout. `reply` is always plain text —
the backend strips/escapes any markup attempt before it ships.

Frontend contract additions in `frontend/src/types/index.ts` (must
match byte-for-byte, per AGENTS.md 2.1):
```ts
export interface AssistantReply {
  reply: string;
  components: AssistantComponent[];
}
export type AssistantComponent =
  | { type: "channel_status"; data: { isLive: boolean; viewerCount: number; startedAt: string | null } }
  | { type: "vod_suggestion"; data: { id: string; title: string | null; durationSeconds: number | null } };
```

### D3. LLM provider + safety rules (Architect)

- **Provider:** OpenAI-compatible HTTP client (raw `net/http`, no new
  dependency), endpoint + model + key from env vars:
  `ASSISTANT_LLM_URL`, `ASSISTANT_LLM_MODEL`, `ASSISTANT_LLM_API_KEY`
  (dev default logs a `slog.Warn` at startup — AGENTS.md 4.2).
  DeepSeek's OpenAI-compatible API is the reference target (same
  pattern the bot repo uses).
- **Budget:** per-request caps from env (`ASSISTANT_MAX_OUTPUT_TOKENS`
  default 800, request timeout default 15s) — the LLM NEVER receives
  unbounded context.
- **System prompt (pinned in Go, config-interpolated only):** the
  assistant may answer only from the provided context block; must
  refuse anything else; must never fabricate numbers; must not output
  markdown, links, or component types outside the allowlist; responses
  are strict JSON `{reply, components[]}`.
- **Validation ladder (malformed → fallback, never panic):** strict
  JSON decode (unknown fields rejected), allowlist check, per-type
  schema check. Any failure → `{reply: "<recovered text or apology>",
  components: []}`. Matches the bot repo's malformed→HOLD discipline.
- **Rate limit:** per-user, default 10 requests/minute (env-tunable);
  429 response carries the plain-language message.

### D4. UI design (UX Designer) — the pilot surface

**Placement:** on `/channel/[id]`, the existing chat column becomes a
tabbed stack: `Chat | Assistant`. Chat stays primary/default (chat is
real-time social, assistant is on-demand). On mobile the panel stacks
below the player like chat does today.

**Catalog entry (new, per `specs/component-catalog-v1.md`):**

```markdown
### Catalog: assistant-panel

**Purpose / Behavior:** stream-context Q&A. Sends text to
POST /api/assistant/query; renders the validated reply + allowlisted
components.

**Props:**
| Prop | Type | Required | Default | Source |
|------|------|----------|---------|--------|
| channelId | string | yes | — | route param |
| isSignedIn | boolean | yes | — | page auth |

**States (data-bound):**
| State | Trigger | Renders |
|-------|---------|---------|
| Idle/empty | no messages | 1-line explainer + examples |
| Thinking | request in flight | 3-line skeleton |
| Populated | reply received | reply text + components |
| Failed | 4xx/5xx/timeout | plain-language error + Retry |
| Rate limited | 429 | "slow down" copy + countdown hint |

**Action semantics:** send → POST (stateless); Retry resends the last
message; components are NAVIGATION ONLY (vod_suggestion → /vods/[id]);
no write actions exist in the pilot.

**Responsive:** full column width; input mirrors chat-input patterns
(Enter sends, 500-char cap).

**Accessibility:** role=log + aria-live=polite on the thread; thinking
state exposed via aria-busy; errors role=alert; component cards keep
their own catalog a11y.

**Tokens:** surface, surface-raised, text, text-muted,
primary/primary-text. No new tokens.
```

**Component rendering (the A2UI pattern, in-house):** a tiny renderer
maps validated `components[]` → existing React components: a
`channel_status` renders as a status line (reusing StreamInfo's
live/offline text pattern), `vod_suggestion` as a compact row linking
to `/vods/[id]` (reusing VodCard's data + a row layout). Unknown types
are impossible by contract (backend validates), and the renderer
throws on anything unexpected in dev.

**Decision record — why not the A2UI runtime:** the protocol is early
preview with no official React renderer (`references/a2ui.md` §7), and
a new dependency violates AGENTS.md 1.1 without approval. The
in-house mapper is ~100 lines, exercises the exact catalog + data
binding + validation concepts, and can be swapped for a standard
renderer later without changing the catalog entries.

### D5. Task checklist (PM)

1. [ ] Backend: `assistant` module skeleton + domain types/allowlist +
   table-driven validation tests (happy + every error path).
2. [ ] Backend: OpenAI-compatible LLM adapter (raw `net/http`,
   config-driven, pinned fake-provider tests; one real captured
   response fixture per AGENTS.md external-payload rule).
3. [ ] Backend: context assembly + prompt builder (pinned prompt test;
   test asserts secrets never appear in prompts).
4. [ ] Backend: `POST /api/assistant/query` handler — auth, validation,
   rate limit, error contract; integration test via `httptest` +
   fake provider (no network).
5. [ ] Frontend: `AssistantReply` types in `src/types/index.ts`
   (exact contract) + API client function in `lib/api.ts`.
6. [ ] Frontend: `AssistantPanel` component + the in-house catalog
   renderer; render tests for every state (idle, thinking, populated,
   failed, rate-limited) + a component-render test per allowlist type.
7. [ ] Frontend: channel page tabs (Chat | Assistant), mobile stacking;
   layout test.
8. [ ] Update `specs/component-catalog-v1.md` with the
   `assistant-panel` entry (maintenance rule) and pin the two
   allowlist component types.
9. [ ] QA pass: run the spec's GWT acceptance criteria against a local
   stack with a fake provider; verify no secrets in prompts/logs.

### D6. Open questions for the Review Gate

1. **Model/provider default:** DeepSeek (OpenAI-compatible) as the
   reference, config-driven — acceptable, or must it be something else?
2. **Signed-in only?** Proposed: assistant requires sign-in (rate
   limiting + cost control). Confirm or allow anonymous with a lower
   limit.
3. **LLM failure UX:** proposed graceful text fallback + Retry (502
   only when provider unreachable). Confirm.
4. **Rate limit:** 10 req/min default — confirm the number.
5. **Token budget:** 800 output tokens / 15s timeout defaults —
   confirm.
6. **Chat | Assistant tabs vs side-by-side?** Proposed tabs with Chat
   default. Confirm.

## Implementation Notes

(Filled in during implementation — deviations go here.)
