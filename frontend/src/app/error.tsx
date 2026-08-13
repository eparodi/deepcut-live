"use client";
// Client Component — error boundaries must be Client Components

import Link from "next/link";

export default function RootError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <main className="flex-1 flex items-center justify-center px-6 py-16">
      <div
        className="rounded-xl p-8 text-center max-w-md"
        style={{ backgroundColor: "var(--color-surface-raised)" }}
      >
        <h1 className="text-lg font-semibold text-[var(--color-text)]">
          Something went wrong
        </h1>
        <p className="mt-2 text-sm text-[var(--color-text-muted)]">
          An unexpected error occurred. Try again in a moment.
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
            Retry
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
  );
}
