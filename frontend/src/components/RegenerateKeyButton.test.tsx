import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RegenerateKeyButton } from "./RegenerateKeyButton";

// Mock the API module
vi.mock("@/lib/api", () => ({
  regenerateStreamKey: vi.fn(),
}));

import { regenerateStreamKey } from "@/lib/api";

describe("RegenerateKeyButton", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders the regenerate button", () => {
    render(
      <RegenerateKeyButton onRegenerated={vi.fn()} onError={vi.fn()} />
    );
    expect(screen.getByText("🔄 Regenerate Key")).toBeInTheDocument();
  });

  it("shows confirmation dialog on click", async () => {
    const user = userEvent.setup();
    render(
      <RegenerateKeyButton onRegenerated={vi.fn()} onError={vi.fn()} />
    );
    await user.click(screen.getByText("🔄 Regenerate Key"));
    expect(screen.getByText("Regenerate Stream Key?")).toBeInTheDocument();
    expect(screen.getByText("Keep Current Key")).toBeInTheDocument();
    expect(screen.getByText("Regenerate")).toBeInTheDocument();
  });

  it("closes dialog when 'Keep Current Key' is clicked", async () => {
    const user = userEvent.setup();
    render(
      <RegenerateKeyButton onRegenerated={vi.fn()} onError={vi.fn()} />
    );
    await user.click(screen.getByText("🔄 Regenerate Key"));
    await user.click(screen.getByText("Keep Current Key"));
    expect(
      screen.queryByText("Regenerate Stream Key?")
    ).not.toBeInTheDocument();
  });

  it("calls onRegenerated with new key on success", async () => {
    const user = userEvent.setup();
    const onRegenerated = vi.fn();
    vi.mocked(regenerateStreamKey).mockResolvedValue({
      streamKey: "new-key-123",
    });

    render(
      <RegenerateKeyButton
        onRegenerated={onRegenerated}
        onError={vi.fn()}
      />
    );
    await user.click(screen.getByText("🔄 Regenerate Key"));
    await user.click(screen.getByText("Regenerate"));

    await vi.waitFor(() => {
      expect(onRegenerated).toHaveBeenCalledWith("new-key-123");
    });
  });

  it("calls onError on failure", async () => {
    const user = userEvent.setup();
    const onError = vi.fn();
    vi.mocked(regenerateStreamKey).mockRejectedValue(
      new Error("Network error")
    );

    render(
      <RegenerateKeyButton
        onRegenerated={vi.fn()}
        onError={onError}
      />
    );
    await user.click(screen.getByText("🔄 Regenerate Key"));
    await user.click(screen.getByText("Regenerate"));

    await vi.waitFor(() => {
      expect(onError).toHaveBeenCalledWith("Network error");
    });
  });

  // Focus management (specs/ui-ux-audit-v1.md D3)

  it("moves focus into the dialog when it opens", async () => {
    const user = userEvent.setup();
    render(
      <RegenerateKeyButton onRegenerated={vi.fn()} onError={vi.fn()} />
    );
    await user.click(screen.getByText("🔄 Regenerate Key"));
    expect(screen.getByText("Keep Current Key")).toHaveFocus();
  });

  it("closes on Escape and restores focus to the trigger", async () => {
    const user = userEvent.setup();
    render(
      <RegenerateKeyButton onRegenerated={vi.fn()} onError={vi.fn()} />
    );
    const trigger = screen.getByText("🔄 Regenerate Key");
    await user.click(trigger);
    expect(screen.getByText("Regenerate Stream Key?")).toBeInTheDocument();

    await user.keyboard("{Escape}");

    expect(
      screen.queryByText("Regenerate Stream Key?")
    ).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });

  it("traps Tab within the dialog", async () => {
    const user = userEvent.setup();
    render(
      <RegenerateKeyButton onRegenerated={vi.fn()} onError={vi.fn()} />
    );
    await user.click(screen.getByText("🔄 Regenerate Key"));

    const regenerate = screen.getByText("Regenerate");
    regenerate.focus();
    await user.tab();

    // Tab on the last control wraps to the first.
    expect(screen.getByText("Keep Current Key")).toHaveFocus();
  });
});
