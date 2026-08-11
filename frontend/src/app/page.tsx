import { LiveGrid } from "@/components/LiveGrid";
import { VodCard } from "@/components/VodCard";
import type { LiveStream, VodItem } from "@/types";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:3000";

async function getLiveStreams(): Promise<{ streams: LiveStream[]; total: number }> {
  try {
    const res = await fetch(`${API_BASE_URL}/api/streams/live`, {
      cache: "no-store",
    });

    if (!res.ok) {
      return { streams: [], total: 0 };
    }

    const data = await res.json();

    // Handle both array and { streams, total } response shapes
    if (Array.isArray(data)) {
      return { streams: data, total: data.length };
    }
    return {
      streams: data.streams ?? [],
      total: data.total ?? 0,
    };
  } catch {
    return { streams: [], total: 0 };
  }
}

async function getRecentVods(limit = 8): Promise<VodItem[]> {
  try {
    const res = await fetch(
      `${API_BASE_URL}/api/vods?sort=recent&limit=${limit}&offset=0`,
      { cache: "no-store" }
    );

    if (!res.ok) {
      return [];
    }

    const data = await res.json();
    return data.vods ?? [];
  } catch {
    return [];
  }
}

export default async function HomePage() {
  const { streams, total } = await getLiveStreams();
  const recentVods = streams.length === 0 ? await getRecentVods(8) : [];

  return (
    <main className="flex-1 w-full max-w-7xl mx-auto px-6 py-8">
      {/* Hero — shown only when no live streams */}
      {streams.length === 0 && (
        <div className="text-center mb-12 mt-8">
          <h1 className="text-4xl md:text-5xl font-bold text-[var(--color-text)]">
            Stream What You Believe
          </h1>
          <p className="mt-4 text-lg text-[var(--color-text-muted)] max-w-lg mx-auto">
            No censorship. No filters. A live streaming platform for free expression.
          </p>
        </div>
      )}

      <LiveGrid streams={streams} total={total} />

      {/* Empty state when no live streams */}
      {streams.length === 0 && recentVods.length === 0 && (
        <div className="mt-12 text-center">
          <p className="text-[var(--color-text-muted)] text-sm">
            Nothing live right now. Be the first to go live!
          </p>
        </div>
      )}

      {/* Recent Past Streams — shown only when no live streams but VODs exist */}
      {streams.length === 0 && recentVods.length > 0 && (
        <section className="mt-12">
          <h2 className="text-lg font-semibold text-[var(--color-text)] mb-4">
            📼 Recent Past Streams
          </h2>
          <div
            className="grid grid-cols-2 lg:grid-cols-4 gap-4"
            role="list"
          >
            {recentVods.map((vod) => (
              <VodCard key={vod.id} vod={vod} />
            ))}
          </div>
        </section>
      )}
    </main>
  );
}
