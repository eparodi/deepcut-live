"use client";
// Client Component — needs useEffect for hls.js, useRef for video element, useState for states

import { useEffect, useRef, useState, useCallback } from "react";
import Hls from "hls.js";

type PlayerState =
  | "loading"
  | "live"
  | "interrupted"
  | "ended"
  | "error"
  | "unsupported";

interface VideoPlayerProps {
  hlsUrl: string;
  isLive: boolean;
  vodId?: string;
}

export function VideoPlayer({ hlsUrl, isLive, vodId }: VideoPlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const hlsRef = useRef<Hls | null>(null);
  const [playerState, setPlayerState] = useState<PlayerState>("loading");
  const [errorMessage, setErrorMessage] = useState<string>("");

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
            // hls.js will attempt to recover automatically
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

  const handleRetry = () => {
    destroyHls();
    initPlayer();
  };

  // --- Render states ---

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

  return (
    <div className="relative w-full aspect-video rounded-xl overflow-hidden bg-black">
      {/* Video element — always rendered */}
      <video
        ref={videoRef}
        className="w-full h-full"
        controls
        playsInline
        muted={isLive} // Muted for autoplay policy on live streams
        poster={vodId ? `/thumbnails/${vodId}.jpg` : undefined}
      />

      {/* Loading overlay */}
      {playerState === "loading" && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/60">
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
        <div className="absolute inset-0 flex items-center justify-center bg-black/70">
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
        <div className="absolute inset-0 flex items-center justify-center bg-black/70">
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
        <div className="absolute inset-0 flex items-center justify-center bg-black/70">
          <div className="flex flex-col items-center gap-3 text-center px-4">
            <p className="text-4xl">⚠️</p>
            <p className="text-lg font-semibold text-[var(--color-text)]">
              Could not load stream
            </p>
            <p className="text-sm text-[var(--color-text-muted)] max-w-xs">
              {errorMessage || "An unexpected error occurred."}
            </p>
            <button
              onClick={handleRetry}
              className="inline-flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:opacity-90"
              style={{ backgroundColor: "var(--color-primary)" }}
            >
              Retry
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
