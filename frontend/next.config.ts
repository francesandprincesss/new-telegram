import type { NextConfig } from "next";

const config: NextConfig = {
  output: process.env.CHAT_STANDALONE === "1" ? "standalone" : undefined,
  poweredByHeader: false,
  devIndicators: false,
};

export default config;
