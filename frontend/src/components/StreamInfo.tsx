import type { ChannelResponse } from "@/types";

interface StreamInfoProps {
  channel: ChannelResponse;
}

/** Format a viewer count for display */
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

export function StreamInfo({ channel }: StreamInfoProps) {
  const {
    streamerName,
    streamerAvatarUrl,
    streamTitle,
    streamCategory,
    isLive,
    viewerCount,
  } = channel;

  return (
    <div className="space-y-3">
      {/* Stream title */}
      <h1 className="text-xl font-bold text-[var(--color-text)]">
        {streamTitle || "Untitled stream"}
      </h1>

      {/* Streamer info row */}
      <div className="flex items-center gap-3">
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src={streamerAvatarUrl}
          alt={streamerName}
          className="w-10 h-10 rounded-full"
          referrerPolicy="no-referrer"
        />
        <div>
          <p className="text-sm font-semibold text-[var(--color-text)]">
            {streamerName}
          </p>
          <div className="flex items-center gap-2 text-sm text-[var(--color-text-muted)]">
            {isLive ? (
              <>
                <span
                  className="inline-block w-2 h-2 rounded-full animate-pulse"
                  style={{ backgroundColor: "var(--color-danger)" }}
                />
                <span>{formatViewerCount(viewerCount)} viewers</span>
              </>
            ) : (
              <span>Offline</span>
            )}
          </div>
        </div>
      </div>

      {/* Category pill */}
      {streamCategory && (
        <span
          className="inline-block rounded-full px-3 py-1 text-xs font-medium"
          style={{
            backgroundColor: "var(--color-surface-raised)",
            color: "var(--color-primary)",
          }}
        >
          {streamCategory}
        </span>
      )}
    </div>
  );
}
