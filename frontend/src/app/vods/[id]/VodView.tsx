"use client";
// Client Component — VideoPlayer + ChatPanel need interactivity

import Link from "next/link";
import { VideoPlayer } from "@/components/VideoPlayer";
import { ChatPanel } from "@/components/ChatPanel";
import { AVATAR_FALLBACK } from "@/lib/fallbacks";
import type { VodDetail } from "@/types";

interface VodViewProps {
  vod: VodDetail;
  hlsUrl: string;
}

/** Format duration in seconds to human-readable: 3661 → "1h 1m", 45 → "45s" */
function formatDuration(totalSeconds: number): string {
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  if (minutes > 0) {
    return `${minutes}m`;
  }
  return `${seconds}s`;
}

/** Format a viewer count for display: 1205 → "1.2k" */
function formatViewCount(n: number): string {
  if (n >= 1_000_000) {
    const val = n / 1_000_000;
    return val % 1 === 0 ? `${val}M` : `${val.toFixed(1)}M`;
  }
  if (n >= 1_000) {
    const val = n / 1_000;
    return val % 1 === 0 ? `${val}k` : `${val.toFixed(1)}k`;
  }
  return n.toLocaleString();
}

/** Format an ISO date to a readable string: "Jan 15, 2026" */
function formatDate(isoDate: string): string {
  const d = new Date(isoDate);
  const months = [
    "Jan", "Feb", "Mar", "Apr", "May", "Jun",
    "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
  ];
  return `${months[d.getUTCMonth()]} ${d.getUTCDate()}, ${d.getUTCFullYear()}`;
}

export function VodView({ vod, hlsUrl }: VodViewProps) {
  if (!vod) {
    return (
      <div className="min-h-full flex flex-col">
        <main className="flex-1 max-w-7xl mx-auto w-full px-6 py-6">
          <div className="text-center py-16">
            <p className="text-lg text-[var(--color-text-muted)]">
              Unable to load this recording.
            </p>
          </div>
        </main>
      </div>
    );
  }

  const {
    id,
    userName,
    userAvatar,
    title,
    startedAt,
    durationSeconds,
    totalViewers,
    recordingStatus,
  } = vod;

  const durationLabel =
    durationSeconds != null ? formatDuration(durationSeconds) : null;
  const viewLabel = `${formatViewCount(totalViewers)} views`;
  const dateLabel = formatDate(startedAt);

  // Use the VOD's id as the streamId for chat replay
  const streamId = id;

  return (
    <div className="min-h-full flex flex-col">
      <main className="flex-1 max-w-7xl mx-auto w-full px-6 py-6">
        {/* Back link */}
        <div className="mb-4">
          <Link
            href="/search"
            className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors"
          >
            ← Back to search
          </Link>
        </div>

        {/* Processing / Pending state */}
        {(recordingStatus === "processing" || recordingStatus === "pending") && (
          <div
            className="w-full aspect-video rounded-xl flex items-center justify-center mb-6"
            style={{ backgroundColor: "var(--color-surface)" }}
          >
            <div className="text-center">
              <p className="text-4xl mb-3">⏳</p>
              <p className="text-lg text-[var(--color-text)] font-medium">
                Processing — available soon
              </p>
            </div>
          </div>
        )}

        {/* Failed state */}
        {recordingStatus === "failed" && (
          <div
            className="w-full aspect-video rounded-xl flex items-center justify-center mb-6"
            style={{ backgroundColor: "var(--color-surface)" }}
          >
            <div className="text-center">
              <p className="text-4xl mb-3">❌</p>
              <p className="text-lg text-[var(--color-text)] font-medium">
                This recording is unavailable
              </p>
              {vod.recordingError && (
                <p className="mt-2 text-sm text-[var(--color-text-muted)]">
                  {vod.recordingError}
                </p>
              )}
            </div>
          </div>
        )}

        {/* Ready state — player + chat */}
        {recordingStatus === "ready" && (
          <div className="flex flex-col lg:flex-row gap-6 mb-6">
            <div className="flex-1 lg:max-w-[70%] space-y-4">
              <VideoPlayer
                hlsUrl={hlsUrl}
                isLive={false}
                vodId={id}
                viewerCount={totalViewers}
              />

              {/* Info section */}
              <div className="space-y-3">
                <h1 className="text-xl font-bold text-[var(--color-text)]">
                  {title || "Untitled stream"}
                </h1>

                <div className="flex items-center gap-3">
                  <Link
                    href={`/channel/${vod.userId}`}
                    className="flex items-center gap-2 hover:opacity-80 transition-opacity"
                  >
                    {userAvatar ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img
                        src={userAvatar}
                        alt={userName}
                        className="w-8 h-8 rounded-full"
                        referrerPolicy="no-referrer"
                        onError={(e) => {
                          const target = e.target as HTMLImageElement;
                          target.onerror = null; // prevent infinite loop
                          target.src = AVATAR_FALLBACK;
                        }}
                      />
                    ) : (
                      <div
                        className="w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold"
                        style={{
                          backgroundColor: "var(--color-surface)",
                          color: "var(--color-text-muted)",
                        }}
                      >
                        {userName.charAt(0).toUpperCase()}
                      </div>
                    )}
                    <span className="text-sm font-medium text-[var(--color-text)]">
                      {userName}
                    </span>
                  </Link>
                </div>

                <div className="flex items-center gap-3 text-sm text-[var(--color-text-muted)]">
                  <span>{dateLabel}</span>
                  {durationLabel && (
                    <>
                      <span aria-hidden="true">·</span>
                      <span>{durationLabel}</span>
                    </>
                  )}
                  <span aria-hidden="true">·</span>
                  <span>{viewLabel}</span>
                </div>
              </div>
            </div>

            {/* Chat replay — desktop */}
            <aside className="hidden lg:flex lg:flex-col lg:w-[30%] lg:min-w-[300px]" style={{ minHeight: "500px" }}>
              <ChatPanel
                streamId={streamId}
                isSignedIn={false}
                isStreamEnded={true}
              />
            </aside>

            {/* Chat replay — mobile */}
            <section className="lg:hidden">
              <div style={{ minHeight: "400px", maxHeight: "60vh" }}>
                <ChatPanel
                  streamId={streamId}
                  isSignedIn={false}
                  isStreamEnded={true}
                />
              </div>
            </section>
          </div>
        )}
        {/* Fallback: unexpected status */}
        {recordingStatus !== "ready" &&
          recordingStatus !== "processing" &&
          recordingStatus !== "pending" &&
          recordingStatus !== "failed" && (
          <div
            className="w-full aspect-video rounded-xl flex items-center justify-center mb-6"
            style={{ backgroundColor: "var(--color-surface)" }}
          >
            <div className="text-center">
              <p className="text-4xl mb-3">📡</p>
              <p className="text-lg text-[var(--color-text)] font-medium">
                This recording is unavailable
              </p>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
