import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, within } from "@testing-library/react";
import { Navbar } from "./Navbar";

vi.mock("@/lib/api", () => ({
  API_BASE_URL: "http://localhost:8081",
  getMe: vi.fn(),
}));

import { getMe } from "@/lib/api";

describe("Navbar", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders Browse and Search links", () => {
    render(<Navbar initialSignedIn={false} />);
    expect(screen.getByRole("link", { name: "Browse" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Search" })).toBeInTheDocument();
  });

  it("shows the sign-in button when signed out", () => {
    render(<Navbar initialSignedIn={false} />);
    expect(
      screen.getByRole("link", { name: "Sign in with Google" })
    ).toBeInTheDocument();
  });

  it("shows Dashboard with avatar when the profile loads", async () => {
    vi.mocked(getMe).mockResolvedValue({
      id: "u1",
      name: "Ada",
      email: "ada@example.com",
      avatarUrl: "https://example.com/ada.png",
      streamTitle: null,
      streamCategory: null,
      isLive: false,
    });

    render(<Navbar initialSignedIn />);

    const dashboardLink = await screen.findByRole("link", {
      name: /Dashboard/,
    });
    expect(dashboardLink).toHaveAttribute("href", "/dashboard");
    expect(within(dashboardLink).getByAltText("Ada")).toBeInTheDocument();
    expect(
      screen.queryByRole("link", { name: "Sign in with Google" })
    ).not.toBeInTheDocument();
  });

  it("does not show the sign-in button when the profile fetch fails for a signed-in user", async () => {
    vi.mocked(getMe).mockRejectedValue(new Error("network down"));

    render(<Navbar initialSignedIn />);

    expect(await screen.findByRole("link", { name: "Dashboard" })).toBeInTheDocument();
    expect(
      screen.queryByRole("link", { name: "Sign in with Google" })
    ).not.toBeInTheDocument();
  });
});
