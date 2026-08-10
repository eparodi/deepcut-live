"use client";
// Client Component — needs useEffect for hls.js, useRef for video element,
// useState for states, mouse/keyboard event handlers for custom controls

import { useEffect, useRef, useState, useCallback } from "react";
import Hls from "hls.js";

type PlayerState =
  | "loading"
  | "live"
  | "paused"
  | "interrupted"
  | "ended"
  | "error"
  | "unsupported";

interface VideoPlayerProps {
  hlsUrl: string;
  isLive: boolean;
  vodId?: string;
  viewerCount?: number;
  onTheaterChange?: (active: boolean) => void;
}

// ---- Inline SVG Icons ----

function PlayIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
      <path d="M8 5v14l11-7z" />
    </svg>
  );
}

function PauseIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
      <path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z" />
    </svg>
  );
}

function VolumeHighIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
      <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z" />
    </svg>
  );
}

function VolumeLowIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
      <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02z" />
    </svg>
  );
}

function VolumeMuteIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
      <path d="M16.5 12c0-1.77-1.02-3.29-2.5-4.03v2.21l2.45 2.45c.03-.2.05-.41.05-.63zm2.5 0c0 .94-.2 1.82-.54 2.64l1.51 1.51C20.63 14.91 21 13.5 21 12c0-4.28-2.99-7.86-7-8.77v2.06c2.89.86 5 3.54 5 6.71zM4.27 3L3 4.27 7.73 9H3v6h4l5 5v-6.73l4.25 4.25c-.67.52-1.42.93-2.25 1.18v2.06c1.38-.31 2.63-.95 3.69-1.81L19.73 21 21 19.73l-9-9L4.27 3zM12 4L9.91 6.09 12 8.18V4z" />
    </svg>
  );
}

function TheaterIcon({ active }: { active: boolean }) {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" fill={active ? "currentColor" : "none"} stroke="currentColor" strokeWidth="2">
      <rect x="2" y="4" width="20" height="12" rx="2" />
      <path d="M6 20h12" />
      <path d="M12 16v4" />
    </svg>
  );
}

function FullscreenIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3" />
    </svg>
  );
}

function FullscreenExitIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M8 3v3a2 2 0 0 1-2 2H3m18 0h-3a2 2 0 0 1-2-2V3m0 18v-3a2 2 0 0 1 2-2h3M3 16h3a2 2 0 0 1 2 2v3" />
    </svg>
  );
}

function formatViewerCount(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return String(n);
}

// ---- Main Component ----

