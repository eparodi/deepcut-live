import Link from "next/link";

export default function VodNotFound() {
  return (
    <div className="min-h-full flex flex-col">
      <header
        className="w-full border-b"
        style={{ borderColor: "var(--color-surface-raised)" }}
      >
        <div className="max-w-7xl mx-auto px-6 py-4 flex items-center">
          <Link
            href="/"
            className="font-bold text-xl"
            style={{ color: "var(--color-primary)" }}
          >
            DeepCut
          </Link>
        </div>
      </header>

      <main className="flex-1 flex items-center justify-center px-6">
        <div
          className="rounded-xl p-8 text-center max-w-md"
          style={{ backgroundColor: "var(--color-surface-raised)" }}
        >
          <p className="text-5xl mb-4">📼</p>
          <h1 className="text-xl font-bold text-[var(--color-text)]">
            VOD not found
          </h1>
          <p className="mt-2 text-sm text-[var(--color-text-muted)]">
            This past stream doesn&apos;t exist or may have been removed.
          </p>
          <Link
            href="/"
            className="mt-6 inline-flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:opacity-90"
            style={{ backgroundColor: "var(--color-primary)" }}
          >
            ← Back to live streams
          </Link>
        </div>
      </main>
    </div>
  );
}
