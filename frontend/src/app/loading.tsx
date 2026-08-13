export default function RootLoading() {
  return (
    <main className="flex-1 w-full max-w-7xl mx-auto px-6 py-8">
      {/* Section heading skeleton */}
      <div className="skeleton h-6 w-32 rounded mb-6" />

      {/* Live grid skeleton (2 cols mobile, 4 desktop) */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4" role="status" aria-label="Loading live streams">
        {[0, 1, 2, 3].map((i) => (
          <div key={i} className="space-y-3">
            <div className="skeleton w-full aspect-video rounded-xl" />
            <div className="skeleton h-4 w-3/4 rounded" />
            <div className="skeleton h-3 w-1/2 rounded" />
          </div>
        ))}
      </div>
    </main>
  );
}
