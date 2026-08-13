import type { NextConfig } from "next";

const BACKEND_URL = process.env.BACKEND_URL || "http://localhost:8081";
// SRS HTTP server (HLS playlists/segments). Separate from the API backend.
const HLS_URL = process.env.HLS_URL || "http://localhost:8080";

const nextConfig: NextConfig = {
  /* config options here */
  reactCompiler: true,
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${BACKEND_URL}/api/:path*`,
      },
      {
        source: "/hls/:path*",
        destination: `${HLS_URL}/:path*`,
      },
      {
        source: "/live/:path*",
        destination: `${HLS_URL}/live/:path*`,
      },
    ];
  },
};

export default nextConfig;
