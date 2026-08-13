import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import ChannelError from "./error";

describe("ChannelError", () => {
  it("renders error heading", () => {
    const error = new Error("Something broke");
    const reset = vi.fn();
    render(<ChannelError error={error} reset={reset} />);
    expect(
      screen.getByText("Could not load channel")
    ).toBeInTheDocument();
  });

  it("renders error digest when provided", () => {
    const error = Object.assign(new Error("Boom"), { digest: "abc123" });
    const reset = vi.fn();
    render(<ChannelError error={error} reset={reset} />);
    expect(screen.getByText(/abc123/)).toBeInTheDocument();
  });

  it("calls reset when 'Try again' is clicked", () => {
    const error = new Error("Oops");
    const reset = vi.fn();
    render(<ChannelError error={error} reset={reset} />);
    screen.getByText("Retry").click();
    expect(reset).toHaveBeenCalledTimes(1);
  });

  it("renders 'Go home' link", () => {
    const error = new Error("Oops");
    const reset = vi.fn();
    render(<ChannelError error={error} reset={reset} />);
    expect(screen.getByText("Go home")).toBeInTheDocument();
  });
});
