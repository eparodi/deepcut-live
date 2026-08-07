"use client";
// Client Component — needs animation stagger (useEffect for mount)

import { useEffect, useState } from "react";
import type { Analytics } from "@/types";

interface AnalyticsCardsProps {
  analytics: Analytics | null;
  loading: boolean;
  error: string | null;
  onRetry: () => void;
}

/** Format a number for display: 1205 → "1.2k", 14205 → "14.2k", 1234567 → "1.2M" */
function formatNumber(n: number): string {
  if (n >= 1_000_000) {
    const val = n / 1_000_000;
    return val % 1 === 0 ? `${val}M` : `${val.toFixed(1)}M`;
  }
  if (n >= 1_000) {
    const val = n / 1_000;
    return val % 1 === 0 ? `${val}k` : `${val.toFixed(1)}k`;
  }
  return n.toLocaleString();
}

/** Format seconds into human-readable duration */
function formatDuration(totalSeconds: number): string {
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  return `${minutes}m`;
}

interface CardProps {
  value: string;
  label: string;
  delay: number;
}

function StatCard({ value, label, delay }: CardProps) {
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => setVisible(true), delay);
    return () => clearTimeout(timer);
  }, [delay]);

  return (
    <div
      className="rounded-xl p-5 text-center transition-all duration-500"
      style={{
        backgroundColor: "var(--color-surface-raised)",
        opacity: visible ? 1 : 0,
        transform: visible ? "translateY(0)" : "translateY(8px)",
      }}
    >
      <p
        className="text-[var(--text-2xl)] font-bold text-[var(--color-text)]"
        style={{ fontSize: "var(--text-2xl)", lineHeight: "1.2" }}
      >
        {value}
      </p>
      <p
        className="mt-1 text-[var(--color-text-muted)]"
        style={{ fontSize: "var(--text-xs)", lineHeight: "1.5" }}
      >
        {label}
      </p>
    </div>
  );
}

export function AnalyticsCards({
  analytics,
  loading,
  error,
  onRetry,
}: AnalyticsCardsProps) {
  // Loading state: 4 skeleton cards
  if (loading) {
    return (
      <section aria-labelledby="analytics-heading">
        <h2
          id="analytics-heading"
          className="text-lg font-semibold text-[var(--color-text)] mb-4"
        >
          Analytics (This Week)
        </h2>
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          {[0, 1, 2, 3].map((i) => (
            <div
              key={i}
              className="skeleton rounded-xl h-24"
              style={{ backgroundColor: "var(--color-surface-raised)" }}
            />
          ))}
        </div>
      </section>
    );
  }

  // Error state
  if (error) {
    return (
      <section aria-labelledby="analytics-heading">
        <h2
          id="analytics-heading"
          className="text-lg font-semibold text-[var(--color-text)] mb-4"
        >
          Analytics (This Week)
        </h2>
        <div
          className="rounded-xl p-6 text-center"
          style={{ backgroundColor: "var(--color-surface-raised)" }}
        >
          <p className="text-[var(--color-text-muted)] text-sm">
            Analytics unavailable
          </p>
          <button
            onClick={onRetry}
            className="mt-3 text-sm font-medium hover:underline"
            style={{ color: "var(--color-primary)" }}
          >
            Try again
          </button>
        </div>
      </section>
    );
  }

  // Empty state: never streamed
  if (!analytics || analytics.totalStreams === 0) {
    return (
      <section aria-labelledby="analytics-heading">
        <h2
          id="analytics-heading"
          className="text-lg font-semibold text-[var(--color-text)] mb-4"
        >
          Analytics (This Week)
        </h2>
        <div
          className="rounded-xl p-8 text-center"
          style={{ backgroundColor: "var(--color-surface-raised)" }}
        >
          <p className="text-[var(--color-text-muted)]">
            Start streaming to see analytics
          </p>
        </div>
      </section>
    );
  }

  // Populated state
  const cards = [
    {
      value: formatDuration(analytics.totalStreamTimeSeconds),
      label: "stream time",
    },
    { value: formatNumber(analytics.peakViewers), label: "peak viewers" },
    {
      value: formatNumber(analytics.totalUniqueViewers),
      label: "unique viewers",
    },
    { value: formatNumber(analytics.totalStreams), label: "streams this wk" },
  ];

  return (
    <section aria-labelledby="analytics-heading">
      <h2
        id="analytics-heading"
        className="text-lg font-semibold text-[var(--color-text)] mb-4"
      >
        Analytics (This Week)
      </h2>
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {cards.map((card, i) => (
          <StatCard
            key={card.label}
            value={card.value}
            label={card.label}
            delay={i * 50}
          />
        ))}
      </div>
    </section>
  );
}
