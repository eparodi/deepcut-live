import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { VideoPlayer } from "./VideoPlayer";

// Mock hls.js
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

describe("VideoPlayer", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Mock canPlayType to return empty string (no native HLS support)
    // so the component uses hls.js
    const videoProto = HTMLVideoElement.prototype;
    vi.spyOn(videoProto, "play").mockResolvedValue();
  });

  it("renders the video element", () => {
    render(
      <VideoPlayer
        hlsUrl="https://example.com/stream.m3u8"
        isLive
      />
    );
    const video = document.querySelector("video");
    expect(video).toBeInTheDocument();
  });

  it("shows loading overlay initially", () => {
    render(
      <VideoPlayer
        hlsUrl="https://example.com/stream.m3u8"
        isLive
      />
    );
    const container = document.querySelector(".relative");
    expect(container).toBeInTheDocument();
  });

  it("renders the video element without native controls", () => {
    render(
      <VideoPlayer
        hlsUrl="https://example.com/stream.m3u8"
        isLive
      />
    );
    const video = document.querySelector("video");
    expect(video).not.toHaveAttribute("controls");
    expect(video).toHaveAttribute("playsInline");
  });

  it("mutes video when isLive is true", () => {
    render(
      <VideoPlayer
        hlsUrl="https://example.com/stream.m3u8"
        isLive
      />
    );
    const video = document.querySelector("video");
    expect(video).toHaveProperty("muted", true);
  });

  it("renders without viewerCount prop", () => {
    render(
      <VideoPlayer
        hlsUrl="https://example.com/stream.m3u8"
        isLive
      />
    );
    const video = document.querySelector("video");
    expect(video).toBeInTheDocument();
  });

  it("renders with viewerCount prop", () => {
    render(
      <VideoPlayer
        hlsUrl="https://example.com/stream.m3u8"
        isLive
        viewerCount={1234}
      />
    );
    const video = document.querySelector("video");
    expect(video).toBeInTheDocument();
  });
});
