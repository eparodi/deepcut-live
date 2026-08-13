"use client";
// Client Component — needs onClick for sign-in redirect, reads auth state from cookie

import Link from "next/link";
import { useState, useEffect, useCallback } from "react";
import { API_BASE_URL, getMe } from "@/lib/api";
import { AVATAR_FALLBACK } from "@/lib/fallbacks";
import type { User } from "@/types";

interface NavbarProps {
  /** Whether the user has a token cookie (read server-side) */
  initialSignedIn: boolean;
}

// Nav links get vertical padding (compensated by negative margins) so
// their effective tap/click target is >= 24px (WCAG 2.5.8) without
// changing the visual rhythm.
const NAV_LINK_CLASS =
  "text-sm font-medium text-[var(--color-text)] hover:text-[var(--color-primary)] transition-colors px-2 py-2 -my-2";

export function Navbar({ initialSignedIn }: NavbarProps) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(initialSignedIn);
  // Distinguishes "signed in but profile fetch failed" from "signed out"
  // so transient errors never show the sign-in button to signed-in users.
  const [authError, setAuthError] = useState(false);

  const fetchUser = useCallback(() => {
    if (!initialSignedIn) return;
    setLoading(true);
    setAuthError(false);
    getMe()
      .then(setUser)
      .catch(() => setAuthError(true))
      .finally(() => setLoading(false));
  }, [initialSignedIn]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- fetch-on-mount is the intended pattern
    fetchUser();
  }, [fetchUser]);

  return (
    <header
      className="w-full border-b sticky top-0 z-40"
      style={{
        borderColor: "var(--color-surface-raised)",
        backgroundColor: "var(--color-surface)",
      }}
    >
      <div className="max-w-7xl mx-auto px-6 py-3 flex items-center justify-between flex-wrap gap-x-6 gap-y-2">
        {/* Left: Logo + Nav links */}
        <div className="flex items-center gap-6">
          <Link
            href="/"
            className="font-bold text-xl"
            style={{ color: "var(--color-primary)" }}
          >
            DeepCut
          </Link>
          <nav className="flex items-center gap-4">
            <Link href="/" className={NAV_LINK_CLASS}>
              Browse
            </Link>
            <Link href="/search" className={NAV_LINK_CLASS}>
              Search
            </Link>
          </nav>
        </div>

        {/* Right: Auth state */}
        <div className="flex items-center gap-3">
          {loading ? (
            <div className="w-8 h-8 rounded-full skeleton" />
          ) : user ? (
            <Link
              href="/dashboard"
              className="flex items-center gap-2 text-sm font-medium text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors py-1.5 -my-1.5 px-2 -mx-2"
            >
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={user.avatarUrl}
                alt={user.name}
                className="w-8 h-8 rounded-full"
                referrerPolicy="no-referrer"
                onError={(e) => {
                  const target = e.target as HTMLImageElement;
                  target.onerror = null;
                  target.src = AVATAR_FALLBACK;
                }}
              />
              Dashboard
            </Link>
          ) : initialSignedIn && authError ? (
            <Link href="/dashboard" className={NAV_LINK_CLASS}>
              Dashboard
            </Link>
          ) : (
            <a
              href={`${API_BASE_URL}/api/auth/google`}
              className="rounded-lg px-4 py-2 text-sm font-semibold text-white transition-colors hover:opacity-90"
              style={{ backgroundColor: "var(--color-google-blue)" }}
            >
              Sign in with Google
            </a>
          )}
        </div>
      </div>
    </header>
  );
}
