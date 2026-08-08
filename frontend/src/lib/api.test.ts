import { describe, it, expect, vi, beforeEach } from "vitest";
import { ApiError, api } from "./api";

// Mock global fetch
const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

// api.ts uses NEXT_PUBLIC_API_URL env var with this fallback
const BASE_URL = "http://localhost:3000";

describe("ApiError", () => {
  it("extracts error message from JSON body", () => {
    const err = new ApiError(404, { error: "User not found" });
    expect(err.message).toBe("User not found");
    expect(err.status).toBe(404);
    expect(err.name).toBe("ApiError");
  });

  it("falls back to generic message when body has no error field", () => {
    const err = new ApiError(500, { detail: "something" });
    expect(err.message).toBe("API error 500");
    expect(err.status).toBe(500);
  });

  it("falls back to generic message when body is not an object", () => {
    const err = new ApiError(502, "Bad Gateway");
    expect(err.message).toBe("API error 502");
  });
});

describe("api", () => {
  beforeEach(() => {
    mockFetch.mockReset();
  });

  it("sends GET request and returns parsed JSON", async () => {
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify({ id: "1", name: "Alice" }), {
        status: 200,
      })
    );

    const result = await api("/api/me");

    expect(mockFetch).toHaveBeenCalledWith(
      `${BASE_URL}/api/me`,
      expect.objectContaining({
        credentials: "include",
      })
    );
    expect(result).toEqual({ id: "1", name: "Alice" });
  });

  it("sends POST with JSON body and Content-Type header", async () => {
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify({ ok: true }), { status: 201 })
    );

    await api("/api/users", { method: "POST", body: { name: "Bob" } });

    const [, options] = mockFetch.mock.calls[0];
    expect(options.method).toBe("POST");
    expect(options.headers).toHaveProperty("Content-Type", "application/json");
    expect(options.body).toBe(JSON.stringify({ name: "Bob" }));
  });

  it("does not set Content-Type for GET requests (no body)", async () => {
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify([]), { status: 200 })
    );

    await api("/api/streams/live");

    const [, options] = mockFetch.mock.calls[0];
    expect(options.headers).not.toHaveProperty("Content-Type");
  });

  it("returns undefined for 204 No Content", async () => {
    mockFetch.mockResolvedValueOnce(new Response(null, { status: 204 }));

    const result = await api("/api/me/stream/end", { method: "POST" });

    expect(result).toBeUndefined();
  });

  it("throws ApiError on non-2xx response with JSON body", async () => {
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify({ error: "Unauthorized" }), { status: 401 })
    );

    await expect(api("/api/me")).rejects.toMatchObject({
      message: "Unauthorized",
      status: 401,
    });
  });

  it("throws ApiError on non-2xx response with non-JSON body", async () => {
    mockFetch.mockResolvedValueOnce(
      new Response("plain text error", { status: 500 })
    );

    await expect(api("/api/me")).rejects.toMatchObject({
      message: "API error 500",
      status: 500,
    });
  });

  it("passes through custom headers", async () => {
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify({ ok: true }), { status: 200 })
    );

    await api("/api/me", {
      headers: { "X-Custom": "value" },
    });

    const [, options] = mockFetch.mock.calls[0];
    expect(options.headers).toHaveProperty("X-Custom", "value");
  });
});
