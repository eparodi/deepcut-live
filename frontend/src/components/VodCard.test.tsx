import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { VodCard } from "./VodCard";
import type { VodItem } from "@/types";

const baseVod: VodItem = {
  id: "vod-123",
  userId: "user-456",
  userName: "TestStreamer",
  userAvatar: "https://example.com/avatar.jpg",
  title: "Amazing Stream",
  startedAt: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString(), // 2 days ago
  endedAt: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000 + 3600 * 1000).toISOString(),
  durationSeconds: 3661, // 1h 1m
  peakViewers: 42,
  totalViewers: 100,
  recordingPath: null,
  recordingStatus: "ready",
  createdAt: new Date().toISOString(),
};

describe("VodCard", () => {
  it("renders title and streamer name", () => {
    render(<VodCard vod={baseVod} />);
    expect(screen.getByText("Amazing Stream")).toBeInTheDocument();
    expect(screen.getByText("TestStreamer")).toBeInTheDocument();
  });

  it("renders fallback title when title is null", () => {
    const vod = { ...baseVod, title: null };
    render(<VodCard vod={vod} />);
    expect(screen.getByText("Untitled stream")).toBeInTheDocument();
  });

  it("shows duration badge for ready VODs", () => {
    render(<VodCard vod={baseVod} />);
    expect(screen.getByText("1h 1m")).toBeInTheDocument();
  });

  it("shows processing badge", () => {
    const vod = { ...baseVod, recordingStatus: "processing" as const };
    render(<VodCard vod={vod} />);
    expect(screen.getByText("Processing")).toBeInTheDocument();
  });

  it("shows failed badge", () => {
    const vod = { ...baseVod, recordingStatus: "failed" as const };
    render(<VodCard vod={vod} />);
    expect(screen.getByText("Unavailable")).toBeInTheDocument();
  });

  it("renders relative time", () => {
    render(<VodCard vod={baseVod} />);
    // 2 days ago
    expect(screen.getByText("2d ago")).toBeInTheDocument();
  });

  it("formats short duration correctly", () => {
    const vod = { ...baseVod, durationSeconds: 2700 };
    render(<VodCard vod={vod} />);
    expect(screen.getByText("45m")).toBeInTheDocument();
  });

  it("does not show duration badge when durationSeconds is null", () => {
    const vod = { ...baseVod, durationSeconds: null };
    render(<VodCard vod={vod} />);
    // The duration badge text is like "1h 1m" — neither should appear
    expect(screen.queryByText(/\d+h/)).not.toBeInTheDocument();
    expect(screen.queryByText(/\d+m\b/)).not.toBeInTheDocument();
  });

  it("links to VOD page", () => {
    render(<VodCard vod={baseVod} />);
    const link = screen.getByRole("listitem");
    expect(link).toHaveAttribute("href", "/vods/vod-123");
  });

  it("renders initials fallback when avatar is null", () => {
    const vod = { ...baseVod, userAvatar: null };
    render(<VodCard vod={vod} />);
    expect(screen.getByText("T")).toBeInTheDocument();
  });

  it("renders avatar image when userAvatar is provided", () => {
    render(<VodCard vod={baseVod} />);
    const img = screen.getByAltText("TestStreamer");
    expect(img).toBeInTheDocument();
    expect(img).toHaveAttribute("src", "https://example.com/avatar.jpg");
  });
});
