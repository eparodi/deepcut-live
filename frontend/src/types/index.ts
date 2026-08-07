// ============================================================
// DeepCut Live — Shared TypeScript Types
// Derived from API contracts in specs/
// ============================================================

/** User profile returned by GET /api/me */
export interface User {
  id: string;
  name: string;
  email: string;
  avatarUrl: string;
  streamKey: string;
  streamTitle: string | null;
  streamCategory: string | null;
  isLive: boolean;
}

/** Stream key returned by POST /api/me/stream-key/regenerate */
export interface RegeneratedKey {
  streamKey: string;
}

/** Settings payload for PATCH /api/me/settings */
export interface StreamSettings {
  streamTitle: string;
  streamCategory?: string;
}

/** Analytics returned by GET /api/me/analytics */
export interface Analytics {
  period: "week" | "month" | "all";
  startDate: string;
  endDate: string;
  totalStreamTimeSeconds: number;
  peakViewers: number;
  totalUniqueViewers: number;
  totalStreams: number;
}

/** Force-end response from POST /api/me/stream/end */
export interface StreamEndResponse {
  status: "offline";
  message: string;
}

/** Live stream info returned by GET /api/streams/live */
export interface LiveStream {
  userId: string;
  streamerName: string;
  streamerAvatarUrl: string;
  streamId: string;
  title: string;
  category: string | null;
  viewerCount: number;
  thumbnailUrl: string | null;
  startedAt: string;
}

/** API error response */
export interface ApiError {
  error: string;
  message?: string;
}
