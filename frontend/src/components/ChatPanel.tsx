"use client";
// Client Component — needs useState for messages and WS state, useEffect for WebSocket lifecycle, useRef for auto-scroll

import { useState, useEffect, useRef, useCallback } from "react";
import { ChatInput } from "./ChatInput";
import { API_BASE_URL } from "@/lib/api";
import { AVATAR_FALLBACK } from "@/lib/fallbacks";
import type { ChatMessage } from "@/types";

type ConnectionState = "connecting" | "connected" | "reconnecting" | "disconnected";

interface ChatPanelProps {
  /** The stream ID to connect to (used in /ws/chat/:streamId) */
  streamId: string;
  /** Whether the current user is signed in (for sending messages) */
  isSignedIn?: boolean;
  /** Whether the stream has ended */
  isStreamEnded?: boolean;
  /** Whether this is a VOD replay (read-only, uses HTTP instead of WS) */
  isVodReplay?: boolean;
  /** Initial messages to display (for VOD replay or initial batch) */
  initialMessages?: ChatMessage[];
}

// Sign-in goes through the Next.js proxy (same-origin), like the Navbar.
// WebSocket must go directly to the backend because Next.js middleware
// doesn't support WS upgrades.
const WS_HOST = process.env.NEXT_PUBLIC_WS_URL || "http://localhost:8081";

