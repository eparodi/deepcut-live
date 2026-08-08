import Link from "next/link";
import { cookies } from "next/headers";
import { getLiveStreams } from "@/lib/api";
import { LiveGrid } from "@/components/LiveGrid";
import type { LiveStreamsResponse } from "@/types";

export const dynamic = "force-dynamic";

async function getLiveData(): Promise<LiveStreamsResponse> {
  try {
    return await getLiveStreams();
  } catch {
    return { streams: [], total: 0 };
  }
}

export default async function HomePage() {
  const cookieStore = await cookies();
  const token = cookieStore.get("token");
  const isAuthenticated = !!token;

  const liveData = await getLiveData();

  return (
    <div className="flex flex-col flex-1">
      {/* Header / Nav */}
      <header className="w-full max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-6">
          <Link href="/" className="text-[var(--color-primary)] font-bold text-xl">
            DeepCut
          </Link>
          <Link
            href="/search"
            className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors hidden sm:inline"
          >
            🔍 Search past streams
          </Link>
        </div>

        <div className="flex items-center gap-3">
          {isAuthenticated ? (
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
              href={`${process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081"}/api/auth/google`}
              className="rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors hover:opacity-90"
              style={{ backgroundColor: "var(--color-google-blue)" }}
            >
              Sign In
            </a>
          )}
        </div>
      </header>

      {/* Unauthenticated CTA banner */}
      {!isAuthenticated && (
        <div className="w-full max-w-7xl mx-auto px-6 pb-2">
          <div
            className="rounded-xl px-6 py-4 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3"
            style={{
              background: `linear-gradient(135deg, var(--color-primary), #772CE8)`,
            }}
          >
            <div>
              <p className="text-white font-semibold">
                🎬 Stream What You Believe
              </p>
              <p className="text-white/80 text-sm">
                No censorship. No filters. Start your own stream today.
              </p>
            </div>
            <a
              href={`${process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081"}/api/auth/google`}
              className="shrink-0 rounded-lg px-5 py-2.5 text-sm font-semibold text-[var(--color-primary)] bg-white transition-colors hover:bg-white/90"
            >
              Start Streaming with Google
            </a>
          </div>
        </div>
      )}

      {/* Main: Live Grid */}
      <main className="flex-1 max-w-7xl mx-auto w-full px-6 py-6">
        <LiveGrid streams={liveData.streams} total={liveData.total} />
      </main>

      {/* Footer */}
      <footer className="w-full max-w-7xl mx-auto px-6 py-6 text-center text-sm text-[var(--color-text-muted)]">
        DeepCut Live &mdash; Free expression streaming platform.
      </footer>
    </div>
  );
}
