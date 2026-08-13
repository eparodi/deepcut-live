"use client";
// Client Component — search input, pagination, error handling

import { useState, useEffect, useCallback, useRef, Suspense } from "react";
import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { searchVods } from "@/lib/api";
import { VodCard } from "@/components/VodCard";
import type { VodItem } from "@/types";

type SearchState =
  | { status: "idle" }
  | { status: "loading" }
  | {
      status: "results";
      /** The query that produced these results (NOT the live input value). */
      query: string;
      vods: VodItem[];
      totalCount: number;
      page: number;
      loadingMore: boolean;
    }
  | { status: "empty"; query: string }
  | { status: "error"; query: string };

const PAGE_SIZE = 20;

function SearchContent() {
  const searchParams = useSearchParams();
  const urlQuery = searchParams.get("q") || "";
  const initialUserId = searchParams.get("userId") || "";

  const [query, setQuery] = useState(urlQuery);
  const [state, setState] = useState<SearchState>({ status: "idle" });
  const abortRef = useRef<AbortController | null>(null);

  const performSearch = useCallback(
    async (q: string, page = 1, append = false) => {
      const hasQuery = q.trim() || initialUserId;
      if (!hasQuery) {
        setState({ status: "idle" });
        return;
      }

      // Abort any in-flight request so a slow earlier search can't
      // overwrite the results of a newer one.
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;

      setState((prev) =>
        append && prev.status === "results"
          ? { ...prev, loadingMore: true }
          : { status: "loading" }
      );

      try {
        const result = await searchVods(
          {
            query: q || undefined,
            userId: initialUserId || undefined,
            page,
            limit: PAGE_SIZE,
          },
          { signal: controller.signal }
        );

        setState((prev) => {
          if (append && prev.status === "results") {
            return {
              status: "results",
              query: q,
              vods: [...prev.vods, ...result.vods],
              totalCount: result.totalCount,
              page,
              loadingMore: false,
            };
          }
          if (result.vods.length === 0) {
            return { status: "empty", query: q };
          }
          return {
            status: "results",
            query: q,
            vods: result.vods,
            totalCount: result.totalCount,
            page,
            loadingMore: false,
          };
        });
      } catch (err) {
        // An aborted request means a newer search took over — ignore it.
        if (err instanceof DOMException && err.name === "AbortError") return;
        setState((prev) =>
          append && prev.status === "results"
            ? { ...prev, loadingMore: false }
            : { status: "error", query: q }
        );
      }
    },
    [initialUserId]
  );

  // Search whenever the URL query changes (initial load AND client-side
  // navigation like /search?q=a → /search?q=b).
  useEffect(() => {
    if (!urlQuery && !initialUserId) return;
    // eslint-disable-next-line react-hooks/set-state-in-effect -- sync the input with the URL-driven search
    setQuery(urlQuery);
    performSearch(urlQuery);
  }, [urlQuery, initialUserId, performSearch]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    performSearch(query);
  };

  const handleLoadMore = () => {
    if (state.status === "results") {
      // Use the query that produced the current results — the input box
      // may have been edited without submitting.
      performSearch(state.query, state.page + 1, true);
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
            Retry
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

// useSearchParams requires a Suspense boundary during static generation.
export default function SearchPage() {
  return (
    <Suspense
      fallback={<main className="flex-1 w-full max-w-7xl mx-auto px-6 py-8" />}
    >
      <SearchContent />
    </Suspense>
  );
}
