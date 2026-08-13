import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import VodLoading from "./loading";

describe("VodLoading", () => {
  it("renders without crashing", () => {
    const { container } = render(<VodLoading />);
    expect(container.firstChild).toBeInTheDocument();
  });

  it("renders skeleton placeholders", () => {
    const { container } = render(<VodLoading />);
    const skeletons = container.querySelectorAll(".skeleton");
    expect(skeletons.length).toBeGreaterThan(0);
  });
});
