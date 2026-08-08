import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { LiveStreamCard } from "./LiveStreamCard";
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

describe("LiveStreamCard", () => {
  it("renders streamer name and title", () => {
    render(<LiveStreamCard stream={baseStream} />);
    expect(screen.getByText("TestStreamer")).toBeInTheDocument();
    expect(screen.getByText("Awesome Stream")).toBeInTheDocument();
  });

  it("renders LIVE badge", () => {
    render(<LiveStreamCard stream={baseStream} />);
    expect(screen.getByText("LIVE")).toBeInTheDocument();
  });

  it("renders the category pill", () => {
    render(<LiveStreamCard stream={baseStream} />);
    expect(screen.getByText("Gaming")).toBeInTheDocument();
  });

  it("shows 'Untitled stream' when title is empty", () => {
    const noTitle = { ...baseStream, title: "" };
    render(<LiveStreamCard stream={noTitle} />);
    expect(screen.getByText("Untitled stream")).toBeInTheDocument();
  });

  it("does not render category pill when category is null", () => {
    const noCategory = { ...baseStream, category: null };
    render(<LiveStreamCard stream={noCategory} />);
    expect(screen.queryByText("Gaming")).not.toBeInTheDocument();
  });

  it("renders thumbnail img when thumbnailUrl is provided", () => {
    render(<LiveStreamCard stream={baseStream} />);
    const img = screen.getByAltText("TestStreamer's stream thumbnail");
    expect(img).toBeInTheDocument();
    expect(img).toHaveAttribute("src", baseStream.thumbnailUrl);
  });

  it("shows placeholder when no thumbnail", () => {
    const noThumb = { ...baseStream, thumbnailUrl: null };
    render(<LiveStreamCard stream={noThumb} />);
    expect(screen.getByLabelText("No thumbnail")).toBeInTheDocument();
  });

  it("links to the channel page", () => {
    render(<LiveStreamCard stream={baseStream} />);
    const link = screen.getByRole("listitem");
    expect(link).toHaveAttribute("href", "/channel/user-1");
  });

  it("formats viewer count with k suffix", () => {
    render(<LiveStreamCard stream={{ ...baseStream, viewerCount: 1500 }} />);
    expect(screen.getByText("1.5k")).toBeInTheDocument();
  });

  it("formats viewer count with M suffix", () => {
    render(
      <LiveStreamCard stream={{ ...baseStream, viewerCount: 2_500_000 }} />
    );
    expect(screen.getByText("2.5M")).toBeInTheDocument();
  });

  it("formats small viewer counts without suffix", () => {
    render(<LiveStreamCard stream={{ ...baseStream, viewerCount: 42 }} />);
    expect(screen.getByText("42")).toBeInTheDocument();
  });

  it("applies isNew animation class", () => {
    const { container } = render(
      <LiveStreamCard stream={baseStream} isNew />
    );
    const link = container.querySelector("a");
    expect(link).toHaveClass("animate-fade-up");
  });

  it("has accessible aria-label on the card", () => {
    render(<LiveStreamCard stream={baseStream} />);
    const card = screen.getByRole("listitem");
    expect(card).toHaveAttribute(
      "aria-label",
      "TestStreamer is live: Awesome Stream. 1234 viewers"
    );
  });
});
