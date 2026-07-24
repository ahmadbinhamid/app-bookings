import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const rootDir = dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  resolve: {
    alias: { '@': resolve(rootDir, 'src') },
    // @flowposltd/ui is linked in via a `file:` symlink into the sibling
    // ui-kit workspace. Without this, Vite resolves that package's own
    // `react` import against ui-kit/node_modules/react instead of this
    // project's copy — two distinct module instances (even at the same
    // version) break React's hook dispatcher ("Invalid hook call").
    dedupe: ['react', 'react-dom'],
  },
  plugins: [
    react(),
    tailwindcss(),
  ],
  build: {
    target: 'esnext',
  },
  server: {
    // Vite's dev-server Host header check rejects any origin it doesn't
    // recognize by default. Embedding this app inside tenant-dashboard
    // during local dev goes through an ngrok tunnel (a random subdomain
    // that changes every time the tunnel restarts), so allowlist the
    // domain suffixes rather than one ephemeral hostname.
    allowedHosts: ['.ngrok-free.dev', '.ngrok-free.app', '.ngrok.io'],
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      // FlowPOS calls these against this app's registered public URL (the
      // ngrok tunnel in local dev) — same routes nginx.conf proxies in the
      // Docker build, mirrored here so the marketplace install/uninstall/
      // webhook lifecycle reaches the Go backend instead of the SPA.
      '^/(install|uninstall|webhooks)$': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  preview: {
    port: 5175,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '^/(install|uninstall|webhooks)$': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
