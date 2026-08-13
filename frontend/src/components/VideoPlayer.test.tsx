import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, act, within } from "@testing-library/react";
import { VideoPlayer } from "./VideoPlayer";

type HlsInstance = {
  on: ReturnType<typeof vi.fn>;
  loadSource: ReturnType<typeof vi.fn>;
  attachMedia: ReturnType<typeof vi.fn>;
  destroy: ReturnType<typeof vi.fn>;
  startLoad: ReturnType<typeof vi.fn>;
  recoverMediaError: ReturnType<typeof vi.fn>;
};

// Instances created by the mocked Hls constructor, so tests can fire the
// MANIFEST_PARSED callback and reach the "live" player state.
const hlsInstances = vi.hoisted(() => [] as HlsInstance[]);

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
    constructor() {
      hlsInstances.push(this as unknown as HlsInstance);
    }
  }
  return { default: MockHls };
});

describe("VideoPlayer", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    hlsInstances.length = 0;
    // Mock canPlayType to return empty string (no native HLS support)
    // so the component uses hls.js
    const videoProto = HTMLVideoElement.prototype;
    vi.spyOn(videoProto, "play").mockResolvedValue();
  });

  /**
   * Fires the mocked hls.js MANIFEST_PARSED callback so the player
   * transitions to the "live" state (where the control bar renders).
   */
  function activateLivePlayer() {
    const instance = hlsInstances[hlsInstances.length - 1];
    const manifestCb = instance.on.mock.calls.find(
      (call: unknown[]) => call[0] === "hlsManifestParsed"
    )?.[1] as (() => void) | undefined;
    expect(manifestCb).toBeTypeOf("function");
    act(() => manifestCb?.());
  }

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

  // Regression (specs/ui-ux-audit-v1.md D2): hidden controls must not
  // stay in the accessibility tree / tab order (WCAG 2.4.11).
  it("removes hidden control bar from the accessibility tree while playing", () => {
    render(
      <VideoPlayer
        hlsUrl="https://example.com/stream.m3u8"
        isLive
      />
    );
    activateLivePlayer();

    const bar = screen.getByTestId("controls-bar");
    expect(bar.className).toContain("invisible");
    expect(bar.className).toContain("pointer-events-none");
  });

  it("keeps the control bar reachable when paused", () => {
    render(
      <VideoPlayer
        hlsUrl="https://example.com/stream.m3u8"
        isLive
      />
    );
    activateLivePlayer();

    const video = document.querySelector("video");
    expect(video).toBeInTheDocument();
    act(() => {
      video?.dispatchEvent(new Event("pause"));
    });

    const bar = screen.getByTestId("controls-bar");
    expect(bar.className).not.toContain("invisible");
    expect(within(bar).getByRole("button", { name: "Play" })).toBeInTheDocument();
  });
});
