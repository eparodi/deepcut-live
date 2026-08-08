import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import ChannelLoading from "./loading";

describe("ChannelLoading", () => {
  it("renders skeleton elements", () => {
    const { container } = render(<ChannelLoading />);
    // skeleton class is used for all loading placeholders
    const skeletons = container.querySelectorAll(".skeleton");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders loading structure without crashing", () => {
    const { container } = render(<ChannelLoading />);
    expect(container.firstChild).toBeInTheDocument();
  });
});
