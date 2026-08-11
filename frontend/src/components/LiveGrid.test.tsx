import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { LiveGrid } from "./LiveGrid";
import type { LiveStream } from "@/types";

// ── Mocks ──────────────────────────────────────────────────────

const mockReplace = vi.fn();
const mockSearchParams = new URLSearchParams();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: mockReplace }),
  useSearchParams: () => mockSearchParams,
}));

// ── Test data ──────────────────────────────────────────────────

const baseStream: LiveStream = {
  userId: "user-1",
  streamerName: "TestStreamer",
  streamerAvatarUrl: "https://example.com/avatar.jpg",
  streamId: "stream-1",
  title: "Awesome Stream",
  category: "Gaming",
  viewerCount: 1234,
  thumbnailUrl: "https://example.com/thumb.jpg",
  startedAt: "2026-01-01T00:00:00Z",
};

const recentStream: LiveStream = {
  ...baseStream,
  userId: "user-2",
  streamerName: "RecentStreamer",
  title: "Brand New Stream",
  viewerCount: 10,
  startedAt: "2026-06-01T00:00:00Z",
};

describe("LiveGrid", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Reset search params to default (no sort param)
    mockSearchParams.delete("sort");
  });

  it("renders empty state when no streams", () => {
    render(<LiveGrid streams={[]} total={0} />);
    expect(screen.getByText("No one is live right now")).toBeInTheDocument();
  });

  it("renders 'Browse past streams' link in empty state", () => {
    render(<LiveGrid streams={[]} total={0} />);
    expect(screen.getByText("Browse past streams")).toBeInTheDocument();
  });

  it("renders stream cards for each stream", () => {
    render(<LiveGrid streams={[baseStream]} total={1} />);
    expect(screen.getByText("TestStreamer")).toBeInTheDocument();
    expect(screen.getByText("Awesome Stream")).toBeInTheDocument();
  });

  it("renders total count with plural 'streams'", () => {
    render(
      <LiveGrid
        streams={[baseStream, { ...baseStream, userId: "user-2" }]}
        total={2}
      />
    );
    expect(screen.getByText(/2 streams/)).toBeInTheDocument();
  });

  it("renders total count with singular 'stream'", () => {
    render(<LiveGrid streams={[baseStream]} total={1} />);
    expect(screen.getByText(/1 stream/)).toBeInTheDocument();
  });

  it("defaults to grid view", () => {
    render(<LiveGrid streams={[baseStream]} total={1} />);
    expect(screen.getByLabelText("Grid view")).toHaveAttribute(
      "aria-pressed",
      "true"
    );
  });

  it("renders live indicator dot", () => {
    const { container } = render(
      <LiveGrid streams={[baseStream]} total={1} />
    );
    const pulseDot = container.querySelector(".animate-pulse");
    expect(pulseDot).toBeInTheDocument();
  });

  // ── Sort tests ───────────────────────────────────────────────

  it("defaults sort to viewers (high to low)", () => {
    render(<LiveGrid streams={[recentStream, baseStream]} total={2} />);
    // baseStream has 1234 viewers, recentStream has 10 — baseStream should be first
    const cards = screen.getAllByRole("listitem");
    expect(cards[0]).toHaveTextContent("TestStreamer");
    expect(cards[1]).toHaveTextContent("RecentStreamer");
  });

  it("sorts by recent when sort=recent is in URL", () => {
    mockSearchParams.set("sort", "recent");
    render(<LiveGrid streams={[baseStream, recentStream]} total={2} />);
    // recentStream started 2026-06-01, baseStream started 2026-01-01
    const cards = screen.getAllByRole("listitem");
    expect(cards[0]).toHaveTextContent("RecentStreamer");
    expect(cards[1]).toHaveTextContent("TestStreamer");
  });

  it("updates URL when switching to recent sort", () => {
    render(<LiveGrid streams={[baseStream]} total={1} />);
    fireEvent.click(screen.getByLabelText("Sort by recent"));
    expect(mockReplace).toHaveBeenCalledWith("/?sort=recent", {
      scroll: false,
    });
  });

  it("removes sort param when switching back to viewers", () => {
    mockSearchParams.set("sort", "recent");
    render(<LiveGrid streams={[baseStream]} total={1} />);
    fireEvent.click(screen.getByLabelText("Sort by viewers"));
    expect(mockReplace).toHaveBeenCalledWith("/", { scroll: false });
  });

  it("highlights active sort button (viewers by default)", () => {
    render(<LiveGrid streams={[baseStream]} total={1} />);
    const viewersBtn = screen.getByLabelText("Sort by viewers");
    const recentBtn = screen.getByLabelText("Sort by recent");
    expect(viewersBtn).toHaveAttribute("aria-checked", "true");
    expect(recentBtn).toHaveAttribute("aria-checked", "false");
  });

  it("highlights recent sort button when sort=recent", () => {
    mockSearchParams.set("sort", "recent");
    render(<LiveGrid streams={[baseStream]} total={1} />);
    expect(screen.getByLabelText("Sort by viewers")).toHaveAttribute(
      "aria-checked",
      "false"
    );
    expect(screen.getByLabelText("Sort by recent")).toHaveAttribute(
      "aria-checked",
      "true"
    );
  });
});
