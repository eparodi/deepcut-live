import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";

// ── Mocks ──────────────────────────────────────────────────────

vi.mock("@/components/LiveGrid", () => ({
  LiveGrid: (props: { streams: unknown[]; total: number }) => (
    <div data-testid="live-grid">
      LiveGrid: {props.total} stream{props.total !== 1 ? "s" : ""}
    </div>
  ),
}));

import HomePage from "./page";

// ── Helpers ────────────────────────────────────────────────────

function mockFetch(response: Response | Error) {
  if (response instanceof Error) {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(response));
  } else {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(response));
  }
}

// ── Tests ──────────────────────────────────────────────────────

describe("HomePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.unstubAllGlobals();
  });

  it("renders the LiveGrid with streams", async () => {
    const streams = [
      {
        userId: "user-1",
        streamerName: "TestStreamer",
        streamerAvatarUrl: "https://example.com/avatar.jpg",
        streamId: "stream-1",
        title: "Awesome Stream",
        category: "Gaming",
        viewerCount: 42,
        thumbnailUrl: null,
        startedAt: "2026-01-01T00:00:00Z",
      },
    ];
    mockFetch(
      new Response(JSON.stringify({ streams, total: 1 }), { status: 200 })
    );

    const jsx = await HomePage();
    render(jsx);

    expect(screen.getByTestId("live-grid")).toBeInTheDocument();
    expect(screen.getByText(/1 stream/)).toBeInTheDocument();
  });

  it("renders empty LiveGrid when no streams", async () => {
    mockFetch(
      new Response(JSON.stringify({ streams: [], total: 0 }), { status: 200 })
    );

    const jsx = await HomePage();
    render(jsx);

    expect(screen.getByTestId("live-grid")).toBeInTheDocument();
    expect(screen.getByText(/0 streams/)).toBeInTheDocument();
  });

  it("handles fetch errors gracefully", async () => {
    mockFetch(new Error("Network down"));

    const jsx = await HomePage();
    render(jsx);

    // Should still render LiveGrid with empty state
    expect(screen.getByTestId("live-grid")).toBeInTheDocument();
  });

  it("shows hero text when no live streams", async () => {
    mockFetch(
      new Response(JSON.stringify({ streams: [], total: 0 }), { status: 200 })
    );

    const jsx = await HomePage();
    render(jsx);

    expect(
      screen.getByText("Stream What You Believe")
    ).toBeInTheDocument();
    expect(
      screen.getByText(/No censorship. No filters/)
    ).toBeInTheDocument();
  });

  it("handles array response shape", async () => {
    mockFetch(new Response(JSON.stringify([]), { status: 200 }));

    const jsx = await HomePage();
    render(jsx);

    expect(screen.getByTestId("live-grid")).toBeInTheDocument();
  });
});
