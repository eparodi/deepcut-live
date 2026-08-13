import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import SearchPage from "./page";
import type { SearchResponse } from "@/types";

// ── Mocks ──────────────────────────────────────────────────────

const mockSearchParams = new URLSearchParams();
let mockSearchVodsResolve: (value: SearchResponse) => void;
let mockSearchVodsReject: (reason: Error) => void;

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: vi.fn() }),
  useSearchParams: () => mockSearchParams,
}));

vi.mock("@/lib/api", () => ({
  searchVods: vi.fn(
    () =>
      new Promise<SearchResponse>((resolve, reject) => {
        mockSearchVodsResolve = resolve;
        mockSearchVodsReject = reject;
      })
  ),
}));

vi.mock("@/components/VodCard", () => ({
  VodCard: ({ vod }: { vod: { id: string; title: string | null; userName: string } }) => (
    <div data-testid="vod-card">
      {vod.title || "Untitled"} by {vod.userName}
    </div>
  ),
}));

// ── Helpers ────────────────────────────────────────────────────

function resolveSearch(vods: SearchResponse["vods"] = [], totalCount = 0) {
  mockSearchVodsResolve({ vods, totalCount, limit: 20, offset: 0 });
}

function rejectSearch() {
  mockSearchVodsReject(new Error("Network error"));
}

// ── Tests ──────────────────────────────────────────────────────

describe("SearchPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockSearchParams.delete("q");
  });

  it("renders empty state before search", () => {
    render(<SearchPage />);
    expect(
      screen.getByText("Search for past streams by title or streamer name")
    ).toBeInTheDocument();
  });

  it("renders search input and button", () => {
    render(<SearchPage />);
    expect(
      screen.getByPlaceholderText("Search past streams...")
    ).toBeInTheDocument();
    expect(screen.getByText("Search")).toBeInTheDocument();
  });

  it("renders back link to browse", () => {
    render(<SearchPage />);
    const link = screen.getByText("← Browse streams");
    expect(link).toBeInTheDocument();
    expect(link.closest("a")).toHaveAttribute("href", "/");
  });

  it("shows loading state during search", async () => {
    render(<SearchPage />);
    const input = screen.getByPlaceholderText("Search past streams...");
    fireEvent.change(input, { target: { value: "test" } });
    fireEvent.click(screen.getByText("Search"));

    expect(screen.getByText("Searching...")).toBeInTheDocument();

    // Clean up
    resolveSearch();
  });

  it("shows results after successful search", async () => {
    render(<SearchPage />);
    const input = screen.getByPlaceholderText("Search past streams...");
    fireEvent.change(input, { target: { value: "test" } });
    fireEvent.click(screen.getByText("Search"));

    await waitFor(() => {
      resolveSearch(
        [
          {
            id: "vod-1",
            userId: "user-1",
            userName: "TestStreamer",
            userAvatar: null,
            title: "Awesome Stream",
            startedAt: "2026-01-01T00:00:00Z",
            endedAt: null,
            durationSeconds: 3600,
            peakViewers: 100,
            totalViewers: 500,
            recordingPath: null,
            recordingStatus: "ready",
            thumbnailUrl: null,
            createdAt: "2026-01-01T00:00:00Z",
          },
        ],
        1
      );
    });

    await waitFor(() => {
      expect(screen.getByTestId("vod-card")).toBeInTheDocument();
    });
  });

  it("shows no results message for empty search", async () => {
    render(<SearchPage />);
    const input = screen.getByPlaceholderText("Search past streams...");
    fireEvent.change(input, { target: { value: "nonexistent" } });
    fireEvent.click(screen.getByText("Search"));

    await waitFor(() => resolveSearch([], 0));

    await waitFor(() => {
      expect(screen.getByText(/No results found/)).toBeInTheDocument();
    });
  });

  it("shows error state and retry button", async () => {
    render(<SearchPage />);
    const input = screen.getByPlaceholderText("Search past streams...");
    fireEvent.change(input, { target: { value: "test" } });
    fireEvent.click(screen.getByText("Search"));

    await waitFor(() => rejectSearch());

    await waitFor(() => {
      expect(screen.getByText("Something went wrong")).toBeInTheDocument();
      expect(screen.getByText("Retry")).toBeInTheDocument();
    });
  });

  it("shows results count", async () => {
    render(<SearchPage />);
    const input = screen.getByPlaceholderText("Search past streams...");
    fireEvent.change(input, { target: { value: "test" } });
    fireEvent.click(screen.getByText("Search"));

    await waitFor(() =>
      resolveSearch(
        [
          {
            id: "vod-1",
            userId: "user-1",
            userName: "TestStreamer",
            userAvatar: null,
            title: "Stream",
            startedAt: "2026-01-01T00:00:00Z",
            endedAt: null,
            durationSeconds: 100,
            peakViewers: 10,
            totalViewers: 20,
            recordingPath: null,
            recordingStatus: "ready",
            thumbnailUrl: null,
            createdAt: "2026-01-01T00:00:00Z",
          },
        ],
        42
      )
    );

    await waitFor(() => {
      expect(screen.getByText(/Showing 1 of 42 results/)).toBeInTheDocument();
    });
  });

  it("pre-fills search from ?q= URL param", () => {
    mockSearchParams.set("q", "prefilled");
    render(<SearchPage />);
    const input = screen.getByPlaceholderText(
      "Search past streams..."
    ) as HTMLInputElement;
    expect(input.value).toBe("prefilled");
  });
});
