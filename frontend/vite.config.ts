import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

// The dev server proxies the API so the browser talks to one origin.
// Point API_TARGET at a backend started with `make up` in ../backend.
const apiTarget = process.env.API_TARGET ?? "http://localhost:8090";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      "/api": { target: apiTarget, ws: true },
    },
  },
  test: {
    environment: "node",
  },
});
