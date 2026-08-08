import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

// Mock next/font/google — not available in jsdom
vi.mock("next/font/google", () => ({
  Inter: () => ({
    subsets: ["latin"],
    variable: "--font-inter",
    className: "inter-font",
  }),
}));

import RootLayout from "./layout";

describe("RootLayout", () => {
  it("renders children inside the body", () => {
    render(
      <RootLayout>
        <p>Hello world</p>
      </RootLayout>
    );
    expect(screen.getByText("Hello world")).toBeInTheDocument();
  });

  it("sets the html lang attribute to en", () => {
    render(
      <RootLayout>
        <p>child</p>
      </RootLayout>
    );
    const html = document.documentElement;
    expect(html).toHaveAttribute("lang", "en");
  });

  it("renders the Inter font variable class", () => {
    render(
      <RootLayout>
        <p>child</p>
      </RootLayout>
    );
    const html = document.documentElement;
    expect(html.className).toContain("antialiased");
  });
});
