import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";

// ── Mocks ──────────────────────────────────────────────────────

const mockRedirect = vi.fn();
vi.mock("next/navigation", () => ({
  redirect: (...args: unknown[]) => mockRedirect(...args),
}));

vi.mock("next/headers", () => ({
  cookies: vi.fn(),
}));

import { cookies } from "next/headers";

// We need to import the component after mocks are set up
import LandingPage from "./page";

// ── Helpers ────────────────────────────────────────────────────

function mockTokenCookie(token: string | null) {
  const get = vi.fn((name: string) => {
    if (name === "token" && token) {
      return { value: token, name: "token" };
    }
    return undefined;
  });
  vi.mocked(cookies).mockResolvedValue({ get } as unknown as Awaited<ReturnType<typeof cookies>>);
}

// ── Tests ──────────────────────────────────────────────────────

describe("LandingPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("redirects to /dashboard when token cookie exists", async () => {
    mockTokenCookie("valid-jwt");

    try {
      await LandingPage();
    } catch {
      // redirect throws in Next.js, but our mock just records the call
    }

    expect(mockRedirect).toHaveBeenCalledWith("/dashboard");
  });

  it("renders the hero heading", async () => {
    mockTokenCookie(null);

    // Mock fetch for live stats
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify([]), { status: 200 })
      )
    );

    const jsx = await LandingPage();
    render(jsx);

    expect(
      screen.getByText("Stream What You Believe")
    ).toBeInTheDocument();
  });

  it("renders the Google sign-in link", async () => {
    mockTokenCookie(null);
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify([]), { status: 200 })
      )
    );

    const jsx = await LandingPage();
    render(jsx);

    expect(
      screen.getByText("Start Streaming with Google")
    ).toBeInTheDocument();
  });

  it("renders live stats (0 live now)", async () => {
    mockTokenCookie(null);
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify([]), { status: 200 })
      )
    );

    const jsx = await LandingPage();
    render(jsx);

    expect(screen.getByText(/live now/)).toBeInTheDocument();
    expect(screen.getByText(/past streams/)).toBeInTheDocument();
  });

  it("handles fetch errors gracefully (shows 0 live)", async () => {
    mockTokenCookie(null);
    vi.stubGlobal(
      "fetch",
      vi.fn().mockRejectedValue(new Error("Network down"))
    );

    const jsx = await LandingPage();
    render(jsx);

    // Should still render with 0 counts (two "0" elements: liveCount and pastStreamsCount)
    const zeros = screen.getAllByText("0");
    expect(zeros.length).toBeGreaterThanOrEqual(2);
  });

  it("renders the footer", async () => {
    mockTokenCookie(null);
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify([]), { status: 200 })
      )
    );

    const jsx = await LandingPage();
    render(jsx);

    expect(
      screen.getByText(/Free expression streaming platform/)
    ).toBeInTheDocument();
  });
});
