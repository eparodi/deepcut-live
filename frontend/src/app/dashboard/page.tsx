"use client";
// Client Component — data fetched on client mount to avoid SSR auth cookie issues.

import { useState, useCallback, useEffect, Component } from "react";
import Link from "next/link";
import { type ReactNode } from "react";
import { getMe, getAnalytics } from "@/lib/api";
import type { User, Analytics } from "@/types";
import { StreamKeyDisplay } from "@/components/StreamKeyDisplay";
import { RegenerateKeyButton } from "@/components/RegenerateKeyButton";
import { StreamSettingsForm } from "@/components/StreamSettingsForm";
import { AnalyticsCards } from "@/components/AnalyticsCards";
import { ForceEndButton } from "@/components/ForceEndButton";
import { useToast } from "@/components/ui/Toast";

// ============================================================
// Dashboard Content
// ============================================================

function DashboardContent() {
  // All hooks at the top — React requires consistent order across renders.
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [analytics, setAnalytics] = useState<Analytics | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [streamEnded, setStreamEnded] = useState(false);
  const { showToast, ToastComponent } = useToast();

  // Fetch data on client mount only — SSR has no auth cookies.
  const fetchData = useCallback(() => {
    setLoading(true);
    setLoadError(null);
    Promise.all([getMe(), getAnalytics("week")])
      .then(([u, a]) => {
        setCurrentUser(u);
        setAnalytics(a);
      })
      .catch((e) => setLoadError(e instanceof Error ? e.message : "Failed to load"))
      .finally(() => setLoading(false));
  }, []);

  	useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- fetch-on-mount is the intended pattern
    fetchData();
  }, [fetchData]);

  const handleRegeneratedKey = useCallback(
    (newKey: string) => {
      setCurrentUser((prev) => prev ? { ...prev, streamKey: newKey } : prev);
      showToast("New stream key generated!", "success");
    },
    [showToast]
  );

  const handleSettingsSaved = useCallback(
    (title: string, category: string) => {
      setCurrentUser((prev) =>
        prev ? { ...prev, streamTitle: title, streamCategory: category } : prev
      );
      showToast("Settings saved!", "success");
    },
    [showToast]
  );

  const handleStreamEnded = useCallback(() => {
    setStreamEnded(true);
    setCurrentUser((prev) => (prev ? { ...prev, isLive: false } : prev));
  }, []);

  const handleError = useCallback(
    (message: string) => showToast(message, "error"),
    [showToast]
  );

  // Early returns AFTER all hooks — all renders call the same hooks.
  if (loading) return <DashboardSkeleton />;

  if (loadError) {
    return (
      <DashboardLayout>
        <div
          className="rounded-xl p-8 text-center"
          style={{ backgroundColor: "var(--color-surface-raised)" }}
        >
          <p className="text-[var(--color-text-muted)] text-lg">Could not load dashboard</p>
          <p className="mt-2 text-sm" style={{ color: "var(--color-danger)" }}>{loadError}</p>
          <button
            onClick={fetchData}
            className="mt-4 inline-flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:opacity-90"
            style={{ backgroundColor: "var(--color-primary)" }}
          >
            Retry
          </button>
        </div>
      </DashboardLayout>
    );
  }

  if (!currentUser) return null;

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
          onRetry={fetchData}
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
      <DashboardContent />
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
                this.setState({ error: null });
                window.location.reload();
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
