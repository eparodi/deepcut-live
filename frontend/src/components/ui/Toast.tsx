"use client";
// Client Component — needs useState for auto-dismiss

import { useState, useEffect, useCallback } from "react";

interface ToastProps {
  message: string;
  variant?: "success" | "error";
  durationMs?: number;
  onDismiss: () => void;
}

export function Toast({
  message,
  variant = "success",
  durationMs = 3000,
  onDismiss,
}: ToastProps) {
  useEffect(() => {
    const timer = setTimeout(onDismiss, durationMs);
    return () => clearTimeout(timer);
  }, [durationMs, onDismiss]);

  const bgColor =
    variant === "success" ? "var(--color-primary)" : "var(--color-danger)";

  // Variant must never be conveyed by color alone (WCAG 1.4.1):
  // errors get role=alert (assertive) + ⚠️, successes role=status + ✓.
  const icon = variant === "success" ? "✓" : "⚠️";
  const role = variant === "success" ? "status" : "alert";
  const ariaLive = variant === "success" ? "polite" : "assertive";

  return (
    <div
      className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 px-5 py-3 rounded-lg text-white text-sm font-medium shadow-lg animate-fade-up"
      style={{ backgroundColor: bgColor }}
      role={role}
      aria-live={ariaLive}
    >
      <span aria-hidden="true">{icon}</span> {message}
    </div>
  );
}

/** Hook to manage toast state */
export function useToast() {
  const [toast, setToast] = useState<{
    message: string;
    variant: "success" | "error";
  } | null>(null);

  const showToast = useCallback(
    (message: string, variant: "success" | "error" = "success") => {
      setToast({ message, variant });
    },
    []
  );

  const dismissToast = useCallback(() => {
    setToast(null);
  }, []);

  const ToastComponent = toast ? (
    <Toast
      message={toast.message}
      variant={toast.variant}
      onDismiss={dismissToast}
    />
  ) : null;

  return { showToast, ToastComponent };
}
