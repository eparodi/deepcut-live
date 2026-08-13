// ============================================================
// Shared display-formatting helpers
// ============================================================

/** Format a count for display: 1205 → "1.2k", 1234567 → "1.2M" */
export function formatViewerCount(n: number): string {
  if (n >= 1_000_000) {
    const val = n / 1_000_000;
    return val % 1 === 0 ? `${val}M` : `${val.toFixed(1)}M`;
  }
  if (n >= 1_000) {
    const val = n / 1_000;
    return val % 1 === 0 ? `${val}k` : `${val.toFixed(1)}k`;
  }
  return n.toLocaleString();
}

/** Format duration in seconds to human-readable: 3661 → "1h 1m", 2700 → "45m", 30 → "30s" */
export function formatDuration(totalSeconds: number): string {
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  if (minutes > 0) {
    return `${minutes}m`;
  }
  return `${seconds}s`;
}

/**
 * Format duration without a seconds granularity: 3661 → "1h 1m", 30 → "0m".
 * Used where sub-minute precision is noise (e.g. analytics totals).
 */
export function formatDurationCoarse(totalSeconds: number): string {
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  return `${minutes}m`;
}
