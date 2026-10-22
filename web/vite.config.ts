import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { appShell } from "./pwa-plugin.ts";

export default defineConfig({
  plugins: [react(), appShell()],
  build: {
    outDir: "../internal/dashboard/ui/dist",
    emptyOutDir: true,
    manifest: true,
  },
  server: {
    proxy: { "/api": "http://127.0.0.1:8090" },
  },
});
