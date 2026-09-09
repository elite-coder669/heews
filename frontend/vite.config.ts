import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Inside Docker, "localhost" refers to the frontend container itself, not the
// backend container — so the proxy target must be overridable. docker-compose
// sets VITE_BACKEND_URL=http://backend:8080; local `npm run dev` on the host
// falls back to localhost:8080.
const backendTarget = process.env.VITE_BACKEND_URL || 'http://localhost:8080';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    host: '0.0.0.0',
    proxy: {
      '/api': backendTarget,
      '/data': backendTarget,
    },
  },
  preview: {
    port: 5173,
  },
});