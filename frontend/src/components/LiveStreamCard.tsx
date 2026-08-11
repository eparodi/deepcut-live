"use client";
// Client Component — needs onClick for navigation

import { useState } from "react";
import Link from "next/link";
import type { LiveStream } from "@/types";

interface LiveStreamCardProps {
  stream: LiveStream;
  /** Called when the card begins to fade out (stream ended) */
  onStreamEnded?: (userId: string) => void;
  /** Whether this is a new stream that should fade in */
  isNew?: boolean;
}

/** Format a viewer count for display: 1205 → "1.2k" */
function formatViewerCount(n: number): string {
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

export function LiveStreamCard({
  stream,
  isNew = false,
}: LiveStreamCardProps) {
  const {
    userId,
    streamerName,
    streamerAvatarUrl,
    title,
    category,
    viewerCount,
    thumbnailUrl,
  } = stream;

  const [imgError, setImgError] = useState(false);
  const showThumbnail = thumbnailUrl && !imgError;

  return (
    <Link
      href={`/channel/${userId}`}
      role="listitem"
      aria-label={`${streamerName} is live: ${title}. ${viewerCount} viewers`}
      className={`group block rounded-xl overflow-hidden transition-all duration-300 hover:scale-[1.02] hover:shadow-lg hover:shadow-[var(--color-primary)]/10 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] ${
        isNew ? "animate-fade-up" : ""
      }`}
      style={{ backgroundColor: "var(--color-surface-raised)" }}
    >
      {/* Thumbnail area */}
      <div className="relative aspect-video bg-[var(--color-surface)]">
        {showThumbnail && (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={thumbnailUrl}
            alt={`${streamerName}'s stream thumbnail`}
            className="w-full h-full object-cover"
            loading="lazy"
            onError={() => setImgError(true)}
          />
        )}
        {!showThumbnail && (
          <div className="w-full h-full flex items-center justify-center">
            <span
              className="text-4xl opacity-20"
              role="img"
              aria-label="No thumbnail"
            >
              🎬
            </span>
          </div>
        )}

        {/* Live indicator — top left */}
        <div
          className="absolute top-2 left-2 flex items-center gap-1.5 rounded-md px-2 py-0.5 text-xs font-bold text-white"
          style={{ backgroundColor: "var(--color-danger)" }}
          aria-label="Live"
        >
          <span className="inline-block w-1.5 h-1.5 rounded-full bg-white animate-pulse" />
          LIVE
        </div>

        {/* Viewer count — bottom right */}
        <div
          className="absolute bottom-2 right-2 flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium"
          style={{
            backgroundColor: "rgba(0,0,0,0.75)",
            color: "var(--color-text)",
          }}
        >
          <span role="img" aria-label="viewers">
            👁
          </span>
          {formatViewerCount(viewerCount)}
        </div>
      </div>

      {/* Info area */}
      <div className="p-3 space-y-2">
        {/* Streamer avatar + name */}
        <div className="flex items-center gap-2">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={streamerAvatarUrl}
            alt={streamerName}
            className="w-6 h-6 rounded-full flex-shrink-0"
            referrerPolicy="no-referrer"
          />
          <span className="text-sm font-medium text-[var(--color-text)] truncate">
            {streamerName}
          </span>
        </div>

        {/* Title — truncated to 2 lines */}
        <p
          className="text-sm text-[var(--color-text-muted)] line-clamp-2 leading-snug"
          style={{ fontSize: "var(--text-sm)" }}
        >
          {title || "Untitled stream"}
        </p>

        {/* Category pill */}
        {category && (
          <span
            className="inline-block rounded-full px-2.5 py-0.5 text-xs font-medium"
            style={{
              backgroundColor: "var(--color-surface)",
              color: "var(--color-primary)",
            }}
          >
            {category}
          </span>
        )}
      </div>
    </Link>
  );
}
