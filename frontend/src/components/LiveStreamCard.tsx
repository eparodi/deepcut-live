"use client";
// Client Component — needs img onError fallback handlers

import Link from "next/link";
import type { LiveStream } from "@/types";
import { AVATAR_FALLBACK, THUMBNAIL_FALLBACK } from "@/lib/fallbacks";
import { formatViewerCount } from "@/lib/format";

interface LiveStreamCardProps {
  stream: LiveStream;
  /** Whether this is a new stream that should fade in */
  isNew?: boolean;
  /** Layout variant: full card (grid) or compact row (list) */
  variant?: "grid" | "list";
}

export function LiveStreamCard({
  stream,
  isNew = false,
  variant = "grid",
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

  // Compact list row: small thumbnail + text, one stream per row.
  if (variant === "list") {
    return (
      <Link
        href={`/channel/${userId}`}
        role="listitem"
        aria-label={`${streamerName} is live: ${title}. ${viewerCount} viewers`}
        className={`group flex items-center gap-3 rounded-xl p-3 transition-colors hover:bg-[var(--color-surface)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] ${
          isNew ? "animate-fade-up" : ""
        }`}
        style={{ backgroundColor: "var(--color-surface-raised)" }}
      >
        {/* Thumbnail */}
        <div className="relative w-32 aspect-video shrink-0 rounded-lg overflow-hidden bg-[var(--color-surface)]">
          {thumbnailUrl ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={thumbnailUrl}
              alt={`${streamerName}'s stream thumbnail`}
              className="w-full h-full object-cover"
              loading="lazy"
              onError={(e) => {
                const target = e.target as HTMLImageElement;
                target.onerror = null;
                target.src = THUMBNAIL_FALLBACK;
              }}
            />
          ) : (
            <div className="w-full h-full flex items-center justify-center">
              <span
                className="text-2xl opacity-20"
                role="img"
                aria-label="No thumbnail"
              >
                🎬
              </span>
            </div>
          )}
          <div
            className="absolute top-1.5 left-1.5 flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] font-bold text-white"
            style={{ backgroundColor: "var(--color-danger)" }}
            aria-label="Live"
          >
            <span className="inline-block w-1 h-1 rounded-full bg-white animate-pulse" />
            LIVE
          </div>
        </div>

        {/* Text */}
        <div className="min-w-0 flex-1 space-y-1">
          <div className="flex items-center gap-2">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={streamerAvatarUrl}
              alt={streamerName}
              className="w-5 h-5 rounded-full flex-shrink-0"
              referrerPolicy="no-referrer"
              onError={(e) => {
                const target = e.target as HTMLImageElement;
                target.onerror = null;
                target.src = AVATAR_FALLBACK;
              }}
            />
            <span className="text-sm font-medium text-[var(--color-text)] truncate">
              {streamerName}
            </span>
          </div>
          <p className="text-sm text-[var(--color-text-muted)] truncate">
            {title || "Untitled stream"}
          </p>
          <div className="flex items-center gap-2 flex-wrap">
            {category && (
              <span
                className="inline-block rounded-full px-2.5 py-0.5 text-xs font-medium"
                style={{
                  backgroundColor: "var(--color-surface)",
                  color: "var(--color-primary-text)",
                }}
              >
                {category}
              </span>
            )}
            <span className="text-xs text-[var(--color-text-muted)]">
              <span role="img" aria-label="viewers">
                👁
              </span>{" "}
              {formatViewerCount(viewerCount)}
            </span>
          </div>
        </div>
      </Link>
    );
  }

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
        {thumbnailUrl && (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={thumbnailUrl}
            alt={`${streamerName}'s stream thumbnail`}
            className="w-full h-full object-cover"
            loading="lazy"
            onError={(e) => {
              const target = e.target as HTMLImageElement;
              target.onerror = null;
              target.src = THUMBNAIL_FALLBACK;
            }}
          />
        )}
        {!thumbnailUrl && (
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
            onError={(e) => {
              const target = e.target as HTMLImageElement;
              target.onerror = null;
              target.src = AVATAR_FALLBACK;
            }}
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
              color: "var(--color-primary-text)",
            }}
          >
            {category}
          </span>
        )}
      </div>
    </Link>
  );
}
