"use client";
// Client Component — needs useState for form state, toasts, interaction handlers.
// Data is fetched via React 19's `use()` hook with Suspense boundaries.

import { useState, useCallback, use } from "react";
import Link from "next/link";
import { Component, Suspense, type ReactNode } from "react";
import { getMe, getAnalytics } from "@/lib/api";
import type { User, Analytics } from "@/types";
import { StreamKeyDisplay } from "@/components/StreamKeyDisplay";
import { RegenerateKeyButton } from "@/components/RegenerateKeyButton";
import { StreamSettingsForm } from "@/components/StreamSettingsForm";
import { AnalyticsCards } from "@/components/AnalyticsCards";
import { ForceEndButton } from "@/components/ForceEndButton";
import { useToast } from "@/components/ui/Toast";

// ============================================================
// Cached promise helpers for React 19 `use()` data fetching.
// Avoids the setState-in-effect lint error.
// ============================================================

let userPromise: Promise<User> | null = null;
let analyticsPromise: Promise<Analytics> | null = null;

function fetchUser(): Promise<User> {
  if (!userPromise) userPromise = getMe();
  return userPromise;
}

function fetchAnalytics(): Promise<Analytics> {
  if (!analyticsPromise) analyticsPromise = getAnalytics("week");
  return analyticsPromise;
}

function resetDataPromises() {
  userPromise = null;
  analyticsPromise = null;
}

// ============================================================
// Dashboard Content (inner — unpacked from Suspense)
// ============================================================

function DashboardContent() {
  const user = use(fetchUser());
  const analytics = use(fetchAnalytics());

  const [currentUser, setCurrentUser] = useState(user);
  const [streamEnded, setStreamEnded] = useState(false);
  const { showToast, ToastComponent } = useToast();

  const handleRegeneratedKey = useCallback(
    (newKey: string) => {
      setCurrentUser((prev) => ({ ...prev, streamKey: newKey }));
      showToast("New stream key generated!", "success");
    },
    [showToast]
  );

  const handleSettingsSaved = useCallback(
    (title: string, category: string) => {
      setCurrentUser((prev) => ({
        ...prev,
        streamTitle: title,
        streamCategory: category,
      }));
      showToast("Settings saved!", "success");
    },
    [showToast]
  );

  const handleStreamEnded = useCallback(() => {
    setStreamEnded(true);
    setCurrentUser((prev) => ({ ...prev, isLive: false }));
  }, []);

  const handleError = useCallback(
    (message: string) => showToast(message, "error"),
    [showToast]
  );

  return (
    <DashboardLayout user={currentUser}>
      {streamEnded && (
        <div
          className="rounded-lg px-4 py-3 text-sm mb-6 text-white"
          style={{ backgroundColor: "var(--color-primary)" }}
        >
          Stream ended. Recording processing...
        </div>
      )}

      <div className="space-y-8">
        <StreamKeyDisplay streamKey={currentUser.streamKey} />

        <RegenerateKeyButton
          onRegenerated={handleRegeneratedKey}
          onError={handleError}
        />

        <hr
          className="border-0 h-px"
          style={{ backgroundColor: "var(--color-surface-raised)" }}
        />

        <StreamSettingsForm
          initialTitle={currentUser.streamTitle}
          initialCategory={currentUser.streamCategory}
          onSave={handleSettingsSaved}
          onError={handleError}
        />

        <hr
          className="border-0 h-px"
          style={{ backgroundColor: "var(--color-surface-raised)" }}
        />

        <AnalyticsCards
          analytics={analytics}
          loading={false}
          error={null}
          onRetry={resetDataPromises}
        />

        {currentUser.isLive && (
          <>
            <hr
              className="border-0 h-px"
              style={{ backgroundColor: "var(--color-surface-raised)" }}
            />
            <div className="flex justify-center">
              <ForceEndButton
                onEnded={handleStreamEnded}
                onError={handleError}
              />
            </div>
          </>
        )}
      </div>

      {ToastComponent}
    </DashboardLayout>
  );
}

