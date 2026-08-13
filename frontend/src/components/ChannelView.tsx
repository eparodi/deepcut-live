"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { getChannel } from "@/lib/api";
import { VideoPlayer } from "@/components/VideoPlayer";
import { StreamInfo } from "@/components/StreamInfo";
import { ChatPanel } from "@/components/ChatPanel";
import type { ChannelResponse } from "@/types";

interface Props {
  id: string;
  initialChannel: ChannelResponse;
  isSignedIn: boolean;
}

export function ChannelView({ id, initialChannel, isSignedIn }: Props) {
  const [channel, setChannel] = useState(initialChannel);
  const [isTheaterMode, setIsTheaterMode] = useState(false);

  useEffect(() => {
    const timer = setInterval(async () => {
      try {
        const fresh = await getChannel(id);
        setChannel(fresh);
      } catch {
        // keep stale data on error
      }
    }, 10_000);
    return () => clearInterval(timer);
  }, [id]);

  const hlsUrl = channel.hlsUrl;

  return (
    <div className="min-h-full flex flex-col">
      <main className="flex-1 max-w-7xl mx-auto w-full px-6 py-6">
        <div className="mb-4">
          <Link
            href="/"
            className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors"
          >
            ← Browse streams
          </Link>
        </div>

        <div className={`flex flex-col gap-6 ${isTheaterMode ? "" : "lg:flex-row"}`}>
          <div className={`flex-1 space-y-4 ${isTheaterMode ? "" : "lg:max-w-[70%]"}`}>
            {hlsUrl ? (
              <VideoPlayer
                hlsUrl={hlsUrl}
                isLive={channel.isLive}
                viewerCount={channel.viewerCount}
                onTheaterChange={setIsTheaterMode}
              />
            ) : (
              <div
                className="w-full aspect-video rounded-xl flex items-center justify-center"
                style={{ backgroundColor: "var(--color-surface)" }}
              >
                <div className="text-center">
                  <p className="text-4xl mb-3">📡</p>
                  <p className="text-lg text-[var(--color-text)] font-medium">
                    {channel.isLive
                      ? "Waiting for stream..."
                      : "Stream is offline"}
                  </p>
                  <p className="mt-2 text-sm text-[var(--color-text-muted)]">
                    {channel.isLive
                      ? "The streamer hasn't started broadcasting yet"
                      : "Check out past streams below"}
                  </p>
                </div>
              </div>
            )}

            <StreamInfo channel={channel} />

            <div className="pt-2">
              <Link
                href={`/search?userId=${channel.userId}`}
                className="text-sm font-medium hover:underline"
                style={{ color: "var(--color-primary-text)" }}
              >
                📼 View past streams →
              </Link>
            </div>
          </div>

          <aside
            className={`hidden lg:flex lg:flex-col ${isTheaterMode ? "lg:w-full" : "lg:w-[30%] lg:min-w-[300px]"}`}
            style={{ minHeight: isTheaterMode ? "300px" : "500px" }}
          >
            {channel.streamId ? (
              <ChatPanel
                streamId={channel.streamId}
                isSignedIn={isSignedIn}
                isStreamEnded={!channel.isLive}
              />
            ) : (
              <div
                className="flex-1 rounded-xl flex items-center justify-center"
                style={{ backgroundColor: "var(--color-surface-raised)" }}
              >
                <p className="text-sm text-[var(--color-text-muted)] text-center px-4">
                  Chat unavailable for this channel
                </p>
              </div>
            )}
          </aside>

          <section className="lg:hidden">
            {channel.streamId ? (
              <div style={{ minHeight: "400px", maxHeight: "60vh" }}>
                <ChatPanel
                  streamId={channel.streamId}
                  isSignedIn={isSignedIn}
                  isStreamEnded={!channel.isLive}
                />
              </div>
            ) : (
              <div
                className="rounded-xl p-6 text-center"
                style={{ backgroundColor: "var(--color-surface-raised)" }}
              >
                <p className="text-sm text-[var(--color-text-muted)]">
                  Chat unavailable for this channel
                </p>
              </div>
            )}
          </section>
        </div>
      </main>
    </div>
  );
}
