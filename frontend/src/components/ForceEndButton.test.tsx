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

  // Focus management (specs/ui-ux-audit-v1.md D3)

  it("moves focus into the dialog when it opens", async () => {
    const user = userEvent.setup();
    render(<ForceEndButton onEnded={vi.fn()} onError={vi.fn()} />);
    await user.click(screen.getByText("⏹ End Stream"));
    expect(screen.getByText("Keep Streaming")).toHaveFocus();
  });

  it("closes on Escape and restores focus to the trigger", async () => {
    const user = userEvent.setup();
    render(<ForceEndButton onEnded={vi.fn()} onError={vi.fn()} />);
    const trigger = screen.getByText("⏹ End Stream");
    await user.click(trigger);
    expect(screen.getByText("End Stream?")).toBeInTheDocument();

    await user.keyboard("{Escape}");

    expect(screen.queryByText("End Stream?")).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });

  it("traps Tab within the dialog", async () => {
    const user = userEvent.setup();
    render(<ForceEndButton onEnded={vi.fn()} onError={vi.fn()} />);
    await user.click(screen.getByText("⏹ End Stream"));

    const confirm = screen.getByText("End Stream");
    confirm.focus();
    await user.tab();

    // Tab on the last control wraps to the first.
    expect(screen.getByText("Keep Streaming")).toHaveFocus();
  });
});
