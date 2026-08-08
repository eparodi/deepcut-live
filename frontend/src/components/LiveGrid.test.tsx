import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { LiveGrid } from "./LiveGrid";
import type { LiveStream } from "@/types";

const baseStream: LiveStream = {
  userId: "user-1",
  streamerName: "TestStreamer",
  streamerAvatarUrl: "https://example.com/avatar.jpg",
  streamId: "stream-1",
  title: "Awesome Stream",
  category: "Gaming",
  viewerCount: 1234,
  thumbnailUrl: "https://example.com/thumb.jpg",
  startedAt: "2026-01-01T00:00:00Z",
};

describe("LiveGrid", () => {
  it("renders empty state when no streams", () => {
    render(<LiveGrid streams={[]} total={0} />);
    expect(screen.getByText("No one is live right now")).toBeInTheDocument();
  });

  it("renders 'Browse past streams' link in empty state", () => {
    render(<LiveGrid streams={[]} total={0} />);
    expect(screen.getByText("Browse past streams")).toBeInTheDocument();
  });

  it("renders stream cards for each stream", () => {
    render(<LiveGrid streams={[baseStream]} total={1} />);
    expect(screen.getByText("TestStreamer")).toBeInTheDocument();
    expect(screen.getByText("Awesome Stream")).toBeInTheDocument();
  });

  it("renders total count with plural 'streams'", () => {
    render(<LiveGrid streams={[baseStream, { ...baseStream, userId: "user-2" }]} total={2} />);
    expect(screen.getByText(/2 streams/)).toBeInTheDocument();
  });

  it("renders total count with singular 'stream'", () => {
    render(<LiveGrid streams={[baseStream]} total={1} />);
    expect(screen.getByText(/1 stream/)).toBeInTheDocument();
  });

  it("defaults to grid view", () => {
    render(<LiveGrid streams={[baseStream]} total={1} />);
    expect(screen.getByLabelText("Grid view")).toHaveAttribute("aria-pressed", "true");
  });

  it("renders live indicator dot", () => {
    const { container } = render(<LiveGrid streams={[baseStream]} total={1} />);
    // The animated pulse dot for LIVE NOW
    const pulseDot = container.querySelector(".animate-pulse");
    expect(pulseDot).toBeInTheDocument();
  });
});
