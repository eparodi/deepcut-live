import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ChatPanel } from "./ChatPanel";
import type { ChatMessage } from "@/types";

vi.mock("@/lib/api", () => ({
  API_BASE_URL: "http://localhost:8081",
  getMe: vi.fn(),
}));

import { getMe } from "@/lib/api";

const meUser = {
  id: "me",
  name: "Me",
  email: "me@example.com",
  avatarUrl: "https://example.com/me.png",
  streamTitle: null,
  streamCategory: null,
  isLive: false,
};

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

    // Default identity for the optimistic-echo flow.
    vi.mocked(getMe).mockResolvedValue(meUser);
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

  // ── Optimistic send feedback (specs/ui-ux-audit-v1.md D9) ─────────

  it("echoes sent messages optimistically and clears the echo on server reply", async () => {
    let capturedWs: {
      onmessage: ((event: MessageEvent) => void) | null;
    } | null = null;
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
      // Capture the instance (not the internal plain object): the
      // component assigns handlers on the instance it received.
      // eslint-disable-next-line @typescript-eslint/no-this-alias -- test stub needs the instance for firing handlers
      capturedWs = this;
      setTimeout(() => {
        if (this.onopen) this.onopen();
      }, 0);
    }
    vi.stubGlobal("WebSocket", MockWs as unknown as typeof WebSocket);
    vi.mocked(getMe).mockResolvedValue(meUser);

    const user = userEvent.setup();
    render(<ChatPanel streamId="stream-1" isSignedIn />);

    // Wait for the identity fetch to settle before sending.
    await waitFor(() => expect(vi.mocked(getMe)).toHaveBeenCalled());
    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });
    const input = screen.getByPlaceholderText("Type a message...");
    await user.type(input, "Hello!{Enter}");

    // Optimistic echo with a sending state.
    await waitFor(() =>
      expect(screen.getByText("sending…")).toBeInTheDocument()
    );
    expect(screen.getByText("Hello!")).toBeInTheDocument();
    expect(input).toHaveValue("");

    // Server echo clears the pending state.
    expect(capturedWs).not.toBeNull();
    act(() => {
      capturedWs?.onmessage?.({
        data: JSON.stringify({
          type: "message",
          payload: {
            id: "m1",
            userId: "me",
            userName: "Me",
            userAvatarUrl: "https://example.com/me.png",
            message: "Hello!",
            sentAt: new Date().toISOString(),
          },
        }),
      } as MessageEvent);
    });
    await waitFor(() =>
      expect(screen.queryByText("sending…")).not.toBeInTheDocument()
    );
  });

  it("marks in-flight messages as failed with a retry affordance when the socket drops", async () => {
    let capturedWs: {
      onclose: ((event: CloseEvent) => void) | null;
      onmessage: ((event: MessageEvent) => void) | null;
    } | null = null;
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
      // eslint-disable-next-line @typescript-eslint/no-this-alias -- test stub needs the instance for firing handlers
      capturedWs = this;
      setTimeout(() => {
        if (this.onopen) this.onopen();
      }, 0);
    }
    vi.stubGlobal("WebSocket", MockWs as unknown as typeof WebSocket);
    vi.mocked(getMe).mockResolvedValue(meUser);

    const user = userEvent.setup();
    render(<ChatPanel streamId="stream-1" isSignedIn />);

    await waitFor(() => expect(vi.mocked(getMe)).toHaveBeenCalled());
    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });
    const input = screen.getByPlaceholderText("Type a message...");
    await user.type(input, "Hello!{Enter}");

    await waitFor(() =>
      expect(screen.getByText("sending…")).toBeInTheDocument()
    );

    // The socket drops before the server echoes the message.
    act(() => {
      capturedWs?.onclose?.({ code: 1006 } as CloseEvent);
    });

    await waitFor(() =>
      expect(screen.getByText("not delivered")).toBeInTheDocument()
    );
    expect(screen.getByText("Retry")).toBeInTheDocument();
  });
});
