import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { ChatPanel } from "./ChatPanel";
import type { ChatMessage } from "@/types";

// ── Helpers ────────────────────────────────────────────────────

/** Create a stub WebSocket class (constructable, no prototype chain issues) */
function createMockWebSocket() {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  function MockWebSocket(this: any) {
    const ws = {
      send: vi.fn(),
      close: vi.fn(),
      onopen: null as (() => void) | null,
      onmessage: null as ((event: MessageEvent) => void) | null,
      onclose: null as ((event: CloseEvent) => void) | null,
      onerror: null as ((event: Event) => void) | null,
    };
    // Define readyState as a configurable writable property to avoid
    // the getter-only issue on the real WebSocket prototype.
    Object.defineProperty(ws, "readyState", {
      value: 0, // CONNECTING
      writable: true,
      configurable: true,
    });
    Object.assign(this, ws);
  }
  return MockWebSocket as unknown as typeof WebSocket;
}

// ── Tests ──────────────────────────────────────────────────────

describe("ChatPanel", () => {
  beforeEach(() => {
    // jsdom doesn't implement scrollIntoView
    Element.prototype.scrollIntoView = vi.fn();

    // Stub WebSocket
    vi.stubGlobal("WebSocket", createMockWebSocket());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders chat header", () => {
    render(<ChatPanel streamId="stream-1" />);
    expect(screen.getByText("💬 CHAT")).toBeInTheDocument();
  });

  it("shows connecting state initially", () => {
    render(<ChatPanel streamId="stream-1" />);
    // The component sets connectionState to "connecting" initially
    expect(screen.getByText("Connecting...")).toBeInTheDocument();
  });

  it("renders VOD replay mode without WebSocket", () => {
    const messages: ChatMessage[] = [
      {
        id: "1",
        userId: "u1",
        userName: "Alice",
        userAvatarUrl: "https://example.com/avatar.jpg",
        message: "Hello!",
        sentAt: "2026-01-01T00:00:00Z",
      },
    ];
    render(
      <ChatPanel
        streamId="stream-1"
        isVodReplay
        initialMessages={messages}
      />
    );
    expect(screen.getByText("Hello!")).toBeInTheDocument();
    expect(screen.getByText("Alice")).toBeInTheDocument();
  });

  it("shows connecting spinner when connected state has no messages", () => {
    // Override WebSocket to fire onopen after construction
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    function MockWs(this: any) {
      const ws = {
        send: vi.fn(),
        close: vi.fn(),
        onopen: null as (() => void) | null,
        onmessage: null as ((event: MessageEvent) => void) | null,
        onclose: null as ((event: CloseEvent) => void) | null,
        onerror: null as ((event: Event) => void) | null,
      };
      Object.defineProperty(ws, "readyState", {
        value: 1, // OPEN
        writable: true,
        configurable: true,
      });
      Object.assign(this, ws);
      // Fire onopen asynchronously
      setTimeout(() => {
        if (this.onopen) this.onopen();
      }, 0);
    }
    vi.stubGlobal("WebSocket", MockWs as unknown as typeof WebSocket);

    render(<ChatPanel streamId="stream-1" isSignedIn />);
    // Initially shows connecting spinner
    expect(
      screen.getByText("Connecting to chat...")
    ).toBeInTheDocument();
  });

  it("does not show chat input in VOD replay mode", () => {
    render(
      <ChatPanel streamId="stream-1" isVodReplay isSignedIn />
    );
    expect(
      screen.queryByPlaceholderText("Type a message...")
    ).not.toBeInTheDocument();
  });

  it("shows connecting badge when not VOD replay", () => {
    render(<ChatPanel streamId="stream-1" isSignedIn />);
    // Connection state is "connecting" — the header shows "Connecting..."
    expect(screen.getByText("Connecting...")).toBeInTheDocument();
  });
});