/** Convert HTTP URL to WebSocket URL — always points to the backend directly */
function getWsUrl(streamId: string): string {
  const url = new URL(WS_HOST);
  const protocol = url.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${url.host}/ws/chat/${streamId}`;
}

/** Format a timestamp for display */
function formatTime(iso: string): string {
  try {
    const d = new Date(iso);
    return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  } catch {
    return "";
  }
}

/** Calculate exponential backoff delay */
function backoffDelay(attempt: number): number {
  return Math.min(1000 * Math.pow(2, attempt), 30000);
}

export function ChatPanel({
  streamId,
  isSignedIn = false,
  isStreamEnded = false,
  isVodReplay = false,
  initialMessages = [],
}: ChatPanelProps) {
  const [messages, setMessages] = useState<ChatMessage[]>(initialMessages);
  const [connectionState, setConnectionState] = useState<ConnectionState>(
    isVodReplay ? "connected" : "connecting"
  );
  const [isRateLimited, setIsRateLimited] = useState(false);
  const [rateLimitSeconds, setRateLimitSeconds] = useState(0);

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectAttemptRef = useRef(0);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pingTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const rateLimitTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const connectRef = useRef<() => void>(() => {});
  // Set on unmount so a late onclose can't schedule a reconnect (which would
  // leak sockets and ping timers past the component's lifetime).
  const disposedRef = useRef(false);

  const signInUrl = `${API_BASE_URL}/api/auth/google`;

  // Auto-scroll to bottom when messages change
  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [messages, scrollToBottom]);

  // Connect to WebSocket
  const connect = useCallback(() => {
    if (isVodReplay) return;

    const wsUrl = getWsUrl(streamId);
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      setConnectionState("connected");
      reconnectAttemptRef.current = 0;

      // Start ping interval (every 30s)
      if (pingTimerRef.current) clearInterval(pingTimerRef.current);
      pingTimerRef.current = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: "ping" }));
        }
      }, 30000);
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);

        switch (data.type) {
          case "message":
            setMessages((prev) => {
              const msg = data.payload as ChatMessage;
              // Deduplicate by ID as defense-in-depth: overlapping connects
              // (StrictMode, fast reconnects) can deliver the initial batch
              // twice.
              if (prev.some((m) => m.id === msg.id)) return prev;
              return [...prev, msg];
            });
            break;
          case "error":
            if (data.payload?.code === "rate_limited") {
              setIsRateLimited(true);
              // Start countdown
              let seconds = 2;
              setRateLimitSeconds(seconds);
              if (rateLimitTimerRef.current)
                clearInterval(rateLimitTimerRef.current);
              rateLimitTimerRef.current = setInterval(() => {
                seconds--;
                if (seconds <= 0) {
                  setIsRateLimited(false);
                  setRateLimitSeconds(0);
                  if (rateLimitTimerRef.current) {
                    clearInterval(rateLimitTimerRef.current);
                    rateLimitTimerRef.current = null;
                  }
                } else {
                  setRateLimitSeconds(seconds);
                }
              }, 1000);
            }
            break;
          case "pong":
            // Keep-alive acknowledged
            break;
        }
      } catch {
        // Ignore unparseable messages
      }
    };

    ws.onclose = (event) => {
      // Unmounted — never reconnect or touch state.
      if (disposedRef.current) return;

      // Stream offline close code
      if (event.code === 4001) {
        setConnectionState("disconnected");
        return;
      }

      // Auto-reconnect with exponential backoff
      const attempt = reconnectAttemptRef.current;
      if (attempt < 10) {
        setConnectionState("reconnecting");
        const delay = backoffDelay(attempt);
        reconnectTimerRef.current = setTimeout(() => {
          reconnectAttemptRef.current = attempt + 1;
          connectRef.current();
        }, delay);
      } else {
        setConnectionState("disconnected");
      }
    };

    ws.onerror = () => {
      // onclose will fire after this, handling reconnect
    };
  }, [streamId, isVodReplay]);

  // Keep connectRef in sync so reconnect logic always calls latest connect
  useEffect(() => {
    connectRef.current = connect;
  });

  useEffect(() => {
    disposedRef.current = false;
    if (!isVodReplay) {
      connect();
    }

    return () => {
      // Cleanup on unmount: mark disposed FIRST, then detach handlers and
      // close — otherwise the socket's onclose fires after cleanup and
      // schedules a reconnect + new ping interval that outlive the component.
      disposedRef.current = true;
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
      }
      if (pingTimerRef.current) {
        clearInterval(pingTimerRef.current);
      }
      if (rateLimitTimerRef.current) {
        clearInterval(rateLimitTimerRef.current);
      }
      if (wsRef.current) {
        wsRef.current.onclose = null;
        wsRef.current.onerror = null;
        wsRef.current.onmessage = null;
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [connect, isVodReplay]);

  // Send message handler
  const handleSend = useCallback(
    (message: string) => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(
          JSON.stringify({
            type: "message",
            payload: { message },
          })
        );
      }
    },
    []
  );

  // Group consecutive messages from the same user
  const shouldShowHeader = (index: number): boolean => {
    if (index === 0) return true;
    const prev = messages[index - 1];
    const curr = messages[index];
    return prev.userId !== curr.userId;
  };

  return (
    <div
      className="flex flex-col h-full rounded-xl overflow-hidden"
      style={{ backgroundColor: "var(--color-surface-raised)" }}
    >
      {/* Header */}
      <div
        className="px-4 py-3 border-b flex items-center justify-between"
        style={{ borderColor: "var(--color-surface)" }}
      >
        <h3 className="text-sm font-semibold text-[var(--color-text)]">
          💬 CHAT
        </h3>
        {connectionState === "reconnecting" && (
          <span
            className="text-xs font-medium"
            style={{ color: "#EAB308" }}
          >
            Reconnecting...
          </span>
        )}
        {(connectionState === "connecting") && (
          <span className="text-xs text-[var(--color-text-muted)]">
            Connecting...
          </span>
        )}
      </div>

      {/* Message list */}
      <div
        className="flex-1 overflow-y-auto px-3 py-3 space-y-2"
        role="log"
        aria-live="polite"
        aria-label="Chat messages"
      >
        {/* Loading state */}
        {connectionState === "connecting" && messages.length === 0 && (
          <div className="flex items-center justify-center h-full">
            <div className="flex flex-col items-center gap-2">
              <div
                className="w-6 h-6 border-2 border-t-transparent rounded-full animate-spin"
                style={{
                  borderColor: "var(--color-primary)",
                  borderTopColor: "transparent",
                }}
              />
              <p className="text-xs text-[var(--color-text-muted)]">
                Connecting to chat...
              </p>
            </div>
          </div>
        )}

        {/* Empty state */}
        {connectionState === "connected" && messages.length === 0 && (
          <div className="flex items-center justify-center h-full">
            <p className="text-xs text-[var(--color-text-muted)]">
              No messages yet. Be the first to chat!
            </p>
          </div>
        )}

        {/* Disconnected / error state */}
        {connectionState === "disconnected" && (
          <div className="flex items-center justify-center h-full">
            <p className="text-xs text-[var(--color-text-muted)] text-center">
              Chat unavailable
            </p>
          </div>
        )}

        {/* Messages */}
        {messages.map((msg, index) => (
          <div key={msg.id} className="flex items-start gap-2">
            {/* Avatar — only show for first in group */}
            <div className="w-6 h-6 shrink-0">
              {shouldShowHeader(index) ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={msg.userAvatarUrl}
                  alt={msg.userName}
                  className="w-6 h-6 rounded-full"
                  referrerPolicy="no-referrer"
                  onError={(e) => {
                    const target = e.target as HTMLImageElement;
                    target.onerror = null;
                    target.src = AVATAR_FALLBACK;
                  }}
                />
              ) : (
                <div className="w-6" />
              )}
            </div>

            <div className="flex-1 min-w-0">
              {/* Name + timestamp — only show for first in group */}
              {shouldShowHeader(index) && (
                <div className="flex items-center gap-2 mb-0.5">
                  <span
                    className="text-xs font-semibold truncate"
                    style={{ color: "var(--color-primary-text)" }}
                  >
                    {msg.userName}
                  </span>
                  <span className="text-xs text-[var(--color-text-muted)] shrink-0">
                    {formatTime(msg.sentAt)}
                  </span>
                </div>
              )}

              {/* Message text */}
              <p
                className="text-sm text-[var(--color-text)] break-words"
                style={{ fontSize: "var(--text-sm)" }}
              >
                {msg.message}
              </p>
            </div>
          </div>
        ))}

        {/* Auto-scroll anchor */}
        <div ref={messagesEndRef} />
      </div>

      {/* Chat input — not shown for VOD replay or ended streams */}
      {!isVodReplay && (
        <ChatInput
          isSignedIn={isSignedIn}
          isReconnecting={connectionState === "reconnecting"}
          isStreamEnded={isStreamEnded}
          isRateLimited={isRateLimited}
          rateLimitSeconds={rateLimitSeconds}
          signInUrl={signInUrl}
          onSend={handleSend}
        />
      )}
    </div>
  );
}
