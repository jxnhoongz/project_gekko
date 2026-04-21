import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import tailwindcss from '@tailwindcss/vite';
import path from 'path';

const BACKEND = process.env.VITE_BACKEND_URL ?? 'http://localhost:8420';

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: { '@': path.resolve(__dirname, './src') },
  },
  server: {
    port: 5173,
    host: '0.0.0.0',
    // Allow any *.ts.net hostname (Tailscale MagicDNS) plus local network
    allowedHosts: ['.ts.net', 'gekko-dev', 'localhost'],
    proxy: {
      '/api':     { target: BACKEND, changeOrigin: true },
      '/uploads': { target: BACKEND, changeOrigin: true },
      '/health':  { target: BACKEND, changeOrigin: true },
    },
  },
});
