# Retro: HLS Low-Latency + Custom Player Controls — 2026-08-10

**Session log:** `specs/memories/2026-08-10-session-log.md` (corrections #16-18)
**PR:** [#20](https://github.com/eparodi/deepcut-live/pull/20)
**Spec:** `specs/hls-low-latency.md`

## Corrections Traced to Missing Rules

### Correction 16: Theater mode `!w-screen` forced full viewport

**What happened:** The `VideoPlayer` component used `!w-screen` to implement
theater mode, which forced the player to 100vw regardless of page layout.
The correct behavior is to expand within the content area and restack the
grid (chat below video).

**Root cause:** No rule about component CSS boundaries. Components should
not override page-level layout (viewport width, body scroll, grid columns).
Layout changes that affect siblings or the page structure should be
communicated upward via callbacks/props.

**Missing rule → Added to `nextjs` skill:**

```markdown
### DO NOT — Use viewport units for component-internal layout changes

Components should not use `vw`, `vh`, `w-screen`, or `h-screen` to
implement features that affect page layout (theater mode, expanded view).
These units ignore the parent container and page structure. Instead:

- Lift the state to the parent component via a callback prop
- Let the parent decide how to rearrange the layout
- Use `%`, `flex`, or `max-w-*` within the component to fill available space

// ❌ Wrong — component forces viewport width, breaks all layouts
isExpanded ? "!w-screen" : ""

// ✅ Right — component expands within parent, parent controls layout
isExpanded ? "!max-w-full" : ""
```

### Correction 17: Control buttons triggered fullscreen on double-click

**What happened:** Clicking theater/volume/play buttons twice rapidly
triggered the parent container's `onDoubleClick={toggleFullscreen}` handler.
Only `onClick` propagation was stopped; `onDoubleClick` bubbled up.

**Root cause:** No rule about stopping both click event types on interactive
controls inside clickable containers.

**Missing rule → Added to `nextjs` skill:**

```markdown
### DO — Stop both onClick and onDoubleClick on controls inside clickable parents

When a container has `onClick` or `onDoubleClick` handlers, every interactive
child (button, input, slider) must stop propagation for BOTH event types:

// ✅ Right
<button
  onClick={(e) => e.stopPropagation()}
  onDoubleClick={(e) => e.stopPropagation()}
>
```

### Correction 18: Unused `fireEvent` import

**What happened:** Removed `fireEvent` usage from tests but left the import.
Already caught by `eslint --max-warnings 0`. No rule change needed.

## Review Findings → Rules to Consider

### Warning: Direct `document.body.style` manipulation

The theater mode effect modifies `document.body.style.backgroundColor`
directly. This could conflict with other components (toasts, modals).

**Proposed rule (not yet added):**

```markdown
### DO — Use CSS class toggling for body-level styles

Prefer `document.body.classList.toggle("theater-mode")` over direct
`document.body.style.` assignments. Define the styles in `globals.css`.
This avoids conflicts with other components and makes cleanup trivial.
```

## Skills Updated

- **`nextjs` skill:** Added "DO NOT use viewport units for component-internal
  layout changes" rule (from correction 16)
- **`nextjs` skill:** Added "Stop both onClick and onDoubleClick on controls
  inside clickable parents" rule (from correction 17)

## What Went Well

- Spec-driven workflow caught scope decisions early (WebRTC vs HLS, industry-standard LL-HLS params)
- Theater mode fix was identified and resolved in one iteration
- All 152 tests passed throughout — no regressions introduced
- PR review found only 2 minor warnings, no critical issues
- The `data/` gitignore issue (SRS config was excluded) was handled with `git add -f` — consider adding `!data/srs.conf` to `.gitignore`
