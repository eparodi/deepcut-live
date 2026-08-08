// ============================================================
// DeepCut Live — API Client
// Points to Go/chi backend at http://localhost:8081
// ============================================================

// Use same-origin URL; Next.js rewrites proxy /api/* to the Go backend.
// Absolute URL needed for SSR (server-side fetch has no base to resolve relatives).
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:3000";

export class ApiError extends Error {
  status: number;
  body: unknown;

  constructor(status: number, body: unknown) {
    const message =
      typeof body === "object" && body !== null && "error" in body
        ? (body as Record<string, string>).error
        : `API error ${status}`;
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }
}

interface RequestOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
}

/**
 * Base fetch wrapper for the Go/chi backend API.
 * All requests include credentials: "include" so HttpOnly JWT cookies
 * are sent automatically on cross-origin requests.
 */
export async function api<T = unknown>(
  path: string,
  options: RequestOptions = {}
): Promise<T> {
  const url = `${API_BASE_URL}${path}`;
  const headers: Record<string, string> = {
    ...(options.headers as Record<string, string>),
  };

  // Only set Content-Type for requests with a body
  if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
  }

  const res = await fetch(url, {
    ...options,
    headers,
    credentials: "include",
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
  });

  // Handle 204 No Content
  if (res.status === 204) {
    return undefined as T;
  }

  // Parse JSON (even for errors — the backend returns JSON errors)
  const data = await res.json().catch(() => null);

  if (!res.ok) {
    throw new ApiError(res.status, data);
  }

  return data as T;
}

// ============================================================
// Typed API helper functions
// ============================================================

import type {
  User,
  RegeneratedKey,
  StreamSettings,
  Analytics,
  StreamEndResponse,
  LiveStream,
} from "@/types";

/** GET /api/me — get current user profile + stream key */
export function getMe(): Promise<User> {
  return api<User>("/api/me");
}

/** POST /api/me/stream-key/regenerate — regenerate stream key */
export function regenerateStreamKey(): Promise<RegeneratedKey> {
  return api<RegeneratedKey>("/api/me/stream-key/regenerate", {
    method: "POST",
  });
}

/** PATCH /api/me/settings — update stream title + category */
export function updateStreamSettings(
  settings: StreamSettings
): Promise<{ streamTitle: string; streamCategory: string }> {
  return api("/api/me/settings", {
    method: "PATCH",
    body: settings,
  });
}

/** GET /api/me/analytics — get streamer analytics */
export function getAnalytics(
  period: "week" | "month" | "all" = "week"
): Promise<Analytics> {
  return api<Analytics>(`/api/me/analytics?period=${period}`);
}

/** POST /api/me/stream/end — force-end current stream */
export function forceEndStream(): Promise<StreamEndResponse> {
  return api<StreamEndResponse>("/api/me/stream/end", {
    method: "POST",
  });
}

/** GET /api/streams/live — get list of live streams */
export function getLiveStreams(): Promise<LiveStream[]> {
  return api<LiveStream[]>("/api/streams/live");
}
