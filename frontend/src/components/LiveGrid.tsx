"use client";
// Client Component — uses useState for grid/list toggle, useSearchParams for sort

import { useState, useMemo } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { LiveStreamCard } from "./LiveStreamCard";
import type { LiveStream } from "@/types";

interface LiveGridProps {
  streams: LiveStream[];
  total: number;
}

type ViewMode = "grid" | "list";
type SortMode = "viewers" | "recent";

export function LiveGrid({ streams, total }: LiveGridProps) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [viewMode, setViewMode] = useState<ViewMode>("grid");

  // Read sort from URL query param, default to viewers
  const sortParam = searchParams.get("sort");
  const sortMode: SortMode = sortParam === "recent" ? "recent" : "viewers";

  // Sort streams client-side
  const sortedStreams = useMemo(() => {
    const sorted = [...streams];
    if (sortMode === "recent") {
      sorted.sort(
        (a, b) =>
          new Date(b.startedAt).getTime() - new Date(a.startedAt).getTime()
      );
    } else {
      sorted.sort((a, b) => b.viewerCount - a.viewerCount);
    }
    return sorted;
  }, [streams, sortMode]);

  // Update URL when sort changes
  const handleSortChange = (mode: SortMode) => {
    const params = new URLSearchParams(searchParams.toString());
    if (mode === "viewers") {
      params.delete("sort");
    } else {
      params.set("sort", mode);
    }
    const query = params.toString();
    router.replace(query ? `/?${query}` : "/", { scroll: false });
  };

  // Empty state
  if (streams.length === 0) {
    return (
      <section aria-label="Live streams" className="py-12">
        <h2 className="text-lg font-semibold text-[var(--color-text)] mb-2">
          🔴 LIVE NOW
        </h2>
        <div
          className="rounded-xl p-10 text-center"
          style={{ backgroundColor: "var(--color-surface-raised)" }}
        >
          <p className="text-3xl mb-3" role="img" aria-label="No live streams">
            🎬
          </p>
          <p className="text-lg text-[var(--color-text)] font-medium">
            No one is live right now
          </p>
          <p className="mt-2 text-sm text-[var(--color-text-muted)]">
            Check out past streams below
          </p>
          <a
            href="/search"
            className="mt-4 inline-flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:opacity-90"
            style={{ backgroundColor: "var(--color-primary)" }}
          >
            Browse past streams
          </a>
        </div>
      </section>
    );
  }

  // Grid columns: 4 on desktop (lg), 2 on mobile
  const gridClass =
    viewMode === "grid"
      ? "grid grid-cols-2 lg:grid-cols-4 gap-4"
      : "flex flex-col gap-3";

  return (
    <section aria-label="Live streams">
      {/* Section header */}
      <div className="flex items-center flex-wrap justify-between mb-4 gap-2">
        <h2 className="text-lg font-semibold text-[var(--color-text)]">
          <span
            className="inline-block w-2 h-2 rounded-full mr-2 animate-pulse"
            style={{ backgroundColor: "var(--color-danger)" }}
          />
          LIVE NOW
          <span className="ml-2 text-sm font-normal text-[var(--color-text-muted)]">
            {total} stream{total !== 1 ? "s" : ""}
          </span>
        </h2>

        <div className="flex items-center gap-3">
          {/* Sort control */}
          <div
            className="flex items-center rounded-md overflow-hidden"
            style={{ backgroundColor: "var(--color-surface)" }}
            role="radiogroup"
            aria-label="Sort streams"
          >
            <button
              onClick={() => handleSortChange("viewers")}
              className={`px-3 py-1 text-xs font-medium transition-colors ${
                sortMode === "viewers"
                  ? "bg-[var(--color-primary)] text-white"
                  : "text-[var(--color-text-muted)] hover:text-[var(--color-text)]"
              }`}
              role="radio"
              aria-checked={sortMode === "viewers"}
              aria-label="Sort by viewers"
            >
              Viewers
            </button>
            <button
              onClick={() => handleSortChange("recent")}
              className={`px-3 py-1 text-xs font-medium transition-colors ${
                sortMode === "recent"
                  ? "bg-[var(--color-primary)] text-white"
                  : "text-[var(--color-text-muted)] hover:text-[var(--color-text)]"
              }`}
              role="radio"
              aria-checked={sortMode === "recent"}
              aria-label="Sort by recent"
            >
              Recent
            </button>
          </div>

          {/* View toggle */}
          <div className="flex items-center gap-1">
            <button
              onClick={() => setViewMode("grid")}
              className={`rounded-md px-2 py-1 text-sm transition-colors ${
                viewMode === "grid"
                  ? "bg-[var(--color-primary)] text-white"
                  : "text-[var(--color-text-muted)] hover:text-[var(--color-text)]"
              }`}
              aria-label="Grid view"
              aria-pressed={viewMode === "grid"}
            >
              ▦
            </button>
            <button
              onClick={() => setViewMode("list")}
              className={`rounded-md px-2 py-1 text-sm transition-colors ${
                viewMode === "list"
                  ? "bg-[var(--color-primary)] text-white"
                  : "text-[var(--color-text-muted)] hover:text-[var(--color-text)]"
              }`}
              aria-label="List view"
              aria-pressed={viewMode === "list"}
            >
              ≡
            </button>
          </div>
        </div>
      </div>

      {/* Stream cards */}
      <div className={gridClass} role="list">
        {sortedStreams.map((stream) => (
          <LiveStreamCard key={stream.userId} stream={stream} />
        ))}
      </div>
    </section>
  );
}
