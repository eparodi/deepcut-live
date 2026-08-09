export default function ChannelLoading() {
  return (
    <div className="min-h-full flex flex-col">
      {/* Header skeleton */}
      <header
        className="w-full border-b"
        style={{ borderColor: "var(--color-surface-raised)" }}
      >
        <div className="max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className="skeleton h-6 w-20 rounded" />
            <div className="skeleton h-4 w-24 rounded" />
          </div>
          <div className="skeleton h-9 w-20 rounded-lg" />
        </div>
      </header>

      <main className="flex-1 max-w-7xl mx-auto w-full px-6 py-6">
        <div className="flex flex-col lg:flex-row gap-6">
          {/* Video skeleton */}
          <div className="flex-1 lg:max-w-[70%] space-y-4">
            <div className="skeleton w-full aspect-video rounded-xl" />
            <div className="space-y-3">
              <div className="skeleton h-7 w-64 rounded" />
              <div className="flex items-center gap-3">
                <div className="skeleton h-10 w-10 rounded-full" />
                <div className="space-y-2">
                  <div className="skeleton h-4 w-32 rounded" />
                  <div className="skeleton h-3 w-20 rounded" />
                </div>
              </div>
            </div>
          </div>

          {/* Chat skeleton */}
          <aside className="hidden lg:flex lg:flex-col lg:w-[30%] lg:min-w-[300px]">
            <div
              className="flex-1 rounded-xl p-4"
              style={{ backgroundColor: "var(--color-surface-raised)" }}
            >
              <div className="skeleton h-5 w-16 rounded mb-4" />
              <div className="space-y-3">
                {[0, 1, 2, 3, 4].map((i) => (
                  <div key={i} className="flex items-start gap-2">
                    <div className="skeleton h-6 w-6 rounded-full shrink-0" />
                    <div className="space-y-1 flex-1">
                      <div className="skeleton h-3 w-20 rounded" />
                      <div className="skeleton h-3 w-full rounded" />
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </aside>
        </div>
      </main>
    </div>
  );
}
