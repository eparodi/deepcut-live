import Link from "next/link";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

// Server-side fetch needs absolute URL; client components use relative URLs via api.ts
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:3000";

interface LiveStreamData {
  liveCount: number;
  pastStreamsCount: number;
  error: boolean;
}

async function getLiveStats(): Promise<LiveStreamData> {
  try {
    const res = await fetch(`${API_BASE_URL}/api/streams/live`, {
      cache: "no-store",
    });

    if (!res.ok) {
      return { liveCount: 0, pastStreamsCount: 0, error: true };
    }

    const streams = await res.json();
    const liveCount = Array.isArray(streams) ? streams.length : 0;

    // [Inference] pastStreamsCount would come from /api/search?q= totals.
    // That endpoint doesn't exist yet, so we show 0 gracefully.
    return { liveCount, pastStreamsCount: 0, error: false };
  } catch {
    // Endpoint may not be deployed yet — show 0 gracefully
    return { liveCount: 0, pastStreamsCount: 0, error: false };
  }
}

export default async function LandingPage() {
  // Redirect authenticated users to dashboard
  const cookieStore = await cookies();
  const token = cookieStore.get("token");
  if (token) {
    redirect("/dashboard");
  }
  const stats = await getLiveStats();

  return (
    <div className="flex flex-col flex-1">
      {/* Header / Nav */}
      <header className="w-full max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
        <Link href="/" className="text-[var(--color-primary)] font-bold text-xl">
          DeepCut
        </Link>
      </header>

      {/* Hero Section */}
      <main className="flex flex-1 flex-col items-center justify-center px-6 text-center">
        <h1 className="text-5xl md:text-7xl font-bold text-[var(--color-text)] max-w-3xl leading-tight">
          Stream What You Believe
        </h1>
        <p className="mt-6 text-lg md:text-xl text-[var(--color-text-muted)] max-w-xl">
          No censorship. No filters.
        </p>

        {/* Google Sign-In Button */}
        <a
          href={`${API_BASE_URL}/api/auth/google`}
          className="mt-10 inline-flex items-center gap-3 rounded-xl px-8 py-4 text-lg font-semibold text-white transition-colors hover:opacity-90"
          style={{ backgroundColor: "var(--color-google-blue)" }}
        >
          <GoogleIcon />
          Start Streaming with Google
        </a>

        {/* Live Stats */}
        <div className="mt-16 flex items-center gap-8 text-[var(--color-text-muted)]">
          <div className="flex items-center gap-2">
            <span className="inline-block w-2.5 h-2.5 rounded-full bg-red-500 animate-pulse" />
            <span>
              <strong className="text-[var(--color-text)]">
                {stats.liveCount.toLocaleString()}
              </strong>{" "}
              live now
            </span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-lg">📼</span>
            <span>
              <strong className="text-[var(--color-text)]">
                {stats.pastStreamsCount.toLocaleString()}
              </strong>{" "}
              past streams
            </span>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="w-full max-w-7xl mx-auto px-6 py-6 text-center text-sm text-[var(--color-text-muted)]">
        DeepCut Live &mdash; Free expression streaming platform.
      </footer>
    </div>
  );
}

/** Google "G" icon — follows Google brand guidelines */
function GoogleIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" aria-hidden="true">
      <path
        d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z"
        fill="#4285F4"
      />
      <path
        d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
        fill="#34A853"
      />
      <path
        d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
        fill="#FBBC05"
      />
      <path
        d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
        fill="#EA4335"
      />
    </svg>
  );
}
