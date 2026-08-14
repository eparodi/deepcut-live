# UI/UX Audit & Enhancement — Frontend (v1)

- **Status:** Draft
- **Owner:** UX/UI Expert (with Frontend Engineer implementing)
- **Created:** 2026-08-13
- **Authorized by:** user instruction "Review the UI/UX … and generate a spec … orchestrate the changes" (2026-08-13)
- **Related spec:** `specs/live-streaming-platform.md`, `specs/browse-vod-discovery.md`
- **Method:** audit of `frontend/src/` against the `ux-ui-expert` skill principles
  (NN/g heuristics, Laws of UX, WCAG 2.2 AA, Apple HIG / Material 3 / Carbon /
  Polaris / Atlassian playbooks). Every finding cites `file:line`.

## Requirements

Audience: viewers (browse, watch, chat) and streamers (dashboard, channel).

- R1: As a viewer using a keyboard, I can see where keyboard focus is at all
  times (WCAG 2.4.7).
- R2: As a viewer with low vision, all body text and controls meet WCAG AA
  contrast (4.5:1 text, 3:1 large/UI).
- R3: As a viewer on a phone, the navigation and grids reflow without
  overflow or broken layouts, and tap targets are at least 24×24 CSS px.
- R4: As a viewer, every data-driven surface has designed loading, empty,
  error, and populated states — no silent failures (NN/g #1, #9).
- R5: As a viewer, every action gives feedback within ~400ms or shows
  progress (Doherty).
- R6: As a streamer, dialogs are keyboard-usable: focus moves in, is trapped,
  Escape closes, and focus returns (NN/g #3).
- R7: As a viewer, copy is consistent: one word per action, one label per
  concept (NN/g #4).

Acceptance criteria (GWT):

- R1 Given chat input is focused When I press Tab Then a visible focus ring
  appears on the next control. And player controls that are faded out are not
  in the Tab order.
- R2 Given the live dashboard When I measure text/background pairs Then all
  body text pairs are ≥4.5:1 (purple links, danger errors, Google-blue
  buttons included).
- R3 Given a 320–640px viewport When I open Browse/Dashboard/Search Then the
  navbar doesn't overflow, grids don't force horizontal page scroll, and
  links/buttons have ≥24px effective height.
- R4 Given the home page loads slowly When the backend is slow Then a
  skeleton layout renders (not a blank page). Given a channel page with an
  unreachable API Then an error+retry state renders (not "Not streaming
  yet").
- R5 Given I send a chat message Then the UI shows the message as pending
  until the server echoes it (or shows a sending state).
- R6 Given the RegenerateKey/ForceEnd dialog is open When I press Escape
  Then the dialog closes and focus returns to the trigger button.
- R7 Given any retry action Then it is labeled identically everywhere
  ("Retry" vs "Try again" — pick one). Given any error page Then it has an
  h1, not an h2.

## Explicit Non-Goals

- No visual redesign, no new design system, no new dependencies.
- No new routes or features (Search link in navbar is a link to the existing
  `/search` route — allowed).
- No backend changes; API contracts untouched.
- No mobile app (there is no `mobile/` directory).
- P2 polish items not listed in the task checklist are backlog, not scope.
- No changes to `frontend/src/types/index.ts` (contract).

## Design — UI Design section

### D1. Color & focus tokens (`frontend/src/app/globals.css`)

Current tokens fail AA for text usage (audit finding: `#9146FF` on
`#0E0E10` ≈4.2:1, on `#18181B` ≈3.8:1; `#DC2626` on surface ≈4.0:1).

- Add text-safe role tokens; keep existing tokens for fills:
  - `--color-primary-text` — light purple (≈#B78CFF range) for text links
    on dark surfaces. Verify ≥4.5:1 against `--color-surface` and raised
    surfaces.
  - `--color-danger-text` — lighter red for error TEXT on dark surfaces
    (fill usages of `--color-danger` stay).
- Replace text usages: `ChannelView.tsx:82-83` (View past streams),
  `LiveStreamCard.tsx:125-128` + `StreamInfo.tsx:68-69` (category pills),
  `AnalyticsCards.tsx:102-103` (Try again), `dashboard/page.tsx:185-186,
  317-321` + `StreamSettingsForm.tsx:102-104` (error text).
- Google-blue sign-in buttons (`Navbar.tsx:87-93`, `ChatInput.tsx:94-101`):
  white on `#4285F4` ≈3.6:1 → darken background (≈`#1A5BBF`-range) to reach
  4.5:1 with white text.
- Focus: `ChatInput.tsx:148` has `outline-none` with no replacement (P0) →
  add `focus:ring-2 focus:ring-[var(--color-primary)]` like
  `search/page.tsx:163`. All interactive elements get a visible
  `focus-visible` treatment (audit: buttons rely on browser default).

### D2. Player control visibility (`VideoPlayer.tsx`)

- Controls fade to `opacity-0` but stay in tab order (`538-546`) → when the
  control bar is hidden, add `invisible` (removes from a11y tree) so
  keyboard users never tab into invisible controls (WCAG 2.4.11).

### D3. Dialog focus management (`RegenerateKeyButton.tsx`, `ForceEndButton.tsx`)

Both dialogs get, without new dependencies:
- initial focus on the dialog (or its first focusable element) on open,
- Escape closes (adds to existing backdrop-click close),
- Tab trap (cycle within dialog),
- focus restored to the trigger on close.
Implementation: a small shared `useDialogFocus` hook under
`frontend/src/lib/` (no new deps).

### D4. Navbar (`Navbar.tsx`)

- Add a "Search" link (route exists at `/search`) — search is currently
  reachable only via empty-state CTAs (audit 8).
- `flex-wrap` so narrow viewports don't overflow; hide "Browse" label on
  `max-sm` if needed (audit 2).
- Give text links padding to reach ≥24px effective target height (audit 4).
- `getMe()` failure must not show "Sign in with Google" to a signed-in user:
  distinguish unknown (show nothing or a subtle retry) from signed-out.

### D5. Home loading state (`frontend/src/app/loading.tsx`)

- New route-level skeleton mirroring the channel skeleton (`channel/[id]/
  loading.tsx` style): navbar-consistent container (`max-w-7xl`) + skeleton
  grid. Home currently renders blank while `getHomeData()` blocks.

### D6. GoLivePreview error state (`GoLivePreview.tsx`)

- Fetch failure currently collapses into "Not streaming yet" (`19-24`,
  `91-121`). Track an error flag; render "Couldn't reach the stream" +
  Retry button instead of the empty state.

### D7. LiveGrid list mode (`LiveGrid.tsx`)

- "List" toggle currently stacks identical full-size cards (`95-99`). Make
  list mode a compact horizontal row: small thumbnail (e.g., 128×72) +
  title + category + viewer count, consistent with the grid card's data.

### D8. Copy & headings consistency

- Retry actions: pick "Retry" everywhere (change `app/error.tsx:36`'s
  "Try again" → "Retry").
- Home page always renders an `h1` (visually hidden if needed; currently
  first heading is `h2` at `LiveGrid.tsx:105`).
- Error pages use `h1` (`app/error.tsx`, `channel/[id]/error.tsx`,
  `dashboard/error.tsx`, `vods/[id]/error.tsx`).
- Dashboard duplicate headings: `StreamKeyDisplay.tsx:46-51` becomes
  "Server & Stream Key"; `StreamSettingsForm.tsx:68-73` stays "Stream
  Settings".
- Error toasts use `role="alert"`; success toasts `role="status"`
  (`Toast.tsx:24-32`), and add a ✓/⚠ icon per variant (no color-only
  meaning, WCAG 1.4.1).

### D9. Chat send feedback (`ChatPanel.tsx`, `ChatInput.tsx`)

- Optimistic echo: append the user's message locally with a "sending"
  visual state until the server echoes it (audit: fire-and-forget with
  zero feedback). On websocket failure surface the message as failed with
  retry.

### D10. Back-link target size

- Bare 14px back links (`ChannelView.tsx:39-44`, `search/page.tsx:138-143`,
  `VodView.tsx:65-70`, `AnalyticsCards.tsx:100-106`) get `py-2 -my-2`
  (or equivalent) for ≥24px effective height.

### D11. Grid step (`page.tsx:72`, `search/page.tsx:236`, `LiveGrid.tsx:98`)

- `grid-cols-2 lg:grid-cols-4` jumps 2→4 → add `md:grid-cols-3`.

### D12. Reduced motion (`globals.css`)

- Wrap `fade-up`, shimmer, `animate-pulse`, `hover:scale` in
  `prefers-reduced-motion` guards (WCAG 2.3.3).

## Task Checklist

1. [ ] D1 color tokens + focus ring (`globals.css`, `ChatInput.tsx`, text
  usages, sign-in buttons) — verify `npx tsc --noEmit` + `npm test`.
2. [ ] D2 player controls hidden-from-tab-order (`VideoPlayer.tsx`).
3. [ ] D3 `useDialogFocus` hook + both dialogs (`lib/`, two components) +
  tests.
4. [ ] D4 Navbar: Search link, wrap, target padding, auth-unknown state +
  test.
5. [ ] D5 home `loading.tsx` + render test.
6. [ ] D6 GoLivePreview error/retry state + test.
7. [ ] D7 LiveGrid compact list mode + test.
8. [ ] D8 copy/headings/toast role & icons + affected tests.
9. [ ] D9 chat optimistic send feedback + test.
10. [ ] D10 back-link target sizes.
11. [ ] D11 `md:grid-cols-3` grid step.
12. [ ] D12 reduced-motion guards (`globals.css`).

Verification per task: `cd frontend && npx tsc --noEmit && npm run lint &&
npm test` (Node per `.nvmrc`).

## Implementation Notes

Implemented on `feat/ux-ui-enhance` (2026-08-13), all 12 tasks, with
`tsc --noEmit`, `eslint`, `vitest` (230 tests), and `next build` green
per task.

- D1: `--color-primary-text` (#B78CFF) and `--color-danger-text`
  (#F87171) text-safe roles; `--color-google-blue` darkened to #1A5BBF
  (≈6.4:1 with white); focus ring restored on the chat input.
- D2: player control bar leaves the a11y tree when hidden (`invisible`,
  visibility participates in the fade transition) + regression tests.
- D3: shared `src/lib/useDialogFocus.ts` hook (initial focus, Tab trap,
  Escape, focus restore, disabled-while-loading) on both dialogs.
- D4: Search link, flex-wrap, ≥24px nav targets, auth-unknown state
  (signed-in users never see "Sign in with Google" on transient
  errors).
- D5: root `app/loading.tsx` skeleton. D6: GoLivePreview error/retry
  state — fetch failure no longer renders "Not streaming yet".
- D7: LiveStreamCard compact list variant (LiveGrid list mode is now a
  real list).
- D8: "Retry" label everywhere, h1 on all error pages + sr-only h1 on
  home, "Server & Stream Key" heading split, error toasts role=alert
  with ⚠️ icon.
- D9: optimistic chat echo (sending/failed states + Retry); ChatInput
  keeps the text when a send fails (onSend now returns success).
- D10-D12: back-link targets, `md:grid-cols-3`, reduced-motion guards.
- Deviations: `useDialogFocus` is a new shared hook, not a new
  dependency (allowed). The player timeline and keyboard-shortcut
  documentation from the audit's "Top priorities" were NOT built
  (out of the spec's task list — backlog). `aria-current` nav marking,
  skip-to-content link, emoji→SVG icon consolidation, and URL-persisted
  view mode are P2 backlog.
- React Compiler note: manual `useMemo` on `combinedMessages` was
  rejected by `react-hooks/preserve-manual-memoization` — dropped the
  manual memoization (the compiler handles it).
