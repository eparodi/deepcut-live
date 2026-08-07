# US1 — Streamer Onboarding

**Status:** Design
**Parent:** `_shared.md`
**Depends on:** nothing (entry point)
**Blocks:** US2, US6

## User Story

As a streamer, I want to sign up with my Google account and get a stream
key so that I can start broadcasting from OBS without friction.

## Acceptance Criteria

- Given I am on the landing page, When I click "Start Streaming", Then I see a "Sign in with Google" button
- Given I click "Sign in with Google", When I authenticate with my Google account, Then I am redirected to my dashboard
- Given I am a newly authenticated streamer, When I land on my dashboard, Then I see my unique stream key prominently displayed with a copy button
- Given I am on my dashboard, When I click "Regenerate Stream Key", Then a confirmation dialog appears, and on confirm my old key is revoked and a new one is generated
- Given I have my stream key, When I configure OBS with `rtmp://<server>/live` and my key, Then OBS connects and I see a green connection indicator

## Edge Cases

- What happens if the same Google account signs up twice? (Re-authenticate, return existing account)
- What happens if the OAuth flow fails mid-way? (Redirect to landing page with error message)
- What happens if the user revokes Google access later? (Account remains but can't sign in until re-authorized)

---

## API Contract

### GET /api/auth/google

**Purpose:** Redirect to Google OAuth consent screen.

**Authentication:** None.

**Response 302:** Redirects to `https://accounts.google.com/o/oauth2/auth?...`

**Query params auto-generated:** `client_id`, `redirect_uri`, `response_type=code`, `scope=openid+profile+email`, `state=<csrf-token>`

---

### GET /api/auth/google/callback

**Purpose:** Handle OAuth callback, create/authenticate user, set session cookie.

**Authentication:** None (called by Google redirect).

**Query params:**
```
code=<authorization-code>&state=<csrf-token>
```

**Success Response 302:** Redirects to `/dashboard`.

**Sets cookie:** `token=<JWT>; HttpOnly; Secure; SameSite=Lax; Path=/; Max-Age=604800`

**Error Responses:**
- 400: Invalid or expired state parameter (CSRF mismatch)
- 400: Missing code parameter
- 502: Google token exchange failed

**Side effects:**
- If first-time user: creates `users` row, generates stream key, returns key in first dashboard load
- If returning user: updates `updated_at`, signs in

---

### GET /api/me

**Purpose:** Return the authenticated user's profile and stream key.

**Authentication:** Bearer cookie (JWT).

**Success Response 200:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Alice Streamer",
  "email": "alice@gmail.com",
  "avatarUrl": "https://lh3.googleusercontent.com/...",
  "streamKey": "sk-a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "streamTitle": null,
  "streamCategory": null,
  "isLive": false
}
```

**Error Responses:**
- 401: Not authenticated (no or invalid cookie)

---

### POST /api/me/stream-key/regenerate

**Purpose:** Revoke the current stream key and generate a new one.

**Authentication:** Bearer cookie (JWT).

**Request body:** Empty (no parameters).

**Success Response 200:**
```json
{
  "streamKey": "sk-new-key-uuid-here"
}
```

**Error Responses:**
- 401: Not authenticated

**Side effects:**
- Old key immediately invalidated (new bcrypt hash stored)
- If user is currently live, existing OBS connection stays active (key is checked only on connect)

---

---

## UI Design

### Screen: Landing Page (unauthenticated)

**Purpose:** Entry point for new streamers. No streams shown — just a CTA.

**Layout:**
```
┌─────────────────────────────────────────┐
│  [Logo]                    [Sign In]    │
├─────────────────────────────────────────┤
│                                         │
│         Stream What You Believe          │
│      No censorship. No filters.          │
│                                         │
│      [Start Streaming with Google]       │
│                                         │
│  🔴 42 live now  │  📼 1,337 past streams│
└─────────────────────────────────────────┘
```

**Components:**
- `HeroSection` — title, subtitle, CTA button
- `GoogleSignInButton` — "Sign in with Google" (Google brand guidelines colors: #4285F4)
- `LiveStats` — counts of live streams and past streams (pulled from `/api/streams/live` + `/api/search?q=` totals)

### Screen: Dashboard

**Purpose:** Streamer's control center after sign-in.

**Layout:**
```
┌──────────────────────────────────────────┐
│  [Logo]  Dashboard  [Avatar ▼]           │
├──────────────────────────────────────────┤
│                                          │
│  Stream Key                              │
│  ┌────────────────────────────────────┐  │
│  │ sk-a1b2c3d4-e5f6-7890-abcd-ef...  │  │
│  │                          [📋 Copy] │  │
│  └────────────────────────────────────┘  │
│  [🔄 Regenerate Key]                     │
│                                          │
│  Stream Settings                         │
│  ┌─ Title ──────────────────────────┐   │
│  │ Late night coding session         │   │
│  └───────────────────────────────────┘   │
│  ┌─ Category ────────────────────── ─┐   │
│  │ Programming                       │   │
│  └───────────────────────────────────┘   │
│  [💾 Save]                               │
│                                          │
│  Analytics (This Week)                   │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐   │
│  │ 12h  │ │ 142  │ │ 1.2k │ │  5   │   │
│  │stream│ │ peak │ │unique│ │streams│  │
│  └──────┘ └──────┘ └──────┘ └──────┘   │
│                                          │
│  [🔴 Force End Stream]  (if live)        │
└──────────────────────────────────────────┘
```

**Components:**
- `StreamKeyDisplay` — key in monospace, masked by default, copy button
- `RegenerateKeyButton` — secondary button with confirmation dialog
- `StreamSettingsForm` — title input + category input + save button
- `AnalyticsCards` — 4 cards in a row: stream time, peak viewers, unique viewers, stream count
- `ForceEndButton` — only visible when `isLive === true`, danger variant, dual-confirm

**States:**
| Component | Loading | Empty | Error | Populated |
|-----------|---------|-------|-------|-----------|
| StreamKeyDisplay | Skeleton rectangle | N/A (always present after signup) | "Could not load key" + retry | Masked key + copy button |
| StreamSettingsForm | Disabled inputs + spinner | Placeholder text in inputs | Red border + error message | Pre-filled with current values |
| AnalyticsCards | 4 skeleton cards | "Start streaming to see analytics" | "Analytics unavailable" | Numbers with labels |

### Component: RegenerateKeyDialog

**Purpose:** Confirmation before revoking stream key.

**Copy:**
- Title: "Regenerate Stream Key?"
- Body: "Your current stream key will stop working immediately. If OBS is streaming, it will disconnect on the next restart. Update OBS with the new key."
- Cancel: "Keep Current Key"
- Confirm: "Regenerate" (danger variant)

---

## Design Tokens

```css
/* Colors */
--color-primary: #9146FF;        /* Purple — streaming brand */
--color-primary-hover: #772CE8;
--color-danger: #DC2626;
--color-surface: #0E0E10;        /* Dark background */
--color-surface-raised: #18181B;
--color-text: #EFEFF1;
--color-text-muted: #ADADB8;
--color-google-blue: #4285F4;

/* Typography */
--font-family: "Inter", system-ui, sans-serif;
--text-xs: 12px / 1.5;
--text-sm: 14px / 1.5;
--text-base: 16px / 1.6;
--text-lg: 18px / 1.5;
--text-xl: 24px / 1.3;
--text-2xl: 32px / 1.2;

/* Spacing */
4px grid: --space-1 (4px) through --space-16 (64px)
/* Breakpoints */
Mobile: < 640px
Desktop: >= 1024px

/* Copied stream key toast: success variant, auto-dismiss 3 seconds */
```

## Implementation Notes

- Stream key format: `sk-<uuid>` for human readability. Only the UUID portion is validated against the hash.
- CSRF state token for OAuth: generate random 32-byte hex, store in short-lived cookie, verify on callback.
- JWT payload: `{ sub: user_id, email: user_email, exp: now+7d }`. Signed with HMAC-SHA256. Secret from env var.
