"use client";
// Client Component — needs state for dialog and API call

import { useState, useCallback } from "react";
import { Spinner } from "@/components/ui/Spinner";
import { forceEndStream } from "@/lib/api";
import { useDialogFocus } from "@/lib/useDialogFocus";

interface ForceEndButtonProps {
  onEnded: () => void;
  onError: (message: string) => void;
}

export function ForceEndButton({ onEnded, onError }: ForceEndButtonProps) {
  const [showDialog, setShowDialog] = useState(false);
  const [loading, setLoading] = useState(false);

  const closeDialog = useCallback(() => setShowDialog(false), []);
  const dialogRef = useDialogFocus({
    open: showDialog,
    onClose: closeDialog,
    enabled: !loading,
  });

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
            onClick={() => !loading && closeDialog()}
          />

          {/* Dialog card */}
          <div
            ref={dialogRef}
            tabIndex={-1}
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
                onClick={closeDialog}
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

