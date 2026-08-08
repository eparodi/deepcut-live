"use client";
// Client Component — needs useState for copy interaction and toast

import { useCallback } from "react";
import { useToast } from "@/components/ui/Toast";

interface StreamKeyDisplayProps {
  streamKey?: string;
}

const RTMP_URL = process.env.NEXT_PUBLIC_RTMP_URL || "rtmp://localhost:1935/live";

/** Mask the middle portion of the stream key for display */
function maskKey(key: string): string {
  if (key.length <= 12) return key;
  const prefix = key.slice(0, 6);
  const suffix = key.slice(-6);
  const maskedLength = key.length - 12;
  return `${prefix}${"•".repeat(Math.min(maskedLength, 20))}${suffix}`;
}

export function StreamKeyDisplay({ streamKey }: StreamKeyDisplayProps) {
  const { showToast, ToastComponent } = useToast();

  const handleCopyKey = useCallback(async () => {
    if (!streamKey) return;
    try {
      await navigator.clipboard.writeText(streamKey);
      showToast("Stream key copied to clipboard!", "success");
    } catch {
      showToast("Could not copy. Please select and copy manually.", "error");
    }
  }, [streamKey, showToast]);

  const handleCopyURL = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(RTMP_URL);
      showToast("Server URL copied to clipboard!", "success");
    } catch {
      showToast("Could not copy. Please select and copy manually.", "error");
    }
  }, [showToast]);

  return (
    <section aria-labelledby="key-heading">
      <h2
        id="key-heading"
        className="text-lg font-semibold text-[var(--color-text)] mb-4"
      >
        Stream Settings
      </h2>

      {/* Server URL */}
      <div className="mb-3">
        <label className="block text-xs font-medium text-[var(--color-text-muted)] mb-1.5 uppercase tracking-wider">
          Server
        </label>
        <div
          className="flex items-center justify-between gap-3 px-4 py-3 rounded-lg border"
          style={{
            backgroundColor: "var(--color-surface)",
            borderColor: "var(--color-surface-raised)",
          }}
        >
          <code className="text-sm font-mono text-[var(--color-text)] select-all">
            {RTMP_URL}
          </code>
          <button
            onClick={handleCopyURL}
            className="shrink-0 inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors hover:opacity-80 text-white"
            style={{ backgroundColor: "var(--color-primary)" }}
            aria-label="Copy server URL to clipboard"
          >
            📋 Copy
          </button>
        </div>
      </div>

      {/* Stream Key */}
      <div>
        <label className="block text-xs font-medium text-[var(--color-text-muted)] mb-1.5 uppercase tracking-wider">
          Stream Key
        </label>
        <div
          className="flex items-center justify-between gap-3 px-4 py-3 rounded-lg border"
          style={{
            backgroundColor: "var(--color-surface)",
            borderColor: "var(--color-surface-raised)",
          }}
        >
          <code className="text-sm font-mono text-[var(--color-text)] select-all break-all">
            {streamKey ? maskKey(streamKey) : (
              <span className="text-[var(--color-text-muted)]">
                No stream key — regenerate to create one
              </span>
            )}
          </code>
          <button
            onClick={handleCopyKey}
            disabled={!streamKey}
            className="shrink-0 inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors hover:opacity-80 text-white disabled:opacity-40 disabled:cursor-not-allowed"
            style={{ backgroundColor: "var(--color-primary)" }}
            aria-label="Copy stream key to clipboard"
          >
            📋 Copy
          </button>
        </div>
      </div>

      {/* OBS Quick Setup */}
      {streamKey && (
        <p className="mt-3 text-xs text-[var(--color-text-muted)] leading-relaxed">
          In OBS, go to <strong>Settings → Stream</strong>, set Service to{" "}
          <strong>Custom</strong>, paste the Server URL and Stream Key above.
        </p>
      )}

      {ToastComponent}
    </section>
  );
}
