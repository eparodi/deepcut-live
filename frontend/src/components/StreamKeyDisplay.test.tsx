import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, act } from "@testing-library/react";
import { StreamKeyDisplay } from "./StreamKeyDisplay";

// Mock the Toast hook
vi.mock("@/components/ui/Toast", () => ({
  useToast: vi.fn(() => ({
    showToast: vi.fn(),
    ToastComponent: null,
  })),
}));

import { useToast } from "@/components/ui/Toast";

describe("StreamKeyDisplay", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders the server URL", () => {
    render(<StreamKeyDisplay streamKey="test-key" />);
    expect(
      screen.getByText("rtmp://localhost:1935/live")
    ).toBeInTheDocument();
  });

  it("masks the stream key", () => {
    render(
      <StreamKeyDisplay streamKey="dc_live_abcdefghijklmnopqrstuvwxyz1234567890" />
    );
    // Should show first 6 chars, bullets, last 6 chars
    expect(screen.getByText(/dc_liv/)).toBeInTheDocument();
    expect(screen.getByText(/7890/)).toBeInTheDocument();
  });

  it("renders copy buttons", () => {
    render(<StreamKeyDisplay streamKey="test-key" />);
    const copyButtons = screen.getAllByText("📋 Copy");
    expect(copyButtons.length).toBe(2);
  });

  it("shows placeholder when no stream key", () => {
    render(<StreamKeyDisplay streamKey={undefined} />);
    expect(
      screen.getByText(/No stream key/)
    ).toBeInTheDocument();
  });

  it("disables key copy button when no key", () => {
    render(<StreamKeyDisplay streamKey={undefined} />);
    const keyCopyButton = screen.getAllByText("📋 Copy")[1];
    expect(keyCopyButton).toBeDisabled();
  });

  it("shows OBS setup hint when key exists", () => {
    render(<StreamKeyDisplay streamKey="test-key" />);
    expect(screen.getByText(/In OBS/)).toBeInTheDocument();
  });

  it("shows toast on key copy", async () => {
    const showToast = vi.fn();
    vi.mocked(useToast).mockReturnValue({
      showToast,
      ToastComponent: null,
    });

    // Mock clipboard API
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, {
      clipboard: { writeText },
    });

    render(<StreamKeyDisplay streamKey="test-key" />);
    const keyCopyButton = screen.getAllByText("📋 Copy")[1];
    await act(async () => {
      keyCopyButton.click();
    });

    expect(showToast).toHaveBeenCalledWith(
      "Stream key copied to clipboard!",
      "success"
    );
  });
});
