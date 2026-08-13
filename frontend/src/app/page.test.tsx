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

vi.mock("@/lib/api", () => ({
  getLiveStreams: vi.fn(),
  getRecentVods: vi.fn(),
}));

import HomePage from "./page";
import { getLiveStreams, getRecentVods } from "@/lib/api";

const mockGetLiveStreams = vi.mocked(getLiveStreams);
const mockGetRecentVods = vi.mocked(getRecentVods);

// ── Helpers ────────────────────────────────────────────────────

const emptySearchResponse = {
  vods: [],
  totalCount: 0,
  limit: 8,
  offset: 0,
};

// ── Tests ──────────────────────────────────────────────────────

describe("HomePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetRecentVods.mockResolvedValue(emptySearchResponse);
  });

  it("renders the LiveGrid with streams", async () => {
    mockGetLiveStreams.mockResolvedValue({
      streams: [
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
      ],
      total: 1,
    });

    const jsx = await HomePage();
    render(jsx);

    expect(screen.getByTestId("live-grid")).toBeInTheDocument();
    expect(screen.getByText(/1 stream/)).toBeInTheDocument();
  });

  it("renders empty LiveGrid when no streams", async () => {
    mockGetLiveStreams.mockResolvedValue({ streams: [], total: 0 });

    const jsx = await HomePage();
    render(jsx);

    expect(screen.getByTestId("live-grid")).toBeInTheDocument();
    expect(screen.getByText(/0 streams/)).toBeInTheDocument();
  });

  it("shows an error state (not the empty state) when the API fails", async () => {
    mockGetLiveStreams.mockRejectedValue(new Error("Network down"));

    const jsx = await HomePage();
    render(jsx);

    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(
      screen.getByText(/Couldn't load streams right now/)
    ).toBeInTheDocument();
    // Must NOT claim the platform is empty when the backend is down.
    expect(
      screen.queryByText(/Nothing live right now/)
    ).not.toBeInTheDocument();
    expect(screen.queryByTestId("live-grid")).not.toBeInTheDocument();
  });

  it("shows hero text when no live streams", async () => {
    mockGetLiveStreams.mockResolvedValue({ streams: [], total: 0 });

    const jsx = await HomePage();
    render(jsx);

    expect(
      screen.getByText("Stream What You Believe")
    ).toBeInTheDocument();
    expect(
      screen.getByText(/No censorship. No filters/)
    ).toBeInTheDocument();
  });

  it("shows the empty state when nothing is live and no VODs exist", async () => {
    mockGetLiveStreams.mockResolvedValue({ streams: [], total: 0 });

    const jsx = await HomePage();
    render(jsx);

    expect(
      screen.getByText(/Nothing live right now/)
    ).toBeInTheDocument();
  });

  it("shows recent VODs when nothing is live but VODs exist", async () => {
    mockGetLiveStreams.mockResolvedValue({ streams: [], total: 0 });
    mockGetRecentVods.mockResolvedValue({
      vods: [
        {
          id: "vod-1",
          userId: "user-1",
          userName: "TestStreamer",
          userAvatar: null,
          title: "Past Stream",
          startedAt: "2026-01-01T00:00:00Z",
          endedAt: "2026-01-01T01:00:00Z",
          durationSeconds: 3600,
          peakViewers: 10,
          totalViewers: 100,
          recordingPath: null,
          recordingStatus: "ready" as const,
          thumbnailUrl: null,
          createdAt: "2026-01-01T01:00:00Z",
        },
      ],
      totalCount: 1,
      limit: 8,
      offset: 0,
    });

    const jsx = await HomePage();
    render(jsx);

    expect(screen.getByText(/Recent Past Streams/)).toBeInTheDocument();
    expect(screen.getByText("Past Stream")).toBeInTheDocument();
  });
});
