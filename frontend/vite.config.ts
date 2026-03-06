import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  base: "/ui/",
  build: {
    outDir: "../ui",
    emptyOutDir: false
  },
  test: {
    environment: "jsdom",
    setupFiles: ["./vitest.setup.ts"],
  }
});
