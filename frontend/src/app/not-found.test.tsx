import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import RootNotFound from "./not-found";

describe("RootNotFound", () => {
  it("renders the not-found heading", () => {
    render(<RootNotFound />);
    expect(screen.getByText("Page not found")).toBeInTheDocument();
  });

  it("links back to the home page", () => {
    render(<RootNotFound />);
    const link = screen.getByRole("link", {
      name: /Back to live streams/,
    });
    expect(link).toHaveAttribute("href", "/");
  });
});
