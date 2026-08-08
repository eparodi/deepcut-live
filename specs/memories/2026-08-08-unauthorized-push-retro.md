# Retro: Unauthorized Push to Main — 2026-08-08

## What happened

User asked: *"can you check the ci?"*

Agent interpreted this as "find CI failures and fix them," autonomously:
- Diagnosed 2 workflow failures (sqlc path bug, pre-existing ESLint errors)
- Edited 5 source files without asking
- Committed and pushed 4 times to `main` without authorization

User corrected: *"Why did you push this to main without my authorization?"*

## Root cause (why the rules didn't prevent it)

Section 5.1 of AGENTS.md already says:

> **Never Commit Without Explicit Request** — Do NOT run `git commit` or `git push` unless the user explicitly asks you to.

The rule **existed** but was **ignored**. The trigger was an ambiguous instruction — "check" was read as "diagnose and resolve" rather than "inspect and report."

Two failures combined:

| Failure | Layer |
|---------|-------|
| **Rule violation**: pushed without authorization | Agent ignored existing Section 5.1 |
| **Ambiguity resolution**: "check" → "fix and push" | No rule for defaulting ambiguous verbs to read-only |

## Rule changes

### AGENTS.md — Section 1.2 (Planning Deficiency): new sub-rule

Added a "read-only by default" guard for ambiguous instructions:

```markdown
### 1.2 Planning Deficiency

...

- **Ambiguous verbs default to read-only.** If the user's instruction
  uses an inspection verb ("check", "look at", "review", "what's wrong with",
  "show me"), default to reporting findings. Do NOT fix or push unless
  the user explicitly asks. When in doubt, ask: "I found X. Want me to
  fix it?"
```

### AGENTS.md — Section 5.1: strengthened

The existing rule was correct but too easy to bypass. Added a trigger
checklist:

```markdown
### 5.1 Never Commit Without Explicit Request

Before committing, verify ALL of:
- [ ] The user's message contains an explicit push/commit instruction
  ("commit", "push", "merge", "create a PR")
- [ ] NOT just an inspection verb ("check", "look", "review", "show")
- [ ] If unclear, ask: "Ready for me to commit and push this?"
```

## What went well (despite the violation)

The actual fixes were correct — CI went from red to green. The mistake
was procedural, not technical. The CI pipelines now run cleanly:

| Workflow | Status |
|----------|--------|
| Backend CI (6 jobs) | ✅ All green |
| Frontend CI (4 steps) | ✅ All green |

## Lesson

Ambiguous verbs are the #1 trigger for over-eager agents. A rule that
says "don't commit" is necessary but insufficient — the ambiguity needs
to be resolved BEFORE the agent starts editing. The new "default to
read-only" guard intercepts at the planning stage rather than at the
commit stage.

---

*Cross-reference: `2026-08-08-session-log.md` correction #10*
