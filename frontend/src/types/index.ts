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

/** VOD item from GET /api/channel/:userId/vods */
export interface VodItem {
  id: string;
  title: string;
  category: string | null;
  startedAt: string;
  durationSeconds: number;
  thumbnailUrl: string | null;
  status: "ready" | "processing" | "failed";
}

/** VOD list response from GET /api/channel/:userId/vods */
export interface VodsResponse {
  vods: VodItem[];
  total: number;
  page: number;
}

/** VOD detail from GET /api/vods/:vodId */
export interface VodDetail {
  id: string;
  title: string;
  category: string | null;
  streamerId: string;
  streamerName: string;
  streamerAvatarUrl: string;
  startedAt: string;
  durationSeconds: number;
  hlsUrl: string | null;
  viewerCount: number;
  status: "ready" | "processing" | "failed";
  message?: string;
}

/** Search result item from GET /api/search */
export interface SearchResult {
  vodId: string;
  title: string;
  streamerName: string;
  streamerAvatarUrl: string;
  startedAt: string;
  durationSeconds: number;
  thumbnailUrl: string | null;
}

/** Search response from GET /api/search */
export interface SearchResponse {
  results: SearchResult[];
  total: number;
  page: number;
}

/** API error response */
export interface ApiError {
  error: string;
  message?: string;
}
