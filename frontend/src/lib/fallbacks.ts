// Shared inline-SVG fallback images for <img> onError handling (AGENTS.md §10.3).
// Data URIs so the fallback never needs a network round-trip.

/** Dark 16:9 placeholder with a 🎬 clapper, used when thumbnails 404. */
export const THUMBNAIL_FALLBACK =
  "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='320' height='180'%3E%3Crect fill='%231a1a2e' width='320' height='180'/%3E%3Ctext fill='%234a4a6a' x='50%25' y='50%25' dominant-baseline='middle' text-anchor='middle' font-size='40'%3E%F0%9F%8E%AC%3C/text%3E%3C/svg%3E";

/** Generic user silhouette, used when streamer avatars 404. */
export const AVATAR_FALLBACK =
  "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='96' height='96'%3E%3Crect fill='%232a2a3e' width='96' height='96'/%3E%3Ccircle cx='48' cy='38' r='16' fill='%234a4a6a'/%3E%3Cpath d='M20 80c4-16 16-22 28-22s24 6 28 22' fill='%234a4a6a'/%3E%3C/svg%3E";
