"use client";
// Client Component — needs useState for copy interaction and toast

import { useCallback } from "react";
import { useToast } from "@/components/ui/Toast";

interface StreamKeyDisplayProps {
  streamKey: string;
}

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

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(streamKey);
      showToast("Stream key copied to clipboard!", "success");
    } catch {
      // Fallback for older browsers
      showToast("Could not copy. Please select and copy manually.", "error");
    }
  }, [streamKey, showToast]);

  return (
    <section aria-labelledby="key-heading">
      <h2
        id="key-heading"
        className="text-lg font-semibold text-[var(--color-text)] mb-4"
      >
        Stream Key
      </h2>

      <div
        className="flex items-center justify-between gap-3 px-4 py-3 rounded-lg border"
        style={{
          backgroundColor: "var(--color-surface)",
          borderColor: "var(--color-surface-raised)",
        }}
      >
        <code className="text-sm font-mono text-[var(--color-text)] select-all break-all">
          {maskKey(streamKey)}
        </code>
        <button
          onClick={handleCopy}
          className="shrink-0 inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors hover:opacity-80 text-white"
          style={{ backgroundColor: "var(--color-primary)" }}
          aria-label="Copy stream key to clipboard"
        >
          📋 Copy
        </button>
      </div>

      {ToastComponent}
    </section>
  );
}
