import { describe, it, expect } from "vitest";
import { NextRequest } from "next/server";
import { proxy } from "./proxy";

describe("proxy middleware", () => {
  const createRequest = (pathname: string, token?: string) => {
    const url = `http://localhost:3000${pathname}`;
    const headers = new Headers();
    if (token) {
      headers.set(
        "cookie",
        `token=${token}; Path=/; HttpOnly`
      );
    }
    return new NextRequest(url, { headers });
  };

  it("redirects to / when accessing /dashboard without token", () => {
    const req = createRequest("/dashboard");
    const res = proxy(req);
    expect(res.status).toBe(307);
    expect(res.headers.get("Location")).toBe("http://localhost:3000/");
  });

  it("allows access to /dashboard with token", () => {
    const req = createRequest("/dashboard", "valid-jwt");
    const res = proxy(req);
    expect(res.status).toBe(200);
  });

  it("allows access to non-dashboard routes without token", () => {
    const req = createRequest("/channel/user-1");
    const res = proxy(req);
    expect(res.status).toBe(200);
  });

  it("allows access to root route without token", () => {
    const req = createRequest("/");
    const res = proxy(req);
    expect(res.status).toBe(200);
  });
});
