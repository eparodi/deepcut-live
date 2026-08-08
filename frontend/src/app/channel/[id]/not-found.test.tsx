import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import ChannelNotFound from "./not-found";

describe("ChannelNotFound", () => {
  it("renders 'Channel not found' heading", () => {
    render(<ChannelNotFound />);
    expect(screen.getByText("Channel not found")).toBeInTheDocument();
  });

  it("renders a descriptive message", () => {
    render(<ChannelNotFound />);
    expect(
      screen.getByText(/doesn't exist or may have been removed/i)
    ).toBeInTheDocument();
  });

  it("renders a link back to live streams", () => {
    render(<ChannelNotFound />);
    expect(
      screen.getByText("← Back to live streams")
    ).toBeInTheDocument();
  });
});
