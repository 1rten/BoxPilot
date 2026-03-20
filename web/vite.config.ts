import path from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, __dirname, "");
  const apiOrigin = env.DEV_API_ORIGIN || "http://localhost:8080";

  return {
    plugins: [react()],
    server: {
      proxy: {
        "/api": apiOrigin,
        "/healthz": apiOrigin,
      },
    },
  };
});
