import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import RootLoading from "./loading";

describe("RootLoading", () => {
  it("renders skeleton placeholders", () => {
    render(<RootLoading />);
    const region = screen.getByRole("status");
    expect(region).toBeInTheDocument();
    expect(region.querySelectorAll(".skeleton").length).toBeGreaterThan(0);
  });
});
