import path from "node:path";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// API manzili: dev da Vite /api va /.well-known ni Go serverga yo'naltiradi.
const api = process.env.API_URL ?? "http://localhost:5000";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { "@": path.resolve(import.meta.dirname, "src") },
  },
  server: {
    // make dev emaildagi linkni shu portga yo'naltiradi, port band bo'lsa boshqasiga o'tmasin
    port: 5173,
    strictPort: true,
    proxy: {
      "/api": api,
      "/.well-known": api,
    },
  },
});
