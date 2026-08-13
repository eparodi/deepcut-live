"use client";
// Client Component — data fetched on client mount to avoid SSR auth cookie issues.

import { useState, useCallback, useEffect, useRef, Component } from "react";
import { type ReactNode } from "react";
import { getMe, getAnalytics } from "@/lib/api";
import type { User, Analytics } from "@/types";
import { StreamKeyDisplay } from "@/components/StreamKeyDisplay";
import { RegenerateKeyButton } from "@/components/RegenerateKeyButton";
import { GoLivePreview } from "@/components/GoLivePreview";
import { StreamSettingsForm } from "@/components/StreamSettingsForm";
import { AnalyticsCards } from "@/components/AnalyticsCards";
import { ForceEndButton } from "@/components/ForceEndButton";
import { useToast } from "@/components/ui/Toast";

// ============================================================
// Dashboard Content
// ============================================================

function DashboardContent() {
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [analytics, setAnalytics] = useState<Analytics | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [streamEnded, setStreamEnded] = useState(false);
  const wasLiveRef = useRef(false);
  const { showToast, ToastComponent } = useToast();

  // Fetch data — isPoll=true means silent background refresh (no loading spinner,
  // no error state change, ignores transient errors).
  const fetchData = useCallback((isPoll = false) => {
    if (!isPoll) {
      setLoading(true);
    }
    if (!isPoll) {
      setLoadError(null);
    }
    Promise.all([getMe(), getAnalytics("week")])
      .then(([u, a]) => {
        // Detect transition to live and notify the user.
        if (!wasLiveRef.current && u.isLive) {
          showToast("You are now live! 🎥", "success");
        }
        wasLiveRef.current = u.isLive;
        setCurrentUser(u);
        setAnalytics(a);
      })
      .catch((e) => {
        // On polls, silently ignore transient errors — keep the UI running.
        if (!isPoll) {
          setLoadError(e instanceof Error ? e.message : "Failed to load");
        }
      })
      .finally(() => {
        if (!isPoll) {
          setLoading(false);
        }
      });
  }, [showToast]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- fetch-on-mount is the intended pattern
    fetchData();
  }, [fetchData]);

  // WebSocket — connects directly to the backend. Next.js rewrites don't
  // proxy WebSocket upgrades, but cross-port cookies on localhost are sent
  // because SameSite=Lax treats all localhost ports as same-site.
  useEffect(() => {
    const wsHost = process.env.NEXT_PUBLIC_WS_URL || "http://localhost:8081";
    const wsBase = new URL(wsHost);
    const wsProtocol = wsBase.protocol === "https:" ? "wss:" : "ws:";
    const wsURL = `${wsProtocol}//${wsBase.host}/api/streams/ws`;

    let ws: WebSocket | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout>;
    let reconnectAttempt = 0;
    // Set on unmount so a late onclose can't schedule a reconnect — the
    // browser fires onclose after cleanup runs ws.close().
    let disposed = false;

    function connect() {
      ws = new WebSocket(wsURL);

      ws.onopen = () => {
        reconnectAttempt = 0;
        // Refresh immediately on connect to get current state.
        fetchData(true);
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (data.type === "streamStarted" || data.type === "streamEnded") {
            // Refresh user + analytics to get the updated isLive status.
            fetchData(true);
          }
        } catch {
          // Ignore malformed messages.
        }
      };

      ws.onclose = () => {
        if (disposed) return;
        // Reconnect with exponential backoff (1s → 30s cap, max 10 tries);
        // the 30s poll below keeps data fresh if we give up.
        if (reconnectAttempt < 10) {
          const delay = Math.min(1000 * 2 ** reconnectAttempt, 30_000);
          reconnectAttempt += 1;
          reconnectTimer = setTimeout(connect, delay);
        }
      };

      ws.onerror = () => {
        ws?.close();
      };
    }

    connect();

    return () => {
      disposed = true;
      clearTimeout(reconnectTimer);
      if (ws) {
        ws.onclose = null;
        ws.onerror = null;
        ws.onmessage = null;
        ws.close();
      }
    };
  }, [fetchData]);

  // Poll every 30s as a fallback if WebSocket disconnects.
  useEffect(() => {
    const interval = setInterval(() => fetchData(true), 30_000);
    return () => clearInterval(interval);
  }, [fetchData]);

  const handleRegeneratedKey = useCallback(
    (newKey: string) => {
      setCurrentUser((prev) =>
        prev ? { ...prev, streamKey: newKey } : prev
      );
      showToast("New stream key generated!", "success");
    },
    [showToast]
  );

  const handleSettingsSaved = useCallback(
    (title: string, category: string) => {
      setCurrentUser((prev) =>
        prev
          ? { ...prev, streamTitle: title, streamCategory: category }
          : prev
      );
      showToast("Settings saved!", "success");
    },
    [showToast]
  );

  const handleStreamEnded = useCallback(() => {
    setStreamEnded(true);
    setCurrentUser((prev) => (prev ? { ...prev, isLive: false } : prev));
    wasLiveRef.current = false;
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
          <p className="text-[var(--color-text-muted)] text-lg">
            Could not load dashboard
          </p>
          <p className="mt-2 text-sm" style={{ color: "var(--color-danger-text)" }}>
            {loadError}
          </p>
          <button
            onClick={() => fetchData()}
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
    <DashboardLayout>
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

        <GoLivePreview userId={currentUser.id} isLive={currentUser.isLive} />

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
          onRetry={() => fetchData()}
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
// Dashboard Page (outer — with Error Boundary)
// ============================================================

export default function DashboardPage() {
  return (
    <DataErrorBoundary>
      <DashboardContent />
    </DataErrorBoundary>
  );
}

// ============================================================
// Error Boundary — catches fetch rejections
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
              style={{ color: "var(--color-danger-text)" }}
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
// Loading skeleton
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
// Dashboard layout wrapper (no header — Navbar handles it)
// ============================================================

function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <main className="flex-1 max-w-3xl mx-auto w-full px-6 py-8">
      <h1 className="text-2xl font-bold text-[var(--color-text)] mb-6">
        Dashboard
      </h1>
      {children}
    </main>
  );
}
