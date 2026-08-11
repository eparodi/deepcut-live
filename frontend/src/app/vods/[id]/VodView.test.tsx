import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { VodView } from "./VodView";
import type { VodDetail } from "@/types";

// ── Mocks ──────────────────────────────────────────────────────

vi.mock("@/components/VideoPlayer", () => ({
  VideoPlayer: ({ isLive, vodId }: { isLive: boolean; vodId?: string; hlsUrl: string; viewerCount?: number }) => (
    <div data-testid="video-player">
      VideoPlayer: isLive={String(isLive)} vodId={vodId || "none"}
    </div>
  ),
}));

vi.mock("@/components/ChatPanel", () => ({
  ChatPanel: ({ streamId, isStreamEnded }: { streamId: string; isStreamEnded: boolean; isSignedIn: boolean }) => (
    <div data-testid="chat-panel">
      ChatPanel: streamId={streamId} isStreamEnded={String(isStreamEnded)}
    </div>
  ),
}));

// ── Test data ──────────────────────────────────────────────────

const baseVod: VodDetail = {
  id: "vod-123",
  userId: "user-456",
  userName: "TestStreamer",
  userAvatar: "https://example.com/avatar.jpg",
  title: "Amazing Stream",
  startedAt: "2026-01-15T00:00:00Z",
  endedAt: "2026-01-15T01:30:00Z",
  durationSeconds: 5400,
  peakViewers: 1200,
  totalViewers: 5000,
  recordingPath: null,
  recordingStatus: "ready",
  createdAt: "2026-01-15T00:00:00Z",
  hlsUrl: null,
};

const hlsUrl = "/hls/vods/vod-123/index.m3u8";

// ── Tests ──────────────────────────────────────────────────────

describe("VodView", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders player and chat for ready VOD", () => {
    render(<VodView vod={baseVod} hlsUrl={hlsUrl} />);
    expect(screen.getByTestId("video-player")).toBeInTheDocument();
    // Two ChatPanels: one for desktop (hidden), one for mobile (hidden)
    const chatPanels = screen.getAllByTestId("chat-panel");
    expect(chatPanels).toHaveLength(2);
  });

  it("shows processing state", () => {
    const vod = { ...baseVod, recordingStatus: "processing" as const };
    render(<VodView vod={vod} hlsUrl={hlsUrl} />);
    expect(screen.getByText("Processing — available soon")).toBeInTheDocument();
  });

  it("shows processing state for pending VOD", () => {
    const vod = { ...baseVod, recordingStatus: "pending" as const };
    render(<VodView vod={vod} hlsUrl={hlsUrl} />);
    expect(screen.getByText("Processing — available soon")).toBeInTheDocument();
  });

  it("shows failed state", () => {
    const vod = { ...baseVod, recordingStatus: "failed" as const };
    render(<VodView vod={vod} hlsUrl={hlsUrl} />);
    expect(
      screen.getByText("This recording is unavailable")
    ).toBeInTheDocument();
  });

  it("shows failed message when provided", () => {
    const vod = {
      ...baseVod,
      recordingStatus: "failed" as const,
      message: "Recording file corrupted",
    };
    render(<VodView vod={vod} hlsUrl={hlsUrl} />);
    expect(screen.getByText("Recording file corrupted")).toBeInTheDocument();
  });

  it("renders title, streamer name, and date", () => {
    render(<VodView vod={baseVod} hlsUrl={hlsUrl} />);
    expect(screen.getByText("Amazing Stream")).toBeInTheDocument();
    expect(screen.getByText("TestStreamer")).toBeInTheDocument();
    // Date format: "Jan 15, 2026"
    expect(screen.getByText(/Jan 15, 2026/)).toBeInTheDocument();
  });

  it("renders duration and view count", () => {
    render(<VodView vod={baseVod} hlsUrl={hlsUrl} />);
    expect(screen.getByText("1h 30m")).toBeInTheDocument();
    expect(screen.getByText("5k views")).toBeInTheDocument();
  });

  it("renders fallback title when title is null", () => {
    const vod = { ...baseVod, title: null };
    render(<VodView vod={vod} hlsUrl={hlsUrl} />);
    expect(screen.getByText("Untitled stream")).toBeInTheDocument();
  });

  it("streamer name links to channel page", () => {
    render(<VodView vod={baseVod} hlsUrl={hlsUrl} />);
    const link = screen.getByText("TestStreamer").closest("a");
    expect(link).toHaveAttribute("href", "/channel/user-456");
  });

  it("renders back link to search", () => {
    render(<VodView vod={baseVod} hlsUrl={hlsUrl} />);
    const link = screen.getByText("← Back to search");
    expect(link.closest("a")).toHaveAttribute("href", "/search");
  });
});
