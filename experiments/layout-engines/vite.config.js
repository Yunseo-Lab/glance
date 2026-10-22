import { defineConfig } from "vite";
export default defineConfig({
  build: {
    manifest: true,
    rollupOptions: { input: { dnd: "dnd.html", gridstack: "gridstack.html" } },
  },
});
