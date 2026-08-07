"use client";
// Client Component — needs useState for input values, form submission, event handlers

import { useState, type FormEvent } from "react";
import { updateStreamSettings } from "@/lib/api";

interface StreamSettingsFormProps {
  initialTitle: string | null;
  initialCategory: string | null;
  onSave: (title: string, category: string) => void;
  onError: (message: string) => void;
}

export function StreamSettingsForm({
  initialTitle,
  initialCategory,
  onSave,
  onError,
}: StreamSettingsFormProps) {
  const [title, setTitle] = useState(initialTitle || "");
  const [category, setCategory] = useState(initialCategory || "");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isDirty =
    title !== (initialTitle || "") || category !== (initialCategory || "");

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();

    // Client-side validation: title required, 1-100 chars
    const trimmedTitle = title.trim();
    if (!trimmedTitle) {
      setError("Stream title is required.");
      return;
    }
    if (trimmedTitle.length > 100) {
      setError("Stream title must be 100 characters or fewer.");
      return;
    }
    if (category.trim().length > 100) {
      setError("Category must be 100 characters or fewer.");
      return;
    }

    setError(null);
    setSaving(true);

    try {
      await updateStreamSettings({
        streamTitle: trimmedTitle,
        streamCategory: category.trim() || undefined,
      });
      onSave(trimmedTitle, category.trim());
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to save settings.";
      setError(message);
      onError(message);
    } finally {
      setSaving(false);
    }
  }

  return (
    <section aria-labelledby="settings-heading">
      <h2
        id="settings-heading"
        className="text-lg font-semibold text-[var(--color-text)] mb-4"
      >
        Stream Settings
      </h2>

      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Title input */}
        <div>
          <label
            htmlFor="stream-title"
            className="block text-sm text-[var(--color-text-muted)] mb-1"
          >
            Title
          </label>
          <input
            id="stream-title"
            type="text"
            value={title}
            onChange={(e) => {
              setTitle(e.target.value);
              if (error) setError(null);
            }}
            maxLength={100}
            placeholder="What are you streaming?"
            disabled={saving}
            className="w-full px-4 py-3 rounded-lg border text-[var(--color-text)] placeholder-[var(--color-text-muted)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:opacity-50"
            style={{
              backgroundColor: "var(--color-surface)",
              borderColor: error ? "var(--color-danger)" : "var(--color-surface-raised)",
            }}
          />
          {error && (
            <p className="mt-1 text-sm" style={{ color: "var(--color-danger)" }} role="alert">
              {error}
            </p>
          )}
        </div>

        {/* Category input */}
        <div>
          <label
            htmlFor="stream-category"
            className="block text-sm text-[var(--color-text-muted)] mb-1"
          >
            Category
          </label>
          <input
            id="stream-category"
            type="text"
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            maxLength={100}
            placeholder="e.g., Programming, Music, Gaming"
            disabled={saving}
            className="w-full px-4 py-3 rounded-lg border focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:opacity-50 text-[var(--color-text)] placeholder-[var(--color-text-muted)]"
            style={{
              backgroundColor: "var(--color-surface)",
              borderColor: "var(--color-surface-raised)",
            }}
          />
        </div>

        {/* Save button */}
        <button
          type="submit"
          disabled={saving || !isDirty}
          className="inline-flex items-center gap-2 rounded-lg px-6 py-2.5 text-sm font-semibold transition-colors disabled:opacity-40 text-white"
          style={{ backgroundColor: "var(--color-primary)" }}
        >
          {saving ? (
            <>
              <Spinner />
              Saving...
            </>
          ) : (
            "💾 Save"
          )}
        </button>
      </form>
    </section>
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
