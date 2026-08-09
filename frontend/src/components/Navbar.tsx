"use client";
// Client Component — needs onClick for sign-in redirect, reads auth state from cookie

import Link from "next/link";
import { useState, useEffect, useCallback } from "react";
import { getMe } from "@/lib/api";
import type { User } from "@/types";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:3000";

interface NavbarProps {
  /** Whether the user has a token cookie (read server-side) */
  initialSignedIn: boolean;
}

export function Navbar({ initialSignedIn }: NavbarProps) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(initialSignedIn);

  const fetchUser = useCallback(() => {
    if (!initialSignedIn) return;
    setLoading(true);
    getMe()
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setLoading(false));
  }, [initialSignedIn]);

  useEffect(() => {
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
      <div className="max-w-7xl mx-auto px-6 py-3 flex items-center justify-between">
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
            <Link
              href="/"
              className="text-sm font-medium text-[var(--color-text)] hover:text-[var(--color-primary)] transition-colors"
            >
              Browse
            </Link>
          </nav>
        </div>

        {/* Right: Auth state */}
        <div className="flex items-center gap-3">
          {loading ? (
            <div className="w-8 h-8 rounded-full skeleton" />
          ) : user ? (
            <div className="flex items-center gap-3">
              <Link
                href="/dashboard"
                className="text-sm font-medium text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors"
              >
                Dashboard
              </Link>
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={user.avatarUrl}
                alt={user.name}
                className="w-8 h-8 rounded-full"
                referrerPolicy="no-referrer"
              />
            </div>
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
