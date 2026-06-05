import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Em produção o webapp é servido pelo binário do servidor (origem única),
// então a API é relativa ("/api/v1"). Em dev, fazemos proxy para o servidor.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": { target: "http://localhost:4533", changeOrigin: true },
    },
  },
  build: { outDir: "dist" },
});
