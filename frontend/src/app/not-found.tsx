import Link from "next/link";

export default function RootNotFound() {
  return (
    <main className="flex-1 flex items-center justify-center px-6 py-16">
      <div
        className="rounded-xl p-8 text-center max-w-md"
        style={{ backgroundColor: "var(--color-surface-raised)" }}
      >
        <p className="text-5xl mb-4">🔍</p>
        <h1 className="text-xl font-bold text-[var(--color-text)]">
          Page not found
        </h1>
        <p className="mt-2 text-sm text-[var(--color-text-muted)]">
          The page you&apos;re looking for doesn&apos;t exist.
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
  );
}
