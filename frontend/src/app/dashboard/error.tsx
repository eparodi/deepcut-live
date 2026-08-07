"use client";
// Client Component — needs onClick for retry

import Link from "next/link";

export default function DashboardError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="min-h-full flex flex-col">
      {/* Header */}
      <header
        className="w-full border-b"
        style={{ borderColor: "var(--color-surface-raised)" }}
      >
        <div className="max-w-5xl mx-auto px-6 py-4 flex items-center justify-between">
          <Link
            href="/"
            className="font-bold text-xl"
            style={{ color: "var(--color-primary)" }}
          >
            DeepCut
          </Link>
        </div>
      </header>

      <main className="flex-1 max-w-3xl mx-auto w-full px-6 py-8 flex items-center justify-center">
        <div
          className="rounded-xl p-8 text-center max-w-md"
          style={{ backgroundColor: "var(--color-surface-raised)" }}
        >
          <h2 className="text-lg font-semibold text-[var(--color-text)]">
            Something went wrong
          </h2>
          <p className="mt-2 text-sm text-[var(--color-text-muted)]">
            An unexpected error occurred while loading your dashboard.
          </p>
          {error.digest && (
            <p className="mt-1 text-xs text-[var(--color-text-muted)] font-mono">
              Error ID: {error.digest}
            </p>
          )}
          <div className="mt-6 flex gap-3 justify-center">
            <button
              onClick={reset}
              className="rounded-lg px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:opacity-90"
              style={{ backgroundColor: "var(--color-primary)" }}
            >
              Try again
            </button>
            <Link
              href="/"
              className="rounded-lg px-5 py-2.5 text-sm font-medium transition-colors hover:opacity-80"
              style={{
                backgroundColor: "var(--color-surface)",
                color: "var(--color-text)",
              }}
            >
              Go home
            </Link>
          </div>
        </div>
      </main>
    </div>
  );
}
