import { notFound } from "next/navigation";
import Link from "next/link";
import { cookies } from "next/headers";
import { getChannel } from "@/lib/api";
import { VideoPlayer } from "@/components/VideoPlayer";
import { StreamInfo } from "@/components/StreamInfo";
import { ChatPanel } from "@/components/ChatPanel";
import type { Metadata } from "next";

export const dynamic = "force-dynamic";

interface ChannelPageProps {
  params: Promise<{ id: string }>;
}

export async function generateMetadata({
  params,
}: ChannelPageProps): Promise<Metadata> {
  const { id } = await params;

  try {
    const channel = await getChannel(id);
    return {
      title: `${channel.streamerName} — ${channel.streamTitle || "Live Stream"}`,
      description: `Watch ${channel.streamerName} live on DeepCut`,
    };
  } catch {
    return {
      title: "Channel not found — DeepCut",
    };
  }
}

export default async function ChannelPage({ params }: ChannelPageProps) {
  const { id } = await params;

  // Check auth status
  const cookieStore = await cookies();
  const token = cookieStore.get("token");
  const isSignedIn = !!token;

  let channel;
  try {
    channel = await getChannel(id);
  } catch (error: unknown) {
    if (
      error instanceof Error &&
      "status" in error &&
      (error as { status: number }).status === 404
    ) {
      notFound();
    }
    throw error;
  }

  const apiBaseUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

  return (
    <div className="min-h-full flex flex-col">
      {/* Header */}
      <header
        className="w-full border-b"
        style={{ borderColor: "var(--color-surface-raised)" }}
      >
        <div className="max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link
              href="/"
              className="font-bold text-xl"
              style={{ color: "var(--color-primary)" }}
            >
              DeepCut
            </Link>
            <Link
              href="/"
              className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors"
            >
              ← Back to Live
            </Link>
          </div>
          {isSignedIn ? (
            <Link
              href="/dashboard"
              className="rounded-lg px-4 py-2 text-sm font-medium transition-colors hover:opacity-80"
              style={{
                backgroundColor: "var(--color-surface-raised)",
                color: "var(--color-text)",
              }}
            >
              Dashboard
            </Link>
          ) : (
            <a
              href={`${apiBaseUrl}/api/auth/google`}
              className="rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors hover:opacity-90"
              style={{ backgroundColor: "var(--color-google-blue)" }}
            >
              Sign In
            </a>
          )}
        </div>
      </header>

      {/* Main content: video + chat side-by-side on desktop */}
      <main className="flex-1 max-w-7xl mx-auto w-full px-6 py-6">
        <div className="flex flex-col lg:flex-row gap-6">
          {/* Video + Stream Info column */}
          <div className="flex-1 lg:max-w-[70%] space-y-4">
            {channel.hlsUrl ? (
              <VideoPlayer
                hlsUrl={channel.hlsUrl}
                isLive={channel.isLive}
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

            {/* Past VODs link */}
            <div className="pt-2">
              <Link
                href={`/search?q=${encodeURIComponent(channel.streamerName)}`}
                className="text-sm font-medium hover:underline"
                style={{ color: "var(--color-primary)" }}
              >
                📼 View past streams →
              </Link>
            </div>
          </div>

          {/* Chat panel — desktop side panel */}
          <aside
            className="hidden lg:flex lg:flex-col lg:w-[30%] lg:min-w-[300px]"
            style={{ minHeight: "500px" }}
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

          {/* Mobile chat — below video on small screens */}
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
