"use client";

import { useEffect, useRef } from "react";

const FOCUSABLE_SELECTOR =
  'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

/**
 * Keyboard + focus management for a minimal confirm dialog (no deps):
 * - moves focus into the dialog when it opens (least destructive control
 *   first is preferred; callers should order buttons accordingly),
 * - traps Tab / Shift+Tab within the dialog,
 * - closes on Escape while `enabled`,
 * - restores focus to the trigger element when the dialog closes.
 *
 * Pass `enabled={false}` while a request is in flight to keep the dialog
 * modal-consistent (no Escape close mid-request, mirroring the backdrop).
 */
export function useDialogFocus({
  open,
  onClose,
  enabled = true,
}: {
  open: boolean;
  onClose: () => void;
  enabled?: boolean;
}) {
  const triggerRef = useRef<HTMLElement | null>(null);
  const dialogRef = useRef<HTMLDivElement | null>(null);

  // Remember the element that opened the dialog (the trigger).
  useEffect(() => {
    if (open) {
      triggerRef.current = document.activeElement as HTMLElement | null;
    }
  }, [open]);

  // Initial focus on open.
  useEffect(() => {
    if (!open || !dialogRef.current) return;
    const dialog = dialogRef.current;
    const first = dialog.querySelector<HTMLElement>(FOCUSABLE_SELECTOR);
    (first ?? dialog).focus();
  }, [open]);

  // Escape close + Tab trap while open.
  useEffect(() => {
    if (!open) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        if (enabled) {
          e.preventDefault();
          onClose();
        }
        return;
      }
      if (e.key !== "Tab" || !dialogRef.current) return;
      const dialog = dialogRef.current;
      const focusables = Array.from(
        dialog.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)
      );
      if (focusables.length === 0) {
        e.preventDefault();
        dialog.focus();
        return;
      }
      const first = focusables[0];
      const last = focusables[focusables.length - 1];
      const active = document.activeElement;
      const inside = active instanceof Node && dialog.contains(active);
      if (e.shiftKey) {
        if (active === first || !inside) {
          e.preventDefault();
          last.focus();
        }
      } else if (active === last || !inside) {
        e.preventDefault();
        first.focus();
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, enabled, onClose]);

  // Restore focus to the trigger when the dialog closes.
  useEffect(() => {
    if (!open) {
      triggerRef.current?.focus();
    }
  }, [open]);

  return dialogRef;
}
