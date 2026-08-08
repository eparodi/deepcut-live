"use client";
// Client Component — needs useState for dialog state and API call

import { useState } from "react";
import { Spinner } from "@/components/ui/Spinner";
import { regenerateStreamKey } from "@/lib/api";

interface RegenerateKeyButtonProps {
  onRegenerated: (newKey: string) => void;
  onError: (message: string) => void;
}

export function RegenerateKeyButton({
  onRegenerated,
  onError,
}: RegenerateKeyButtonProps) {
  const [showDialog, setShowDialog] = useState(false);
  const [loading, setLoading] = useState(false);

  async function handleConfirm() {
    setLoading(true);
    try {
      const result = await regenerateStreamKey();
      onRegenerated(result.streamKey);
      setShowDialog(false);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to regenerate key.";
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
        className="inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-colors hover:opacity-80"
        style={{
          backgroundColor: "var(--color-surface-raised)",
          color: "var(--color-text)",
        }}
      >
        🔄 Regenerate Key
      </button>

      {/* Confirmation Dialog */}
      {showDialog && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center p-4"
          role="dialog"
          aria-modal="true"
          aria-labelledby="regenerate-dialog-title"
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
              id="regenerate-dialog-title"
              className="text-lg font-semibold text-[var(--color-text)]"
            >
              Regenerate Stream Key?
            </h3>
            <p className="mt-2 text-sm text-[var(--color-text-muted)] leading-relaxed">
              Your current stream key will stop working immediately. If OBS is
              streaming, it will disconnect on the next restart. Update OBS with
              the new key.
            </p>

            <div className="mt-6 flex gap-3 justify-end">
              <button
                onClick={() => setShowDialog(false)}
                disabled={loading}
                className="rounded-lg px-4 py-2 text-sm font-medium transition-colors hover:opacity-80"
                style={{
                  backgroundColor: "var(--color-surface)",
                  color: "var(--color-text)",
                }}
              >
                Keep Current Key
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
                    Regenerating...
                  </>
                ) : (
                  "Regenerate"
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

