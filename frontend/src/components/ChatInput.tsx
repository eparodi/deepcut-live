"use client";
// Client Component — needs useState for input and event handlers

import { useState, useCallback, type KeyboardEvent } from "react";

interface ChatInputProps {
  /** Whether the user is signed in */
  isSignedIn: boolean;
  /** Whether the WebSocket is reconnecting */
  isReconnecting: boolean;
  /** Whether the stream has ended */
  isStreamEnded: boolean;
  /** Whether the user is rate-limited (show countdown) */
  isRateLimited: boolean;
  /** Seconds remaining in rate limit */
  rateLimitSeconds?: number;
  /** Sign-in URL for Google OAuth */
  signInUrl: string;
  /** Called when user sends a message */
  onSend: (message: string) => void;
}

export function ChatInput({
  isSignedIn,
  isReconnecting,
  isStreamEnded,
  isRateLimited,
  rateLimitSeconds = 0,
  signInUrl,
  onSend,
}: ChatInputProps) {
  const [message, setMessage] = useState("");

  const MAX_CHARS = 500;
  const SHOW_COUNTER_AT = 400;

  // Derived, not state — always in sync with the input value.
  const charCount = message.length;

  const handleSend = useCallback(() => {
    const trimmed = message.trim();
    if (!trimmed || trimmed.length === 0) return;
    if (trimmed.length > MAX_CHARS) return;
    if (!isSignedIn) return;
    if (isReconnecting) return;
    if (isStreamEnded) return;
    if (isRateLimited) return;

    onSend(trimmed);
    setMessage("");
  }, [
    message,
    isSignedIn,
    isReconnecting,
    isStreamEnded,
    isRateLimited,
    onSend,
  ]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        handleSend();
      }
    },
    [handleSend]
  );

  const handleChange = useCallback((value: string) => {
    if (value.length <= MAX_CHARS) {
      setMessage(value);
    }
  }, []);

  const isDisabled =
    !isSignedIn || isReconnecting || isStreamEnded || isRateLimited;

  // Stream ended
  if (isStreamEnded) {
    return (
      <div className="p-3 border-t" style={{ borderColor: "var(--color-surface)" }}>
        <p className="text-xs text-[var(--color-text-muted)] text-center">
          Chat closed — stream ended
        </p>
      </div>
    );
  }

  // Not signed in
  if (!isSignedIn) {
    return (
      <div className="p-3 border-t" style={{ borderColor: "var(--color-surface)" }}>
        <a
          href={signInUrl}
          className="flex items-center justify-center gap-2 w-full rounded-lg px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:opacity-90"
          style={{ backgroundColor: "var(--color-google-blue)" }}
        >
          Sign in to chat
        </a>
      </div>
    );
  }

  // Reconnecting
  if (isReconnecting) {
    return (
      <div className="p-3 border-t" style={{ borderColor: "var(--color-surface)" }}>
        <div
          className="rounded-md px-3 py-2 text-xs text-center font-medium"
          style={{
            backgroundColor: "rgba(234, 179, 8, 0.15)",
            color: "#EAB308",
          }}
        >
          Reconnecting...
        </div>
      </div>
    );
  }

  return (
    <div className="p-3 border-t" style={{ borderColor: "var(--color-surface)" }}>
      {/* Rate limit warning */}
      {isRateLimited && (
        <div
          className="rounded-md px-3 py-1.5 text-xs text-center font-medium mb-2"
          style={{
            backgroundColor: "rgba(234, 179, 8, 0.15)",
            color: "#EAB308",
          }}
        >
          Wait {rateLimitSeconds}s before sending again
        </div>
      )}

      <div className="flex items-center gap-2">
        <div className="flex-1 relative">
          <input
            type="text"
            value={message}
            onChange={(e) => handleChange(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Type a message..."
            disabled={isDisabled}
            maxLength={MAX_CHARS}
            aria-label="Type a chat message"
            className="w-full rounded-lg px-3 py-2 text-sm outline-none transition-colors disabled:opacity-50"
            style={{
              backgroundColor: "var(--color-surface)",
              color: "var(--color-text)",
              border: "1px solid transparent",
            }}
          />
          {/* Character counter (shown when approaching limit) */}
          {charCount >= SHOW_COUNTER_AT && (
            <span
              className="absolute right-2 top-1/2 -translate-y-1/2 text-xs font-mono"
              style={{
                color:
                  charCount >= MAX_CHARS
                    ? "var(--color-danger)"
                    : "var(--color-text-muted)",
              }}
            >
              {charCount}/{MAX_CHARS}
            </span>
          )}
        </div>

        <button
          type="button"
          onClick={handleSend}
          disabled={isDisabled || message.trim().length === 0}
          className="shrink-0 rounded-lg px-4 py-2 text-sm font-semibold text-white transition-colors hover:opacity-90 disabled:opacity-40 disabled:cursor-not-allowed"
          style={{ backgroundColor: "var(--color-primary)" }}
          aria-label="Send message"
        >
          Send
        </button>
      </div>
    </div>
  );
}
