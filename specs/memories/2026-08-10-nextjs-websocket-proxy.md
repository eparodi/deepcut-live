# Learning: Next.js Cannot Proxy WebSocket Connections

**Date:** 2026-08-10
**Discovered during:** Chat implementation (US4)

## The Problem

Next.js's built-in proxy (via `next.config.ts` rewrites or middleware) handles HTTP requests but does **not** support WebSocket upgrade proxying. When the frontend tries to open a WebSocket connection through the Next.js server (port 3000), the connection fails with:

```
WebSocket connection failed: WebSocket is closed before the connection is established.
```

This happens because Next.js's internal HTTP server (or the rewrite middleware) doesn't forward the `Upgrade: websocket` header to the backend.

## The Pattern

WebSocket URLs must **always** point directly to the backend, bypassing the Next.js proxy:

```typescript
// ❌ Wrong: goes through Next.js proxy (port 3000)
const wsUrl = `ws://localhost:3000/ws/chat/${streamId}`;

// ✅ Correct: goes directly to backend (port 8081)
const wsUrl = `ws://localhost:8081/ws/chat/${streamId}`;
```

## Solution

Use a separate environment variable for the WebSocket host, distinct from the API proxy URL:

```typescript
const API_HOST = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";  // REST calls
const WS_HOST  = process.env.NEXT_PUBLIC_WS_URL  || "http://localhost:8081";  // WebSocket calls

function getWsUrl(streamId: string): string {
  const url = new URL(WS_HOST);
  const protocol = url.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${url.host}/ws/chat/${streamId}`;
}
```

In production with a reverse proxy (nginx, Cloudflare), the proxy handles WebSocket upgrades at the edge, so both `API_HOST` and `WS_HOST` can point to the same origin. But in local development, the Next.js dev server cannot proxy WebSockets, so `WS_HOST` must point to the backend directly.

## Rule to Add

Add to the `nextjs` skill:

> **WebSocket connections must bypass the Next.js proxy.** Use a separate `NEXT_PUBLIC_WS_URL` environment variable for WebSocket URLs. In local dev, this points directly to the backend (e.g., `http://localhost:8081`). In production behind a reverse proxy that supports WS upgrades, it can match the API URL.
