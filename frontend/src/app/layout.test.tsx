import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

// ── Mocks ──────────────────────────────────────────────────────

vi.mock("next/font/google", () => ({
  Inter: () => ({
    subsets: ["latin"],
    variable: "--font-inter",
    className: "inter-font",
  }),
}));

vi.mock("next/headers", () => ({
  cookies: vi.fn(() =>
    Promise.resolve({
      get: vi.fn(() => undefined),
    })
  ),
}));

vi.mock("@/components/Navbar", () => ({
  Navbar: ({ initialSignedIn }: { initialSignedIn: boolean }) => (
    <nav data-testid="navbar" data-signed-in={initialSignedIn}>
      Navbar
    </nav>
  ),
}));

import RootLayout from "./layout";

// ── Tests ────────────────────────────────────────────────────

describe("RootLayout", () => {
  it("renders children inside the body", async () => {
    const jsx = await RootLayout({
      children: <p>Hello world</p>,
    });
    render(jsx);
    expect(screen.getByText("Hello world")).toBeInTheDocument();
  });

  it("sets the html lang attribute to en", async () => {
    const jsx = await RootLayout({
      children: <p>child</p>,
    });
    render(jsx);
    const html = document.documentElement;
    expect(html).toHaveAttribute("lang", "en");
  });

  it("renders the Navbar component", async () => {
    const jsx = await RootLayout({
      children: <p>child</p>,
    });
    render(jsx);
    expect(screen.getByTestId("navbar")).toBeInTheDocument();
  });

  it("renders the Inter font variable class", async () => {
    const jsx = await RootLayout({
      children: <p>child</p>,
    });
    render(jsx);
    const html = document.documentElement;
    expect(html.className).toContain("antialiased");
  });

  it("passes signedIn=false to Navbar when no token cookie", async () => {
    const jsx = await RootLayout({
      children: <p>child</p>,
    });
    render(jsx);
    const navbar = screen.getByTestId("navbar");
    expect(navbar.getAttribute("data-signed-in")).toBe("false");
  });
});
