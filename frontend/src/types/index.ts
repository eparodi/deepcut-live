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
  streamKey?: string;
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

/** Live streams list response from GET /api/streams/live */
export interface LiveStreamsResponse {
  streams: LiveStream[];
  total: number;
}

/** Channel info returned by GET /api/channel/:userId */
export interface ChannelResponse {
  userId: string;
  streamerName: string;
  streamerAvatarUrl: string;
  streamTitle: string;
  streamCategory: string | null;
  isLive: boolean;
  viewerCount: number;
  hlsUrl: string | null;
  thumbnailUrl: string | null;
  startedAt: string | null;
  streamId: string | null;
}

/** Chat message (used in both WS and HTTP responses) */
export interface ChatMessage {
  id: string;
  userId: string;
  userName: string;
  userAvatarUrl: string;
  message: string;
  sentAt: string;
}

/** Chat history response from GET /api/chat/:streamId/messages */
export interface ChatMessagesResponse {
  messages: ChatMessage[];
  hasMore: boolean;
}

/** VOD item from GET /api/vods and GET /api/channel/:userId/vods */
export interface VodItem {
  id: string;
  userId: string;
  userName: string;
  userAvatar: string | null;
  title: string | null;
  startedAt: string;
  endedAt: string | null;
  durationSeconds: number | null;
  peakViewers: number;
  totalViewers: number;
  recordingPath: string | null;
  recordingStatus: "ready" | "processing" | "failed" | "pending";
  createdAt: string;
}

/** VOD list response from GET /api/channel/:userId/vods */
export type VodsResponse = VodItem[];

/** VOD detail from GET /api/vods/:vodId */
export interface VodDetail {
  id: string;
  userId: string;
  userName: string;
  userAvatar: string | null;
  title: string | null;
  startedAt: string;
  endedAt: string | null;
  durationSeconds: number | null;
  peakViewers: number;
  totalViewers: number;
  recordingPath: string | null;
  recordingStatus: "ready" | "processing" | "failed" | "pending";
  createdAt: string;
  /** HLS playback URL (derived from recordingPath by the backend or frontend) */
  hlsUrl: string | null;
  /** Error message when recordingStatus is "failed" */
  message?: string;
}

/** Search response from GET /api/vods */
export interface SearchResponse {
  vods: VodItem[];
  totalCount: number;
  limit: number;
  offset: number;
}

/** API error response */
export interface ApiError {
  error: string;
  message?: string;
}
