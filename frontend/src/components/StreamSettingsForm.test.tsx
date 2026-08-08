import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { StreamSettingsForm } from "./StreamSettingsForm";

vi.mock("@/lib/api", () => ({
  updateStreamSettings: vi.fn(),
}));

import { updateStreamSettings } from "@/lib/api";

describe("StreamSettingsForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders title and category inputs", () => {
    render(
      <StreamSettingsForm
        initialTitle="My Stream"
        initialCategory="Gaming"
        onSave={vi.fn()}
        onError={vi.fn()}
      />
    );
    expect(screen.getByLabelText("Title")).toHaveValue("My Stream");
    expect(screen.getByLabelText("Category")).toHaveValue("Gaming");
  });

  it("renders with null initial values as empty strings", () => {
    render(
      <StreamSettingsForm
        initialTitle={null}
        initialCategory={null}
        onSave={vi.fn()}
        onError={vi.fn()}
      />
    );
    expect(screen.getByLabelText("Title")).toHaveValue("");
    expect(screen.getByLabelText("Category")).toHaveValue("");
  });

  it("disables save button when form is not dirty", () => {
    render(
      <StreamSettingsForm
        initialTitle="My Stream"
        initialCategory="Gaming"
        onSave={vi.fn()}
        onError={vi.fn()}
      />
    );
    expect(screen.getByText("💾 Save")).toBeDisabled();
  });

  it("enables save button when title changes", async () => {
    const user = userEvent.setup();
    render(
      <StreamSettingsForm
        initialTitle="Old Title"
        initialCategory="Gaming"
        onSave={vi.fn()}
        onError={vi.fn()}
      />
    );
    const titleInput = screen.getByLabelText("Title");
    await user.clear(titleInput);
    await user.type(titleInput, "New Title");
    expect(screen.getByText("💾 Save")).not.toBeDisabled();
  });

  it("shows error when submitting empty title", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    render(
      <StreamSettingsForm
        initialTitle="Old Title"
        initialCategory="Gaming"
        onSave={onSave}
        onError={vi.fn()}
      />
    );
    const titleInput = screen.getByLabelText("Title");
    await user.clear(titleInput);
    await user.click(screen.getByText("💾 Save"));
    expect(screen.getByText("Stream title is required.")).toBeInTheDocument();
    expect(onSave).not.toHaveBeenCalled();
  });

  it("calls onSave with updated values on success", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    vi.mocked(updateStreamSettings).mockResolvedValue({
      streamTitle: "New Title",
      streamCategory: "Music",
    });

    render(
      <StreamSettingsForm
        initialTitle="Old Title"
        initialCategory="Gaming"
        onSave={onSave}
        onError={vi.fn()}
      />
    );
    const titleInput = screen.getByLabelText("Title");
    await user.clear(titleInput);
    await user.type(titleInput, "New Title");

    const categoryInput = screen.getByLabelText("Category");
    await user.clear(categoryInput);
    await user.type(categoryInput, "Music");

    await user.click(screen.getByText("💾 Save"));

    await vi.waitFor(() => {
      expect(onSave).toHaveBeenCalledWith("New Title", "Music");
    });
  });

  it("calls onError on API failure", async () => {
    const user = userEvent.setup();
    const onError = vi.fn();
    vi.mocked(updateStreamSettings).mockRejectedValue(
      new Error("Server error")
    );

    render(
      <StreamSettingsForm
        initialTitle="Old Title"
        initialCategory="Gaming"
        onSave={vi.fn()}
        onError={onError}
      />
    );
    const titleInput = screen.getByLabelText("Title");
    await user.clear(titleInput);
    await user.type(titleInput, "New Title");
    await user.click(screen.getByText("💾 Save"));

    await vi.waitFor(() => {
      expect(onError).toHaveBeenCalledWith("Server error");
    });
  });
});
