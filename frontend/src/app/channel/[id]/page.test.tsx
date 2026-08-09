import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";

// ── Mocks ──────────────────────────────────────────────────────

const mockNotFound = vi.fn();
vi.mock("next/navigation", () => ({
  notFound: (...args: unknown[]) => mockNotFound(...args),
}));

vi.mock("next/headers", () => ({
  cookies: vi.fn(),
}));

vi.mock("@/lib/api", () => ({
  getChannel: vi.fn(),
}));

// Mock hls.js (required by VideoPlayer which is rendered inside ChannelPage)
vi.mock("hls.js", () => {
  class MockHls {
    static isSupported() {
      return true;
    }
    static Events = {
      MANIFEST_PARSED: "hlsManifestParsed",
      ERROR: "hlsError",
    };
    static ErrorTypes = {
      NETWORK_ERROR: "networkError",
      MEDIA_ERROR: "mediaError",
      OTHER_ERROR: "otherError",
    };
    loadSource = vi.fn();
    attachMedia = vi.fn();
    destroy = vi.fn();
    on = vi.fn();
    startLoad = vi.fn();
    recoverMediaError = vi.fn();
  }
  return { default: MockHls };
});

// jsdom doesn't implement scrollIntoView (used by ChatPanel)
Element.prototype.scrollIntoView = vi.fn();

import { cookies } from "next/headers";
import { getChannel } from "@/lib/api";
import ChannelPage from "./page";
import type { ChannelResponse } from "@/types";

// ── Helpers ────────────────────────────────────────────────────

function mockTokenCookie(token: string | null) {
  const get = vi.fn((name: string) => {
    if (name === "token" && token) {
      return { value: token, name: "token" };
    }
    return undefined;
  });
  vi.mocked(cookies).mockResolvedValue({
    get,
  } as unknown as Awaited<ReturnType<typeof cookies>>);
}

const baseChannel: ChannelResponse = {
  userId: "user-1",
  streamerName: "TestStreamer",
  streamerAvatarUrl: "https://example.com/avatar.jpg",
  streamTitle: "Awesome Stream",
  streamCategory: "Gaming",
  isLive: true,
  viewerCount: 1234,
  hlsUrl: "https://example.com/stream.m3u8",
  startedAt: "2026-01-01T00:00:00Z",
  streamId: "stream-1",
};

// ── Tests ──────────────────────────────────────────────────────

describe("ChannelPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Mock HTMLVideoElement.prototype.play to avoid unhandled promise rejections
    vi.spyOn(HTMLVideoElement.prototype, "play").mockResolvedValue();
  });

  it("renders the channel page with stream info", async () => {
    mockTokenCookie(null);
    vi.mocked(getChannel).mockResolvedValue(baseChannel);

    const jsx = await ChannelPage({
      params: Promise.resolve({ id: "user-1" }),
    });
    render(jsx);

    expect(screen.getByText("TestStreamer")).toBeInTheDocument();
    expect(screen.getByText("Awesome Stream")).toBeInTheDocument();
  });

  it("renders browse streams link", async () => {
    mockTokenCookie(null);
    vi.mocked(getChannel).mockResolvedValue(baseChannel);

    const jsx = await ChannelPage({
      params: Promise.resolve({ id: "user-1" }),
    });
    render(jsx);

    expect(screen.getByText("← Browse streams")).toBeInTheDocument();
  });

  it("calls notFound when channel returns 404", async () => {
    mockTokenCookie(null);
    const notFoundError = new Error("Not found") as Error & {
      status: number;
    };
    notFoundError.status = 404;
    vi.mocked(getChannel).mockRejectedValue(notFoundError);

    try {
      await ChannelPage({ params: Promise.resolve({ id: "nonexistent" }) });
    } catch {
      // Component may throw after notFound()
    }

    expect(mockNotFound).toHaveBeenCalled();
  });

  it("renders waiting state when no hlsUrl but isLive", async () => {
    mockTokenCookie(null);
    const channelWithoutHls: ChannelResponse = {
      ...baseChannel,
      hlsUrl: null,
      isLive: true,
    };
    vi.mocked(getChannel).mockResolvedValue(channelWithoutHls);

    const jsx = await ChannelPage({
      params: Promise.resolve({ id: "user-1" }),
    });
    render(jsx);

    expect(screen.getByText("Waiting for stream...")).toBeInTheDocument();
  });

  it("renders offline state when not live and no hlsUrl", async () => {
    mockTokenCookie(null);
    const offlineChannel: ChannelResponse = {
      ...baseChannel,
      hlsUrl: null,
      isLive: false,
    };
    vi.mocked(getChannel).mockResolvedValue(offlineChannel);

    const jsx = await ChannelPage({
      params: Promise.resolve({ id: "user-1" }),
    });
    render(jsx);

    expect(screen.getByText("Stream is offline")).toBeInTheDocument();
  });

  it("renders past streams link", async () => {
    mockTokenCookie(null);
    vi.mocked(getChannel).mockResolvedValue(baseChannel);

    const jsx = await ChannelPage({
      params: Promise.resolve({ id: "user-1" }),
    });
    render(jsx);

    expect(
      screen.getByText("📼 View past streams →")
    ).toBeInTheDocument();
  });
});
