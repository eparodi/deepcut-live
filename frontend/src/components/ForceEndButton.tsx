"use client";
// Client Component — needs state for dialog and API call

import { useState } from "react";
import { forceEndStream } from "@/lib/api";

interface ForceEndButtonProps {
  onEnded: () => void;
  onError: (message: string) => void;
}

export function ForceEndButton({ onEnded, onError }: ForceEndButtonProps) {
  const [showDialog, setShowDialog] = useState(false);
  const [loading, setLoading] = useState(false);

  async function handleConfirm() {
    setLoading(true);
    try {
      await forceEndStream();
      onEnded();
      setShowDialog(false);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to end stream.";
      onError(message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <button
        onClick={() => setShowDialog(true)}
        disabled={loading}
        className="inline-flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:opacity-90 disabled:opacity-50"
        style={{ backgroundColor: "var(--color-danger)" }}
      >
        ⏹ End Stream
      </button>

      {/* Confirmation Dialog */}
      {showDialog && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center p-4"
          role="dialog"
          aria-modal="true"
          aria-labelledby="end-stream-dialog-title"
        >
          {/* Backdrop */}
          <div
            className="absolute inset-0 bg-black/60"
            onClick={() => !loading && setShowDialog(false)}
          />

          {/* Dialog card */}
          <div
            className="relative w-full max-w-md rounded-xl p-6 shadow-2xl"
            style={{ backgroundColor: "var(--color-surface-raised)" }}
          >
            <h3
              id="end-stream-dialog-title"
              className="text-lg font-semibold text-[var(--color-text)]"
            >
              End Stream?
            </h3>
            <p className="mt-2 text-sm text-[var(--color-text-muted)] leading-relaxed">
              Your stream will end immediately. Viewers will see a &ldquo;Stream
              ended&rdquo; screen. A recording will be saved.
            </p>

            <div className="mt-6 flex gap-3 justify-end">
              <button
                onClick={() => setShowDialog(false)}
                disabled={loading}
                className="rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors hover:opacity-90"
                style={{ backgroundColor: "var(--color-primary)" }}
              >
                Keep Streaming
              </button>
              <button
                onClick={handleConfirm}
                disabled={loading}
                className="inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold text-white transition-colors hover:opacity-90 disabled:opacity-50"
                style={{ backgroundColor: "var(--color-danger)" }}
              >
                {loading ? (
                  <>
                    <Spinner />
                    Ending...
                  </>
                ) : (
                  "End Stream"
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

function Spinner() {
  return (
    <svg
      className="animate-spin h-4 w-4"
      viewBox="0 0 24 24"
      fill="none"
      aria-hidden="true"
    >
      <circle
        className="opacity-25"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        strokeWidth="4"
      />
      <path
        className="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
      />
    </svg>
  );
}
