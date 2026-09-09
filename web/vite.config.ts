import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import path from 'path';

export default defineConfig({
  plugins: [svelte()],
  resolve: {
    alias: {
      '$lib': path.resolve('./src/lib'),
      '$components': path.resolve('./src/components'),
      '$stores': path.resolve('./src/stores'),
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    sourcemap: false,
    rollupOptions: {
      input: './index.html',
    }
  },
  server: {
    port: 5173,
    // In dev mode, proxy API and WebSocket calls to the Go backend.
    // Backend always serves HTTPS (self-signed in dev) on :8989 — see cmd/webux/main.go.
    proxy: {
      '/api': {
        target: 'https://localhost:8989',
        changeOrigin: true,
        secure: false, // dev 自签证书, 跳过 TLS 校验
      },
      '/ws': {
        target: 'wss://localhost:8989',
        ws: true,
        secure: false,
      }
    }
  }
});
