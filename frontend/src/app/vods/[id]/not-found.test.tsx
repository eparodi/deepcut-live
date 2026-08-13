import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import VodNotFound from "./not-found";

describe("VodNotFound", () => {
  it("renders the not-found heading", () => {
    render(<VodNotFound />);
    expect(screen.getByText("VOD not found")).toBeInTheDocument();
  });

  it("explains the VOD may have been removed", () => {
    render(<VodNotFound />);
    expect(
      screen.getByText(/doesn't exist or may have been removed/)
    ).toBeInTheDocument();
  });

  it("links back to the home page", () => {
    render(<VodNotFound />);
    const link = screen.getByRole("link", {
      name: /Back to live streams/,
    });
    expect(link).toHaveAttribute("href", "/");
  });
});
