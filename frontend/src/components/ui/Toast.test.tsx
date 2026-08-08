import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, act } from "@testing-library/react";
import { Toast, useToast } from "./Toast";

describe("Toast", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders the message", () => {
    const onDismiss = vi.fn();
    render(
      <Toast message="Copied!" variant="success" onDismiss={onDismiss} />
    );
    expect(screen.getByText("Copied!")).toBeInTheDocument();
  });

  it("calls onDismiss after the default duration (3s)", () => {
    const onDismiss = vi.fn();
    render(
      <Toast message="Saved" variant="success" onDismiss={onDismiss} />
    );

    // Should not have been called yet
    expect(onDismiss).not.toHaveBeenCalled();

    // Advance past default durationMs (3000)
    act(() => {
      vi.advanceTimersByTime(3000);
    });

    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it("calls onDismiss after a custom duration", () => {
    const onDismiss = vi.fn();
    render(
      <Toast
        message="Error"
        variant="error"
        durationMs={5000}
        onDismiss={onDismiss}
      />
    );

    act(() => {
      vi.advanceTimersByTime(4999);
    });
    expect(onDismiss).not.toHaveBeenCalled();

    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it("cleans up the timer on unmount", () => {
    const onDismiss = vi.fn();
    const { unmount } = render(
      <Toast message="Test" variant="success" onDismiss={onDismiss} />
    );

    unmount();

    // Advance past full duration — timer should have been cleared
    act(() => {
      vi.advanceTimersByTime(3000);
    });

    expect(onDismiss).not.toHaveBeenCalled();
  });

  it("applies success background color", () => {
    const onDismiss = vi.fn();
    render(
      <Toast message="OK" variant="success" onDismiss={onDismiss} />
    );
    const el = screen.getByRole("status");
    expect(el.style.backgroundColor).toBe("var(--color-primary)");
  });

  it("applies error background color", () => {
    const onDismiss = vi.fn();
    render(
      <Toast message="Fail" variant="error" onDismiss={onDismiss} />
    );
    const el = screen.getByRole("status");
    expect(el.style.backgroundColor).toBe("var(--color-danger)");
  });

  it("has aria-live polite for screen readers", () => {
    const onDismiss = vi.fn();
    render(
      <Toast message="Notification" variant="success" onDismiss={onDismiss} />
    );
    expect(screen.getByRole("status")).toHaveAttribute(
      "aria-live",
      "polite"
    );
  });
});

// ── useToast hook ──────────────────────────────────────────────────────

/** Test component that exercises the useToast hook */
function ToastConsumer({
  triggerRef,
}: {
  triggerRef: {
    current: null | ((msg: string, variant: "success" | "error") => void);
  };
}) {
  const { showToast, ToastComponent } = useToast();

  // Expose showToast to the test via ref
  triggerRef.current = showToast;

  return <div>{ToastComponent}</div>;
}

describe("useToast", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("shows a success toast and dismisses after timeout", () => {
    const triggerRef: { current: null | ((msg: string, variant: "success" | "error") => void) } = { current: null };
    render(<ToastConsumer triggerRef={triggerRef} />);

    // Trigger the toast
    act(() => {
      triggerRef.current?.("Done!", "success");
    });

    expect(screen.getByText("Done!")).toBeInTheDocument();

    // Advance past the auto-dismiss timeout
    act(() => {
      vi.advanceTimersByTime(3000);
    });

    expect(screen.queryByText("Done!")).not.toBeInTheDocument();
  });

  it("shows an error toast", () => {
    const triggerRef: { current: null | ((msg: string, variant: "success" | "error") => void) } = { current: null };
    render(<ToastConsumer triggerRef={triggerRef} />);

    act(() => {
      triggerRef.current?.("Oops!", "error");
    });

    expect(screen.getByText("Oops!")).toBeInTheDocument();
  });

  it("only shows one toast at a time", () => {
    const triggerRef: { current: null | ((msg: string, variant: "success" | "error") => void) } = { current: null };
    render(<ToastConsumer triggerRef={triggerRef} />);

    act(() => {
      triggerRef.current?.("First", "success");
    });
    expect(screen.getByText("First")).toBeInTheDocument();

    act(() => {
      triggerRef.current?.("Second", "error");
    });

    // Second toast replaces first
    expect(screen.queryByText("First")).not.toBeInTheDocument();
    expect(screen.getByText("Second")).toBeInTheDocument();
  });
});
