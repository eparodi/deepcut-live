import { LiveGrid } from "@/components/LiveGrid";
import { VodCard } from "@/components/VodCard";
import { getLiveStreams, getRecentVods } from "@/lib/api";
import type { LiveStream, VodItem } from "@/types";

export const dynamic = "force-dynamic";

interface HomeData {
  streams: LiveStream[];
  total: number;
  recentVods: VodItem[];
  /** True when the backend could not be reached (distinct from "nothing live"). */
  loadFailed: boolean;
}

async function getHomeData(): Promise<HomeData> {
  try {
    const { streams, total } = await getLiveStreams();
    const recentVods =
      streams.length === 0 ? (await getRecentVods(8)).vods : [];
    return { streams, total, recentVods, loadFailed: false };
  } catch {
    // Distinguish backend failure from a genuinely empty platform so we
    // don't render "Nothing live right now" when the API is down.
    return { streams: [], total: 0, recentVods: [], loadFailed: true };
  }
}

export default async function HomePage() {
  const { streams, total, recentVods, loadFailed } = await getHomeData();

  return (
    <main className="flex-1 w-full max-w-7xl mx-auto px-6 py-8">
      {/* Page-level h1 must exist even when streams render (WCAG heading
          order): the hero h1 below only renders for the empty state. */}
      {streams.length > 0 && (
        <h1 className="sr-only">DeepCut Live — Browse live streams</h1>
      )}

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

      {loadFailed ? (
        <div className="mt-12 text-center" role="alert">
          <p className="text-[var(--color-text-muted)] text-sm">
            Couldn&apos;t load streams right now. Try refreshing in a moment.
          </p>
        </div>
      ) : (
        <>
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
                className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4"
                role="list"
              >
                {recentVods.map((vod) => (
                  <VodCard key={vod.id} vod={vod} />
                ))}
              </div>
            </section>
          )}
        </>
      )}
    </main>
  );
}
