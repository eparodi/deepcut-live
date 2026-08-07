import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

/**
 * Edge-compatible middleware for auth guards.
 *
 * - `/dashboard/*`: requires `token` cookie (JWT from backend).
 *   Redirects unauthenticated users to `/`.
 */
export function middleware(request: NextRequest) {
  const token = request.cookies.get("token")?.value;

  // Protect dashboard routes
  if (request.nextUrl.pathname.startsWith("/dashboard")) {
    if (!token) {
      return NextResponse.redirect(new URL("/", request.url));
    }
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/dashboard/:path*"],
};
