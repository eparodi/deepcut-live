"use client";
// Client Component — search input, pagination, error handling

import { useState, useEffect, useCallback, useRef } from "react";
import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { searchVods } from "@/lib/api";
import { VodCard } from "@/components/VodCard";
import type { VodItem } from "@/types";

type SearchState =
  | { status: "idle" }
  | { status: "loading" }
  | { status: "results"; vods: VodItem[]; totalCount: number; page: number; loadingMore: boolean }
  | { status: "empty"; query: string }
  | { status: "error"; query: string };

const PAGE_SIZE = 20;

export default function SearchPage() {
  const searchParams = useSearchParams();
  const initialQuery = searchParams.get("q") || "";
  const initialUserId = searchParams.get("userId") || "";
  const hasAutoSearched = useRef(false);

  const [query, setQuery] = useState(initialQuery);
  const [state, setState] = useState<SearchState>({ status: "idle" });

  const performSearch = useCallback(
    async (q: string, page = 1, append = false) => {
      const hasQuery = q.trim() || initialUserId;
      if (!hasQuery) {
        setState({ status: "idle" });
        return;
      }

      if (append && state.status === "results") {
        setState({ ...state, loadingMore: true });
      } else {
        setState({ status: "loading" });
      }

      try {
        const result = await searchVods({ query: q || undefined, userId: initialUserId || undefined, page, limit: PAGE_SIZE });

        if (result.vods.length === 0 && !append) {
          setState({ status: "empty", query: q });
          return;
        }

        if (append && state.status === "results") {
          setState({
            status: "results",
            vods: [...state.vods, ...result.vods],
            totalCount: result.totalCount,
            page,
            loadingMore: false,
          });
        } else {
          setState({
            status: "results",
            vods: result.vods,
            totalCount: result.totalCount,
            page,
            loadingMore: false,
          });
        }
      } catch {
        if (append && state.status === "results") {
          setState({ ...state, loadingMore: false });
        } else {
          				setState({ status: "error", query: q });
          			}
          		}
          	},
          	[state, initialUserId]
          );

  // Auto-search on initial load if ?q= or ?userId= is present
  useEffect(() => {
    if ((initialQuery || initialUserId) && !hasAutoSearched.current) {
      hasAutoSearched.current = true;
      performSearch(initialQuery);
    }
  }, [initialQuery, initialUserId, performSearch]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    performSearch(query);
  };

  const handleLoadMore = () => {
    if (state.status === "results") {
      performSearch(query, state.page + 1, true);
    }
  };

  const hasMore =
    state.status === "results" &&
    state.vods.length < state.totalCount;

  const isInitialLoading = state.status === "loading";
  const isLoadMoreLoading =
    state.status === "results" && state.loadingMore;

  return (
    <main className="flex-1 w-full max-w-7xl mx-auto px-6 py-8">
      {/* Back link */}
      <div className="mb-6">
        <Link
          href="/"
          className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors"
        >
          ← Browse streams
        </Link>
      </div>

      {/* Search form */}
      <form onSubmit={handleSubmit} className="mb-8">
        <div className="flex gap-2">
          <div className="relative flex-1">
            <span
              className="absolute left-3 top-1/2 -translate-y-1/2 text-lg"
              role="img"
              aria-hidden="true"
            >
              🔍
            </span>
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search past streams..."
              aria-label="Search past streams"
              className="w-full pl-10 pr-4 py-2.5 rounded-lg text-sm transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
              style={{
                backgroundColor: "var(--color-surface-raised)",
                color: "var(--color-text)",
              }}
            />
          </div>
          <button
            type="submit"
            disabled={isInitialLoading}
            className="rounded-lg px-6 py-2.5 text-sm font-semibold text-white transition-colors hover:opacity-90 disabled:opacity-50"
            style={{ backgroundColor: "var(--color-primary)" }}
          >
            {isInitialLoading ? "Searching…" : "Search"}
          </button>
        </div>
      </form>

      {/* States */}
      {state.status === "idle" && (
        <div className="text-center py-16">
          <p className="text-4xl mb-4" role="img" aria-label="Search">
            🔎
          </p>
          <p className="text-lg text-[var(--color-text)] font-medium">
            Search for past streams by title or streamer name
          </p>
        </div>
      )}

      {state.status === "loading" && (
        <div className="text-center py-16">
          <p className="text-lg text-[var(--color-text-muted)]">
            Searching...
          </p>
        </div>
      )}

      {state.status === "empty" && (
        <div className="text-center py-16">
          <p className="text-4xl mb-4" role="img" aria-label="No results">
            📭
          </p>
          <p className="text-lg text-[var(--color-text)] font-medium">
            No results found for &ldquo;{state.query}&rdquo;
          </p>
          <p className="mt-2 text-sm text-[var(--color-text-muted)]">
            Try a different search term
          </p>
        </div>
      )}

      {state.status === "error" && (
        <div className="text-center py-16">
          <p className="text-4xl mb-4" role="img" aria-label="Error">
            ⚠️
          </p>
          <p className="text-lg text-[var(--color-text)] font-medium">
            Something went wrong
          </p>
          <button
            onClick={() => performSearch(state.query)}
            className="mt-4 inline-flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:opacity-90"
            style={{ backgroundColor: "var(--color-primary)" }}
          >
            Try again
          </button>
        </div>
      )}

      {state.status === "results" && (
        <>
          <div
            className="grid grid-cols-2 lg:grid-cols-4 gap-4"
            role="list"
          >
            {state.vods.map((vod) => (
              <VodCard key={vod.id} vod={vod} />
            ))}
          </div>

          <div className="mt-8 text-center">
            {hasMore && (
              <button
                onClick={handleLoadMore}
                disabled={isLoadMoreLoading}
                className="inline-flex items-center gap-2 rounded-lg px-6 py-2.5 text-sm font-semibold text-white transition-colors hover:opacity-90 disabled:opacity-50"
                style={{ backgroundColor: "var(--color-primary)" }}
              >
                {isLoadMoreLoading ? "Loading..." : "Load more"}
              </button>
            )}

            <p className="mt-3 text-xs text-[var(--color-text-muted)]">
              Showing {state.vods.length} of {state.totalCount} results
            </p>
          </div>
        </>
      )}
    </main>
  );
}
