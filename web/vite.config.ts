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
  // Kutubxonalar alohida chunk'larda: ilova kodi o'zgarganda brauzer keshidagi vendor
  // fayllar qayta yuklanmaydi. Guruhlar priority bo'yicha tekshiriladi, qolgani — vendor.
  build: {
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            {
              name: "react",
              test: /node_modules[\\/](react|react-dom|scheduler|react-router)[\\/]/,
              priority: 30,
            },
            { name: "radix", test: /node_modules[\\/](radix-ui|@radix-ui|@floating-ui)[\\/]/, priority: 20 },
            { name: "tanstack", test: /node_modules[\\/]@tanstack[\\/]/, priority: 20 },
            { name: "vendor", test: /node_modules[\\/]/, priority: 10 },
          ],
        },
      },
    },
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
