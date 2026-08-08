import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, act, renderHook } from "@testing-library/react";
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

describe("useToast", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("starts with no toast", () => {
    const { result } = renderHook(() => useToast());

    expect(result.current.ToastComponent).toBeNull();
  });

  it("shows a success toast and dismisses after timeout", () => {
    const { result, rerender } = renderHook(() => useToast());

    act(() => {
      result.current.showToast("Done!", "success");
    });
    rerender();

    // ToastComponent is a React element, render it to query
    const { container: c1 } = render(result.current.ToastComponent);
    expect(c1).toHaveTextContent("Done!");

    // Advance past the auto-dismiss timeout
    act(() => {
      vi.advanceTimersByTime(3000);
    });
    rerender();

    expect(result.current.ToastComponent).toBeNull();
  });

  it("shows an error toast", () => {
    const { result, rerender } = renderHook(() => useToast());

    act(() => {
      result.current.showToast("Oops!", "error");
    });
    rerender();

    const { container } = render(result.current.ToastComponent);
    expect(container).toHaveTextContent("Oops!");
  });

  it("only shows one toast at a time", () => {
    const { result, rerender } = renderHook(() => useToast());

    act(() => {
      result.current.showToast("First", "success");
    });
    rerender();

    act(() => {
      result.current.showToast("Second", "error");
    });
    rerender();

    // Second toast replaces first
    const { container } = render(result.current.ToastComponent);
    expect(container).toHaveTextContent("Second");
    expect(container).not.toHaveTextContent("First");
  });

  it("showToast defaults variant to success", () => {
    const { result, rerender } = renderHook(() => useToast());

    act(() => {
      result.current.showToast("Default variant");
    });
    rerender();

    const { container } = render(result.current.ToastComponent);
    const el = container.firstElementChild as HTMLElement;
    expect(el.style.backgroundColor).toBe("var(--color-primary)");
  });
});