// ============================================================
// Dashboard Page (outer — with Suspense + Error Boundary)
// ============================================================

export default function DashboardPage() {
  return (
    <DataErrorBoundary>
      <Suspense fallback={<DashboardSkeleton />}>
        <DashboardContent />
      </Suspense>
    </DataErrorBoundary>
  );
}

// ============================================================
// Error Boundary — catches fetch rejections from `use()`
// ============================================================

interface EBProps {
  children: ReactNode;
}
interface EBState {
  error: Error | null;
}

class DataErrorBoundary extends Component<EBProps, EBState> {
  constructor(props: EBProps) {
    super(props);
    this.state = { error: null };
  }

  static getDerivedStateFromError(error: Error): EBState {
    return { error };
  }

  render() {
    if (this.state.error) {
      return (
        <DashboardLayout>
          <div
            className="rounded-xl p-8 text-center"
            style={{ backgroundColor: "var(--color-surface-raised)" }}
          >
            <p className="text-[var(--color-text-muted)] text-lg">
              Could not load dashboard
            </p>
            <p
              className="mt-2 text-sm"
              style={{ color: "var(--color-danger)" }}
            >
              {this.state.error.message}
            </p>
            <button
              onClick={() => {
                resetDataPromises();
                this.setState({ error: null });
              }}
              className="mt-4 inline-flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:opacity-90"
              style={{ backgroundColor: "var(--color-primary)" }}
            >
              Retry
            </button>
          </div>
        </DashboardLayout>
      );
    }

    return <>{this.props.children}</>;
  }
}

// ============================================================
// Loading skeleton (shown while Suspense waits for data)
// ============================================================

function DashboardSkeleton() {
  return (
    <DashboardLayout>
      <div className="space-y-8 animate-pulse">
        <section>
          <div className="skeleton h-5 w-24 mb-4" />
          <div className="skeleton h-14 w-full rounded-lg" />
        </section>
        <div className="skeleton h-10 w-40 rounded-lg" />
        <hr
          className="border-0 h-px"
          style={{ backgroundColor: "var(--color-surface-raised)" }}
        />
        <section>
          <div className="skeleton h-5 w-32 mb-4" />
          <div className="space-y-4">
            <div className="skeleton h-12 w-full rounded-lg" />
            <div className="skeleton h-12 w-full rounded-lg" />
            <div className="skeleton h-10 w-20 rounded-lg" />
          </div>
        </section>
        <hr
          className="border-0 h-px"
          style={{ backgroundColor: "var(--color-surface-raised)" }}
        />
        <section>
          <div className="skeleton h-5 w-40 mb-4" />
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            {[0, 1, 2, 3].map((i) => (
              <div key={i} className="skeleton h-24 rounded-xl" />
            ))}
          </div>
        </section>
      </div>
    </DashboardLayout>
  );
}

// ============================================================
// Dashboard layout wrapper with header
// ============================================================

function DashboardLayout({
  user,
  children,
}: {
  user?: User;
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-full flex flex-col">
      <header
        className="w-full border-b"
        style={{ borderColor: "var(--color-surface-raised)" }}
      >
        <div className="max-w-5xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-6">
            <Link
              href="/"
              className="font-bold text-xl"
              style={{ color: "var(--color-primary)" }}
            >
              DeepCut
            </Link>
            <span className="text-sm text-[var(--color-text-muted)]">
              Dashboard
            </span>
          </div>

          {user && (
            <div className="flex items-center gap-3">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={user.avatarUrl}
                alt={user.name}
                className="w-8 h-8 rounded-full"
                referrerPolicy="no-referrer"
              />
              <span className="text-sm text-[var(--color-text)] hidden sm:inline">
                {user.name}
              </span>
            </div>
          )}
        </div>
      </header>

      <main className="flex-1 max-w-3xl mx-auto w-full px-6 py-8">
        {children}
      </main>
    </div>
  );
}
