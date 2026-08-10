import type { NextConfig } from "next";

const BACKEND_URL = process.env.BACKEND_URL || "http://localhost:8081";

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
        destination: `http://localhost:8080/:path*`,
      },
      {
        source: "/live/:path*",
        destination: `http://localhost:8080/live/:path*`,
      },
    ];
  },
};

export default nextConfig;