export function VideoPlayer({ hlsUrl, isLive, vodId, viewerCount = 0, onTheaterChange }: VideoPlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const hlsRef = useRef<Hls | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const hideTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const [playerState, setPlayerState] = useState<PlayerState>("loading");
  const [errorMessage, setErrorMessage] = useState<string>("");
  const [isPlaying, setIsPlaying] = useState(false);
  const [isMuted, setIsMuted] = useState(isLive);
  const [volume, setVolume] = useState(1);
  const [showControls, setShowControls] = useState(false);
  const [isTheaterMode, setIsTheaterMode] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);

  // ---- hls.js lifecycle (unchanged core logic) ----

  const destroyHls = useCallback(() => {
    if (hlsRef.current) {
      hlsRef.current.destroy();
      hlsRef.current = null;
    }
  }, []);

  const initPlayer = useCallback(() => {
    const video = videoRef.current;
    if (!video) return;

    setPlayerState("loading");
    setErrorMessage("");

    // Check if HLS is natively supported (Safari)
    if (video.canPlayType("application/vnd.apple.mpegurl")) {
      video.src = hlsUrl;
      video.play().catch(() => {
        // Autoplay may be blocked; that's fine, user can click play
      });
      setPlayerState(isLive ? "live" : "live");
      return;
    }

    // Check if Hls.js is supported
    if (!Hls.isSupported()) {
      setPlayerState("unsupported");
      return;
    }

    // Use hls.js
    const hls = new Hls({
      enableWorker: true,
      lowLatencyMode: isLive,
      backBufferLength: isLive ? 30 : 90,
      liveSyncDurationCount: isLive ? 1 : undefined,
      maxMaxBufferLength: isLive ? 10 : undefined,
    });

    hlsRef.current = hls;

    hls.loadSource(hlsUrl);
    hls.attachMedia(video);

    hls.on(Hls.Events.MANIFEST_PARSED, () => {
      setPlayerState("live");
      video.play().catch(() => {
        // Autoplay may be blocked
      });
    });

    hls.on(Hls.Events.ERROR, (_event, data) => {
      if (data.fatal) {
        switch (data.type) {
          case Hls.ErrorTypes.NETWORK_ERROR:
            setPlayerState("interrupted");
            setErrorMessage("Stream interrupted — reconnecting...");
            hls.startLoad();
            break;
          case Hls.ErrorTypes.MEDIA_ERROR:
            setPlayerState("interrupted");
            setErrorMessage("Media error — attempting to recover...");
            hls.recoverMediaError();
            break;
          default:
            setPlayerState("error");
            setErrorMessage("Could not load stream");
            destroyHls();
            break;
        }
      }
    });
  }, [hlsUrl, isLive, destroyHls]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- init-on-mount is the intended pattern
    initPlayer();
    return () => {
      destroyHls();
    };
  }, [initPlayer, destroyHls]);

  // ---- Video event listeners ----

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    const onPlay = () => {
      setIsPlaying(true);
      if (playerState === "paused") setPlayerState("live");
    };
    const onPause = () => {
      setIsPlaying(false);
      if (playerState === "live") setPlayerState("paused");
    };
    const onVolumeChange = () => {
      setIsMuted(video.muted);
      setVolume(video.volume);
    };

    video.addEventListener("play", onPlay);
    video.addEventListener("pause", onPause);
    video.addEventListener("volumechange", onVolumeChange);

    // Sync initial state
    setIsMuted(video.muted);
    setVolume(video.volume);

    return () => {
      video.removeEventListener("play", onPlay);
      video.removeEventListener("pause", onPause);
      video.removeEventListener("volumechange", onVolumeChange);
    };
  }, [playerState]);

  // Controls visibility: shown on hover, permanently when paused
  const controlsVisible = showControls || playerState === "paused";

  // ---- Controls auto-hide ----

  const resetHideTimer = useCallback(() => {
    if (hideTimerRef.current) clearTimeout(hideTimerRef.current);
    setShowControls(true);
    hideTimerRef.current = setTimeout(() => {
      setShowControls(false);
    }, 3000);
  }, []);

  // ---- Fullscreen change listener ----

  useEffect(() => {
    const onFsChange = () => {
      setIsFullscreen(!!document.fullscreenElement);
    };
    document.addEventListener("fullscreenchange", onFsChange);
    return () => document.removeEventListener("fullscreenchange", onFsChange);
  }, []);

  // ---- Theater mode ----

  useEffect(() => {
    if (isTheaterMode) {
      document.body.style.backgroundColor = "#000";
    } else {
      document.body.style.backgroundColor = "";
    }
    onTheaterChange?.(isTheaterMode);
    return () => {
      document.body.style.backgroundColor = "";
    };
  }, [isTheaterMode, onTheaterChange]);

  // ---- Player actions ----

  const togglePlay = useCallback(() => {
    const video = videoRef.current;
    if (!video) return;
    if (video.paused) {
      video.play().catch(() => {});
    } else {
      video.pause();
    }
  }, []);

  const toggleMute = useCallback(() => {
    const video = videoRef.current;
    if (!video) return;
    video.muted = !video.muted;
  }, []);

  const setVolumeLevel = useCallback((level: number) => {
    const video = videoRef.current;
    if (!video) return;
    const clamped = Math.max(0, Math.min(1, level));
    video.volume = clamped;
    video.muted = clamped === 0;
    setVolume(clamped);
  }, []);

  const toggleFullscreen = useCallback(() => {
    const container = containerRef.current;
    if (!container) return;
    if (document.fullscreenElement) {
      document.exitFullscreen().catch(() => {});
    } else {
      container.requestFullscreen().catch(() => {});
    }
  }, []);

  const toggleTheater = useCallback(() => {
    setIsTheaterMode((prev) => !prev);
  }, []);

  // ---- Keyboard shortcuts ----

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Don't capture when typing in input fields
      const tag = (e.target as HTMLElement)?.tagName;
      if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return;

      switch (e.key) {
        case " ":
          e.preventDefault();
          togglePlay();
          break;
        case "f":
        case "F":
          e.preventDefault();
          toggleFullscreen();
          break;
        case "m":
        case "M":
          e.preventDefault();
          toggleMute();
          break;
        case "ArrowUp":
          e.preventDefault();
          setVolumeLevel(volume + 0.05);
          break;
        case "ArrowDown":
          e.preventDefault();
          setVolumeLevel(volume - 0.05);
          break;
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [togglePlay, toggleMute, toggleFullscreen, setVolumeLevel, volume]);

  const handleRetry = () => {
    destroyHls();
    initPlayer();
  };

  // ---- Render ----

  // Unsupported browser
  if (playerState === "unsupported") {
    return (
      <div
        className="relative w-full aspect-video flex flex-col items-center justify-center rounded-xl"
        style={{ backgroundColor: "var(--color-surface)" }}
      >
        <p className="text-4xl mb-3">🚫</p>
        <p className="text-lg text-[var(--color-text)] font-medium">
          Browser not supported
        </p>
        <p className="mt-2 text-sm text-[var(--color-text-muted)] max-w-xs text-center">
          Your browser does not support HLS playback. Please use{" "}
          <strong>Chrome</strong>, <strong>Firefox</strong>, or{" "}
          <strong>Edge</strong>.
        </p>
      </div>
    );
  }

  const isActive = playerState === "live" || playerState === "paused";

  return (
    <div
      ref={containerRef}
      className={`relative w-full aspect-video rounded-xl overflow-hidden bg-black group ${
        isTheaterMode ? "!max-w-full" : ""
      }`}
      onMouseMove={resetHideTimer}
      onMouseLeave={() => {
        if (hideTimerRef.current) clearTimeout(hideTimerRef.current);
        setShowControls(false);
      }}
      onTouchStart={resetHideTimer}
      onClick={togglePlay}
      onDoubleClick={toggleFullscreen}
    >
      {/* Video element */}
      <video
        ref={videoRef}
        className="w-full h-full"
        playsInline
        muted={isLive}
        poster={vodId ? `/thumbnails/${vodId}.jpg` : undefined}
      />

      {/* Loading overlay */}
      {playerState === "loading" && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/60 z-30">
          <div className="flex flex-col items-center gap-3">
            <div
              className="w-10 h-10 border-4 border-t-transparent rounded-full animate-spin"
              style={{ borderColor: "var(--color-primary)", borderTopColor: "transparent" }}
            />
            <p className="text-sm text-[var(--color-text-muted)]">
              Loading stream...
            </p>
          </div>
        </div>
      )}

      {/* Stream interrupted overlay */}
      {playerState === "interrupted" && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/70 z-30">
          <div className="flex flex-col items-center gap-3 text-center px-4">
            <div
              className="w-10 h-10 border-4 border-t-transparent rounded-full animate-spin"
              style={{
                borderColor: "var(--color-primary)",
                borderTopColor: "transparent",
              }}
            />
            <p
              className="text-sm font-medium"
              style={{ color: "var(--color-primary)" }}
            >
              Stream Interrupted
            </p>
            <p className="text-sm text-[var(--color-text-muted)]">
              {errorMessage || "Reconnecting..."}
            </p>
          </div>
        </div>
      )}

      {/* Stream ended overlay */}
      {playerState === "ended" && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/70 z-30">
          <div className="flex flex-col items-center gap-3 text-center px-4">
            <p className="text-4xl">📼</p>
            <p className="text-lg font-semibold text-[var(--color-text)]">
              Stream ended
            </p>
            {vodId && (
              <a
                href={`/vods/${vodId}`}
                className="inline-flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:opacity-90"
                style={{ backgroundColor: "var(--color-primary)" }}
              >
                Watch VOD
              </a>
            )}
          </div>
        </div>
      )}

      {/* Fatal error overlay */}
      {playerState === "error" && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/70 z-30">
          <div className="flex flex-col items-center gap-3 text-center px-4">
            <p className="text-4xl">⚠️</p>
            <p className="text-lg font-semibold text-[var(--color-text)]">
              Could not load stream
            </p>
            <p className="text-sm text-[var(--color-text-muted)] max-w-xs">
              {errorMessage || "An unexpected error occurred."}
            </p>
            <button
              onClick={(e) => { e.stopPropagation(); handleRetry(); }}
              className="inline-flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:opacity-90"
              style={{ backgroundColor: "var(--color-primary)" }}
            >
              Retry
            </button>
          </div>
        </div>
      )}

      {/* ---- Custom Controls (only when stream is active) ---- */}

      {isActive && (
        <>
          {/* LIVE badge — top-left */}
          {isLive && viewerCount > 0 && (
            <div className="absolute top-4 left-4 z-20">
              <span
                className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold text-white animate-fade-up"
                style={{ backgroundColor: "var(--color-live)" }}
              >
                <span
                  className="w-1.5 h-1.5 rounded-full bg-white animate-pulse"
                  style={{ animationDuration: "1.5s" }}
                />
                LIVE
                {controlsVisible && (
                  <span className="opacity-80 ml-0.5">
                    · {formatViewerCount(viewerCount)}
                    {controlsVisible && viewerCount >= 1000 && (
                      <span className="hidden sm:inline"> viewers</span>
                    )}
                  </span>
                )}
              </span>
            </div>
          )}

          {/* Center play button — shown when paused */}
          {playerState === "paused" && (
            <div className="absolute inset-0 flex items-center justify-center z-10">
              <button
                onClick={(e) => { e.stopPropagation(); togglePlay(); }}
                onDoubleClick={(e) => e.stopPropagation()}
                className="w-16 h-16 flex items-center justify-center rounded-full transition-opacity hover:opacity-90"
                style={{ backgroundColor: "rgba(0,0,0,0.5)", color: "var(--color-text)" }}
                aria-label="Play"
              >
                <PlayIcon />
              </button>
            </div>
          )}

          {/* Bottom control bar */}
          <div
            className={`absolute bottom-0 left-0 right-0 z-20 transition-opacity duration-300 ${
              controlsVisible ? "opacity-100" : "opacity-0 pointer-events-none"
            }`}
            style={{
              background: "linear-gradient(to top, rgba(0,0,0,0.8), transparent)",
              height: "64px",
            }}
          >
            <div className="absolute bottom-3 left-4 right-4 flex items-center gap-2">
              {/* Play/Pause */}
              <button
                onClick={(e) => { e.stopPropagation(); togglePlay(); }}
                onDoubleClick={(e) => e.stopPropagation()}
                className="p-2 rounded transition-colors hover:text-white"
                style={{ color: "var(--color-text)" }}
                aria-label={isPlaying ? "Pause" : "Play"}
              >
                {isPlaying ? <PauseIcon /> : <PlayIcon />}
              </button>

              {/* Volume */}
              <div className="flex items-center gap-1 group/vol">
                <button
                  onClick={(e) => { e.stopPropagation(); toggleMute(); }}
                  onDoubleClick={(e) => e.stopPropagation()}
                  className="p-2 rounded transition-colors hover:text-white"
                  style={{ color: "var(--color-text)" }}
                  aria-label={isMuted || volume === 0 ? "Unmute" : "Mute"}
                >
                  {isMuted || volume === 0 ? (
                    <VolumeMuteIcon />
                  ) : volume <= 0.5 ? (
                    <VolumeLowIcon />
                  ) : (
                    <VolumeHighIcon />
                  )}
                </button>
                <div className="hidden sm:flex items-center w-0 opacity-0 group-hover/vol:w-20 group-hover/vol:opacity-100 transition-all duration-150 overflow-hidden">
                  <input
                    type="range"
                    min="0"
                    max="100"
                    value={isMuted ? 0 : volume * 100}
                    onChange={(e) => {
                      e.stopPropagation();
                      setVolumeLevel(Number(e.target.value) / 100);
                    }}
                    className="w-20 h-1 appearance-none rounded cursor-pointer"
                    style={{
                      background: `linear-gradient(to right, var(--color-text) ${isMuted ? 0 : volume * 100}%, rgba(173,173,184,0.3) ${isMuted ? 0 : volume * 100}%)`,
                      accentColor: "var(--color-text)",
                    }}
                    aria-label="Volume"
                    aria-valuemin={0}
                    aria-valuemax={100}
                    aria-valuenow={isMuted ? 0 : Math.round(volume * 100)}
                    role="slider"
                  />
                </div>
              </div>

              {/* Spacer */}
              <div className="flex-1" />

              {/* Theater mode */}
              <button
                onClick={(e) => { e.stopPropagation(); toggleTheater(); }}
                onDoubleClick={(e) => e.stopPropagation()}
                className="hidden sm:flex p-2 rounded transition-colors hover:text-white"
                style={{ color: isTheaterMode ? "var(--color-primary)" : "var(--color-text)" }}
                aria-label={isTheaterMode ? "Exit theater mode" : "Theater mode"}
                aria-pressed={isTheaterMode}
              >
                <TheaterIcon active={isTheaterMode} />
              </button>

              {/* Fullscreen */}
              <button
                onClick={(e) => { e.stopPropagation(); toggleFullscreen(); }}
                onDoubleClick={(e) => e.stopPropagation()}
                className="p-2 rounded transition-colors hover:text-white"
                style={{ color: "var(--color-text)" }}
                aria-label={isFullscreen ? "Exit fullscreen" : "Fullscreen"}
              >
                {isFullscreen ? <FullscreenExitIcon /> : <FullscreenIcon />}
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
