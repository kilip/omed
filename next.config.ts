import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* config options here */
  transpilePackages: ["@electric-sql/pglite"],
};

export default nextConfig;
