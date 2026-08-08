import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import DashboardLoading from "./loading";

describe("DashboardLoading", () => {
  it("renders skeleton elements", () => {
    const { container } = render(<DashboardLoading />);
    const skeletons = container.querySelectorAll(".skeleton");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders loading structure without crashing", () => {
    const { container } = render(<DashboardLoading />);
    expect(container.firstChild).toBeInTheDocument();
  });
});
