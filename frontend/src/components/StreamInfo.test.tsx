import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { StreamInfo } from "./StreamInfo";
import type { ChannelResponse } from "@/types";

const liveChannel: ChannelResponse = {
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

const offlineChannel: ChannelResponse = {
  ...liveChannel,
  isLive: false,
  viewerCount: 0,
  hlsUrl: null,
  streamId: null,
};

describe("StreamInfo", () => {
  it("renders the stream title", () => {
    render(<StreamInfo channel={liveChannel} />);
    expect(screen.getByText("Awesome Stream")).toBeInTheDocument();
  });

  it("shows 'Untitled stream' when streamTitle is empty", () => {
    const noTitle = { ...liveChannel, streamTitle: "" };
    render(<StreamInfo channel={noTitle} />);
    expect(screen.getByText("Untitled stream")).toBeInTheDocument();
  });

  it("renders streamer name and avatar", () => {
    render(<StreamInfo channel={liveChannel} />);
    expect(screen.getByText("TestStreamer")).toBeInTheDocument();
    expect(screen.getByAltText("TestStreamer")).toBeInTheDocument();
  });

  it("renders viewer count when live", () => {
    render(<StreamInfo channel={liveChannel} />);
    expect(screen.getByText("1.2k viewers")).toBeInTheDocument();
  });

  it("renders 'Offline' when not live", () => {
    render(<StreamInfo channel={offlineChannel} />);
    expect(screen.getByText("Offline")).toBeInTheDocument();
  });

  it("renders category pill when category exists", () => {
    render(<StreamInfo channel={liveChannel} />);
    expect(screen.getByText("Gaming")).toBeInTheDocument();
  });

  it("does not render category pill when category is null", () => {
    const noCategory = { ...liveChannel, streamCategory: null };
    render(<StreamInfo channel={noCategory} />);
    expect(screen.queryByText("Gaming")).not.toBeInTheDocument();
  });

  it("formats small viewer counts without suffix", () => {
    const small = { ...liveChannel, viewerCount: 42 };
    render(<StreamInfo channel={small} />);
    expect(screen.getByText("42 viewers")).toBeInTheDocument();
  });
});
