import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import DashboardError from "./error";

describe("DashboardError", () => {
  it("renders error heading", () => {
    const error = new Error("Dashboard failed");
    const reset = vi.fn();
    render(<DashboardError error={error} reset={reset} />);
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
  });

  it("renders error digest when provided", () => {
    const error = Object.assign(new Error("Boom"), { digest: "xyz789" });
    const reset = vi.fn();
    render(<DashboardError error={error} reset={reset} />);
    expect(screen.getByText(/xyz789/)).toBeInTheDocument();
  });

  it("calls reset when 'Try again' is clicked", () => {
    const error = new Error("Oops");
    const reset = vi.fn();
    render(<DashboardError error={error} reset={reset} />);
    screen.getByText("Retry").click();
    expect(reset).toHaveBeenCalledTimes(1);
  });

  it("renders 'Go home' link", () => {
    const error = new Error("Oops");
    const reset = vi.fn();
    render(<DashboardError error={error} reset={reset} />);
    expect(screen.getByText("Go home")).toBeInTheDocument();
  });
});
