import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ForceEndButton } from "./ForceEndButton";

// Mock the API module
vi.mock("@/lib/api", () => ({
  forceEndStream: vi.fn(),
}));

import { forceEndStream } from "@/lib/api";

describe("ForceEndButton", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders the end stream button", () => {
    render(<ForceEndButton onEnded={vi.fn()} onError={vi.fn()} />);
    expect(screen.getByText("⏹ End Stream")).toBeInTheDocument();
  });

  it("shows confirmation dialog on click", async () => {
    const user = userEvent.setup();
    render(<ForceEndButton onEnded={vi.fn()} onError={vi.fn()} />);
    await user.click(screen.getByText("⏹ End Stream"));
    expect(screen.getByText("End Stream?")).toBeInTheDocument();
    expect(screen.getByText("Keep Streaming")).toBeInTheDocument();
    expect(screen.getByText("End Stream")).toBeInTheDocument();
  });

  it("closes dialog when 'Keep Streaming' is clicked", async () => {
    const user = userEvent.setup();
    render(<ForceEndButton onEnded={vi.fn()} onError={vi.fn()} />);
    await user.click(screen.getByText("⏹ End Stream"));
    expect(screen.getByText("End Stream?")).toBeInTheDocument();
    await user.click(screen.getByText("Keep Streaming"));
    expect(screen.queryByText("End Stream?")).not.toBeInTheDocument();
  });

  it("closes dialog when backdrop is clicked", async () => {
    const user = userEvent.setup();
    render(<ForceEndButton onEnded={vi.fn()} onError={vi.fn()} />);
    await user.click(screen.getByText("⏹ End Stream"));
    // Click the backdrop (the black overlay)
    const backdrop = document.querySelector(".bg-black\\/60");
    if (backdrop) {
      await user.click(backdrop);
    }
    expect(screen.queryByText("End Stream?")).not.toBeInTheDocument();
  });

  it("calls onEnded after successful API call", async () => {
    const user = userEvent.setup();
    const onEnded = vi.fn();
    const mockForceEnd = vi.mocked(forceEndStream).mockResolvedValue({
      status: "offline" as const,
      message: "Stream ended",
    });

    render(<ForceEndButton onEnded={onEnded} onError={vi.fn()} />);
    await user.click(screen.getByText("⏹ End Stream"));
    await user.click(screen.getByText("End Stream"));

    // Wait for the async handler to resolve
    await vi.waitFor(() => {
      expect(mockForceEnd).toHaveBeenCalled();
      expect(onEnded).toHaveBeenCalled();
    });
  });

  it("calls onError when API call fails", async () => {
    const user = userEvent.setup();
    const onError = vi.fn();
    vi.mocked(forceEndStream).mockRejectedValue(new Error("Server error"));

    render(<ForceEndButton onEnded={vi.fn()} onError={onError} />);
    await user.click(screen.getByText("⏹ End Stream"));
    await user.click(screen.getByText("End Stream"));

    await vi.waitFor(() => {
      expect(onError).toHaveBeenCalledWith("Server error");
    });
  });
});
