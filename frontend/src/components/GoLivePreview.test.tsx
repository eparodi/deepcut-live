import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GoLivePreview } from "./GoLivePreview";

vi.mock("@/lib/api", () => ({
  getChannel: vi.fn(),
}));

vi.mock("@/components/VideoPlayer", () => ({
  VideoPlayer: ({ hlsUrl, isLive }: { hlsUrl: string; isLive: boolean }) => (
    <div data-testid="video-player" data-hls={hlsUrl} data-live={isLive}>
      VideoPlayer
    </div>
  ),
}));

import { getChannel } from "@/lib/api";

describe("GoLivePreview", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading skeleton initially", () => {
    vi.mocked(getChannel).mockReturnValue(new Promise(() => {}));

    render(<GoLivePreview userId="user-1" isLive={false} />);

    const skeletons = document.querySelectorAll(".skeleton");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("shows offline state when user is not live", async () => {
    vi.mocked(getChannel).mockResolvedValue({
      userId: "user-1",
      streamerName: "Test User",
      streamerAvatarUrl: "https://example.com/avatar.jpg",
      streamTitle: "My Stream",
      streamCategory: "Gaming",
      isLive: false,
      viewerCount: 0,
      hlsUrl: null,
      thumbnailUrl: null,
      startedAt: null,
      streamId: null,
    });

    render(<GoLivePreview userId="user-1" isLive={false} />);

    await waitFor(() => {
      expect(screen.getByText("Not streaming yet")).toBeInTheDocument();
    });
  });

  it("shows live preview when user is live", async () => {
    vi.mocked(getChannel).mockResolvedValue({
      userId: "user-1",
      streamerName: "Test User",
      streamerAvatarUrl: "https://example.com/avatar.jpg",
      streamTitle: "My Live Stream",
      streamCategory: "Just Chatting",
      isLive: true,
      viewerCount: 42,
      hlsUrl: "/hls/live/test.m3u8",
      thumbnailUrl: "/hls/live/test.m3u8",
      startedAt: "2026-08-10T12:00:00Z",
      streamId: "stream-1",
    });

    render(<GoLivePreview userId="user-1" isLive={true} />);

    await waitFor(() => {
      expect(screen.getByText("LIVE")).toBeInTheDocument();
      expect(screen.getByTestId("video-player")).toBeInTheDocument();
      expect(screen.getByText("📺 View My Stream")).toBeInTheDocument();
      expect(screen.getByText("42 viewers")).toBeInTheDocument();
    });
  });

  it("shows stream key reassurance message", async () => {
    vi.mocked(getChannel).mockResolvedValue({
      userId: "user-1",
      streamerName: "Test User",
      streamerAvatarUrl: "https://example.com/avatar.jpg",
      streamTitle: "My Live Stream",
      streamCategory: "Just Chatting",
      isLive: true,
      viewerCount: 1,
      hlsUrl: "/hls/live/test.m3u8",
      thumbnailUrl: "/hls/live/test.m3u8",
      startedAt: "2026-08-10T12:00:00Z",
      streamId: "stream-1",
    });

    render(<GoLivePreview userId="user-1" isLive={true} />);

    await waitFor(() => {
      expect(
        screen.getByText(/Your stream key is the same one shown above/)
      ).toBeInTheDocument();
    });
  });

  it("shows offline state with channel link", async () => {
    vi.mocked(getChannel).mockResolvedValue({
      userId: "user-1",
      streamerName: "Test User",
      streamerAvatarUrl: "https://example.com/avatar.jpg",
      streamTitle: "My Stream",
      streamCategory: "Gaming",
      isLive: false,
      viewerCount: 0,
      hlsUrl: null,
      thumbnailUrl: null,
      startedAt: null,
      streamId: null,
    });

    render(<GoLivePreview userId="user-1" isLive={false} />);

    await waitFor(() => {
      expect(screen.getByText("📺 View My Channel")).toBeInTheDocument();
      expect(
        screen.getByText(/Your stream key stays the same across sessions/)
      ).toBeInTheDocument();
    });
  });

  it("shows an error state (not the empty state) when the API fails", async () => {
    vi.mocked(getChannel).mockRejectedValue(new Error("Network error"));

    render(<GoLivePreview userId="user-1" isLive={false} />);

    await waitFor(() => {
      expect(screen.getByText("Couldn't load your stream")).toBeInTheDocument();
    });
    expect(screen.queryByText("Not streaming yet")).not.toBeInTheDocument();
  });

  it("recovers to the live state via Retry after a failure", async () => {
    const user = userEvent.setup();
    vi.mocked(getChannel)
      .mockRejectedValueOnce(new Error("Network error"))
      .mockResolvedValue({
        userId: "user-1",
        streamerName: "Test User",
        streamerAvatarUrl: "https://example.com/avatar.jpg",
        streamTitle: "My Live Stream",
        streamCategory: "Just Chatting",
        isLive: true,
        viewerCount: 42,
        hlsUrl: "/hls/live/test.m3u8",
        thumbnailUrl: "/hls/live/test.m3u8",
        startedAt: "2026-08-10T12:00:00Z",
        streamId: "stream-1",
      });

    render(<GoLivePreview userId="user-1" isLive={true} />);

    await waitFor(() => {
      expect(screen.getByText("Couldn't load your stream")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Retry"));

    await waitFor(() => {
      expect(screen.getByText("LIVE")).toBeInTheDocument();
    });
  });
});
