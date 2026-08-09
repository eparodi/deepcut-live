import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";

// ── Mocks ──────────────────────────────────────────────────────

vi.mock("@/lib/api", () => ({
  getMe: vi.fn(),
  getAnalytics: vi.fn(),
  regenerateStreamKey: vi.fn(),
}));

vi.mock("@/components/ui/Toast", () => ({
  useToast: vi.fn(() => ({
    showToast: vi.fn(),
    ToastComponent: null,
  })),
}));

import { getMe, getAnalytics, regenerateStreamKey } from "@/lib/api";
import DashboardPage from "./page";
import type { User, Analytics } from "@/types";

// ── Helpers ────────────────────────────────────────────────────

const mockUser: User = {
  id: "user-1",
  name: "Test User",
  email: "test@example.com",
  avatarUrl: "https://example.com/avatar.jpg",
  streamKey: "sk_test_123456",
  streamTitle: "My Stream",
  streamCategory: "Gaming",
  isLive: false,
};

const mockUserNoKey: User = {
  ...mockUser,
  streamKey: undefined,
};

const mockAnalytics: Analytics = {
  period: "week",
  startDate: "2026-01-01",
  endDate: "2026-01-08",
  totalStreamTimeSeconds: 3665,
  peakViewers: 2500,
  totalUniqueViewers: 10000,
  totalStreams: 5,
};

// ── Tests ──────────────────────────────────────────────────────

describe("DashboardPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading skeleton initially", () => {
    // Don't resolve the API calls yet — keep loading state
    vi.mocked(getMe).mockReturnValue(new Promise(() => {}));
    vi.mocked(getAnalytics).mockReturnValue(new Promise(() => {}));

    render(<DashboardPage />);

    // Should show skeleton elements
    const skeletons = document.querySelectorAll(".skeleton");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders dashboard heading after loading", async () => {
    vi.mocked(getMe).mockResolvedValue(mockUser);
    vi.mocked(getAnalytics).mockResolvedValue(mockAnalytics);

    render(<DashboardPage />);

    await waitFor(() => {
      expect(screen.getByText("Dashboard")).toBeInTheDocument();
    });
  });

  it("renders stream key display", async () => {
    vi.mocked(getMe).mockResolvedValue(mockUser);
    vi.mocked(getAnalytics).mockResolvedValue(mockAnalytics);

    render(<DashboardPage />);

    await waitFor(() => {
      const headings = screen.getAllByText("Stream Settings");
      expect(headings.length).toBeGreaterThanOrEqual(1);
    });
  });

  it("renders analytics section", async () => {
    vi.mocked(getMe).mockResolvedValue(mockUser);
    vi.mocked(getAnalytics).mockResolvedValue(mockAnalytics);

    render(<DashboardPage />);

    await waitFor(() => {
      expect(
        screen.getByText("Analytics (This Week)")
      ).toBeInTheDocument();
    });
  });

  it("auto-generates stream key when user has none", async () => {
    vi.mocked(getMe).mockResolvedValue(mockUserNoKey);
    vi.mocked(getAnalytics).mockResolvedValue(mockAnalytics);
    vi.mocked(regenerateStreamKey).mockResolvedValue({
      streamKey: "sk_auto_generated",
    });

    render(<DashboardPage />);

    await waitFor(() => {
      expect(regenerateStreamKey).toHaveBeenCalled();
    });
  });

  it("does NOT auto-generate key when user already has one", async () => {
    vi.mocked(getMe).mockResolvedValue(mockUser);
    vi.mocked(getAnalytics).mockResolvedValue(mockAnalytics);

    render(<DashboardPage />);

    await waitFor(() => {
      expect(screen.getByText("Dashboard")).toBeInTheDocument();
    });

    // regenerateStreamKey should NOT have been called
    expect(regenerateStreamKey).not.toHaveBeenCalled();
  });

  it("shows error state when API fails", async () => {
    vi.mocked(getMe).mockRejectedValue(new Error("Unauthorized"));
    vi.mocked(getAnalytics).mockRejectedValue(new Error("Unauthorized"));

    render(<DashboardPage />);

    await waitFor(() => {
      expect(
        screen.getByText("Could not load dashboard")
      ).toBeInTheDocument();
    });

    expect(screen.getByText("Retry")).toBeInTheDocument();
  });

  it("shows ForceEndButton when user is live", async () => {
    const liveUser = { ...mockUser, isLive: true };
    vi.mocked(getMe).mockResolvedValue(liveUser);
    vi.mocked(getAnalytics).mockResolvedValue(mockAnalytics);

    render(<DashboardPage />);

    await waitFor(() => {
      expect(
        screen.getByText("⏹ End Stream")
      ).toBeInTheDocument();
    });
  });
});
