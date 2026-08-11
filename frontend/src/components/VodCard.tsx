"use client";
// Client Component — needs onClick for navigation (Link)

import Link from "next/link";
import type { VodItem } from "@/types";

interface VodCardProps {
  vod: VodItem;
}

/** Format duration in seconds to human-readable: 3661 → "1h 1m", 2700 → "45m", 30 → "30s" */
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

/** Format an ISO date to a relative time string: "2 days ago", "just now" */
function formatRelativeTime(isoDate: string): string {
  const now = Date.now();
  const then = new Date(isoDate).getTime();
  const diffMs = now - then;
  const diffSeconds = Math.floor(diffMs / 1000);

  if (diffSeconds < 60) return "just now";
  const diffMinutes = Math.floor(diffSeconds / 60);
  if (diffMinutes < 60) return `${diffMinutes}m ago`;
  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) return `${diffHours}h ago`;
  const diffDays = Math.floor(diffHours / 24);
  if (diffDays < 7) return `${diffDays}d ago`;
  const diffWeeks = Math.floor(diffDays / 7);
  if (diffWeeks < 4) return `${diffWeeks}w ago`;
  const diffMonths = Math.floor(diffDays / 30);
  return `${diffMonths}mo ago`;
}

export function VodCard({ vod }: VodCardProps) {
  const { id, userName, userAvatar, title, durationSeconds, startedAt, recordingStatus } = vod;

  const statusLabel =
    recordingStatus === "processing"
      ? "Processing"
      : recordingStatus === "failed"
        ? "Unavailable"
        : null;

  const durationLabel = durationSeconds != null ? formatDuration(durationSeconds) : null;
  const relativeDate = formatRelativeTime(startedAt);

  return (
    <Link
      href={`/vods/${id}`}
      role="listitem"
      aria-label={`Past stream: ${title || "Untitled stream"} by ${userName}. ${durationLabel ? durationLabel + ", " : ""}streamed ${relativeDate}.`}
      className="group block rounded-xl overflow-hidden transition-all duration-300 hover:scale-[1.02] hover:shadow-lg hover:shadow-[var(--color-primary)]/10 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
      style={{ backgroundColor: "var(--color-surface-raised)" }}
    >
      {/* Thumbnail area */}
      <div className="relative aspect-video bg-[var(--color-surface)]">
        <div className="w-full h-full flex items-center justify-center">
          <span
            className="text-4xl opacity-20"
            role="img"
            aria-label="No thumbnail"
          >
            🎬
          </span>
        </div>

        {/* Duration badge — bottom left */}
        {durationLabel && (
          <div
            className="absolute bottom-2 left-2 rounded-md px-2 py-0.5 text-xs font-medium"
            style={{
              backgroundColor: "rgba(0,0,0,0.75)",
              color: "var(--color-text)",
            }}
          >
            {durationLabel}
          </div>
        )}

        {/* Status badge — top right (only for non-ready) */}
        {statusLabel && (
          <div
            className="absolute top-2 right-2 rounded-md px-2 py-0.5 text-xs font-medium text-white"
            style={{
              backgroundColor:
                recordingStatus === "failed"
                  ? "var(--color-danger)"
                  : "var(--color-primary)",
            }}
            aria-label={statusLabel}
          >
            {statusLabel}
          </div>
        )}
      </div>

      {/* Info area */}
      <div className="p-3 space-y-2">
        {/* Streamer avatar + name */}
        <div className="flex items-center gap-2">
          {userAvatar ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={userAvatar}
              alt={userName}
              className="w-6 h-6 rounded-full flex-shrink-0"
              referrerPolicy="no-referrer"
            />
          ) : (
            <div
              className="w-6 h-6 rounded-full flex-shrink-0 flex items-center justify-center text-xs font-bold"
              style={{
                backgroundColor: "var(--color-surface)",
                color: "var(--color-text-muted)",
              }}
            >
              {userName.charAt(0).toUpperCase()}
            </div>
          )}
          <span className="text-sm font-medium text-[var(--color-text)] truncate">
            {userName}
          </span>
        </div>

        {/* Title — truncated to 2 lines */}
        <p
          className="text-sm text-[var(--color-text-muted)] line-clamp-2 leading-snug"
        >
          {title || "Untitled stream"}
        </p>

        {/* Date */}
        <p className="text-xs text-[var(--color-text-muted)]">
          {relativeDate}
        </p>
      </div>
    </Link>
  );
}
