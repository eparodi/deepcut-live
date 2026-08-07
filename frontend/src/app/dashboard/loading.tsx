export default function DashboardLoading() {
  return (
    <div className="min-h-full flex flex-col">
      {/* Header skeleton */}
      <header
        className="w-full border-b"
        style={{ borderColor: "var(--color-surface-raised)" }}
      >
        <div className="max-w-5xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="skeleton h-6 w-24 rounded" />
          <div className="skeleton h-8 w-8 rounded-full" />
        </div>
      </header>

      <main className="flex-1 max-w-3xl mx-auto w-full px-6 py-8">
        <div className="space-y-8 animate-pulse">
          {/* Stream key skeleton */}
          <section>
            <div className="skeleton h-5 w-24 mb-4" />
            <div className="skeleton h-14 w-full rounded-lg" />
          </section>

          <div className="skeleton h-10 w-40 rounded-lg" />

          <hr
            className="border-0 h-px"
            style={{ backgroundColor: "var(--color-surface-raised)" }}
          />

          {/* Settings skeleton */}
          <section>
            <div className="skeleton h-5 w-32 mb-4" />
            <div className="space-y-4">
              <div className="skeleton h-12 w-full rounded-lg" />
              <div className="skeleton h-12 w-full rounded-lg" />
              <div className="skeleton h-10 w-20 rounded-lg" />
            </div>
          </section>

          <hr
            className="border-0 h-px"
            style={{ backgroundColor: "var(--color-surface-raised)" }}
          />

          {/* Analytics skeleton */}
          <section>
            <div className="skeleton h-5 w-40 mb-4" />
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
              {[0, 1, 2, 3].map((i) => (
                <div key={i} className="skeleton h-24 rounded-xl" />
              ))}
            </div>
          </section>
        </div>
      </main>
    </div>
  );
}
