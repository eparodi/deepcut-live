"use client";
// Client Component — fetches channel info to show live preview + stream link.

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { getChannel } from "@/lib/api";
import type { ChannelResponse } from "@/types";
import { VideoPlayer } from "@/components/VideoPlayer";

interface GoLivePreviewProps {
  userId: string;
  isLive: boolean;
}

export function GoLivePreview({ userId, isLive }: GoLivePreviewProps) {
  const [channel, setChannel] = useState<ChannelResponse | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchChannel = useCallback(() => {
    setLoading(true);
    getChannel(userId)
      .then(setChannel)
      .catch(() => setChannel(null))
      .finally(() => setLoading(false));
  }, [userId]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- fetch-on-mount is intentional
    fetchChannel();
  }, [fetchChannel]);

  // Poll for updates when the channel is live (every 10s)
  useEffect(() => {
    if (!isLive) return;
    const interval = setInterval(fetchChannel, 10_000);
    return () => clearInterval(interval);
  }, [isLive, fetchChannel]);

  const hlsUrl = channel?.hlsUrl ?? null;

  return (
    <section aria-labelledby="go-live-heading">
      <h2
        id="go-live-heading"
        className="text-lg font-semibold text-[var(--color-text)] mb-4"
      >
        Stream Preview
      </h2>

      {loading ? (
        <div
          className="w-full aspect-video rounded-xl skeleton"
        />
      ) : isLive && hlsUrl ? (
        <div className="space-y-4">
          {/* Live badge + viewer count */}
          <div className="flex items-center gap-2">
            <span
              className="inline-flex items-center gap-1.5 rounded-md px-2.5 py-0.5 text-xs font-bold text-white"
              style={{ backgroundColor: "var(--color-danger)" }}
            >
              <span className="inline-block w-1.5 h-1.5 rounded-full bg-white animate-pulse" />
              LIVE
            </span>
            {channel && (
              <span className="text-sm text-[var(--color-text-muted)]">
                {channel.viewerCount} viewer{channel.viewerCount !== 1 ? "s" : ""}
              </span>
            )}
          </div>

          {/* Video preview */}
          <VideoPlayer hlsUrl={hlsUrl} isLive={true} />

          {/* Channel link button */}
          <Link
            href={`/channel/${userId}`}
            className="inline-flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:opacity-90"
            style={{ backgroundColor: "var(--color-primary)" }}
          >
            📺 View My Stream
          </Link>

          {/* Stream key reassurance */}
          <p className="text-xs text-[var(--color-text-muted)]">
            Your stream key is the same one shown above. It stays the same
            until you choose to regenerate it — so you can reuse it every
            streaming session.
          </p>
        </div>
      ) : (
        <div
          className="rounded-xl p-6 text-center space-y-3"
          style={{ backgroundColor: "var(--color-surface)" }}
        >
          <p className="text-4xl">📡</p>
          <h3 className="text-base font-semibold text-[var(--color-text)]">
            Not streaming yet
          </h3>
          <p className="text-sm text-[var(--color-text-muted)] max-w-sm mx-auto">
            Start streaming from OBS using your stream key above. Once
            you&apos;re live, a preview will appear here and you can view
            your stream with chat.
          </p>

          {/* Stream key reuse note */}
          <p className="text-xs text-[var(--color-text-muted)] mt-2">
            Your stream key stays the same across sessions — no need to
            reconfigure OBS each time.
          </p>

          {/* Channel link (always available even when offline) */}
          <Link
            href={`/channel/${userId}`}
            className="inline-flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:opacity-90"
            style={{ backgroundColor: "var(--color-primary)" }}
          >
            📺 View My Channel
          </Link>
        </div>
      )}
    </section>
  );
}
