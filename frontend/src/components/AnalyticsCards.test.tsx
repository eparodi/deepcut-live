import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { AnalyticsCards } from "./AnalyticsCards";
import type { Analytics } from "@/types";

const analytics: Analytics = {
  period: "week",
  startDate: "2026-01-01",
  endDate: "2026-01-08",
  totalStreamTimeSeconds: 3665, // ~1h 1m
  peakViewers: 2500,
  totalUniqueViewers: 10000,
  totalStreams: 5,
};

describe("AnalyticsCards", () => {
  it("renders loading skeletons when loading is true", () => {
    const { container } = render(
      <AnalyticsCards analytics={null} loading error={null} onRetry={vi.fn()} />
    );
    const skeletons = container.querySelectorAll(".skeleton");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders error state with retry button", () => {
    const onRetry = vi.fn();
    render(
      <AnalyticsCards
        analytics={null}
        loading={false}
        error="Network error"
        onRetry={onRetry}
      />
    );
    expect(screen.getByText("Analytics unavailable")).toBeInTheDocument();
    expect(screen.getByText("Retry")).toBeInTheDocument();
  });

  it("calls onRetry when retry button is clicked", () => {
    const onRetry = vi.fn();
    render(
      <AnalyticsCards
        analytics={null}
        loading={false}
        error="Network error"
        onRetry={onRetry}
      />
    );
    screen.getByText("Retry").click();
    expect(onRetry).toHaveBeenCalled();
  });

  it("renders empty state when no streams", () => {
    const emptyAnalytics: Analytics = { ...analytics, totalStreams: 0 };
    render(
      <AnalyticsCards
        analytics={emptyAnalytics}
        loading={false}
        error={null}
        onRetry={vi.fn()}
      />
    );
    expect(
      screen.getByText("Start streaming to see analytics")
    ).toBeInTheDocument();
  });

  it("renders empty state when analytics is null", () => {
    render(
      <AnalyticsCards
        analytics={null}
        loading={false}
        error={null}
        onRetry={vi.fn()}
      />
    );
    expect(
      screen.getByText("Start streaming to see analytics")
    ).toBeInTheDocument();
  });

  it("renders stat cards with formatted values", () => {
    render(
      <AnalyticsCards
        analytics={analytics}
        loading={false}
        error={null}
        onRetry={vi.fn()}
      />
    );
    expect(screen.getByText("1h 1m")).toBeInTheDocument();
    expect(screen.getByText("2.5k")).toBeInTheDocument();
    expect(screen.getByText("10k")).toBeInTheDocument();
    expect(screen.getByText("5")).toBeInTheDocument();
  });

  it("renders heading for the section", () => {
    render(
      <AnalyticsCards
        analytics={analytics}
        loading={false}
        error={null}
        onRetry={vi.fn()}
      />
    );
    expect(screen.getByText("Analytics (This Week)")).toBeInTheDocument();
  });
});
