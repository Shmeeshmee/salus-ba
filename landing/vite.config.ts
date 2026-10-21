/// <reference types="vitest/config" />
import react from '@vitejs/plugin-react'
import { defineConfig, type Plugin } from 'vite'
import { injectApp } from './src/lib/injectApp.ts'

type ServerEntry = { render: () => string }

// The dev server renders the page on the server too, so the client hydrates the same markup as in the
// prerendered build.
function renderOnServerInDev(): Plugin {
  return {
    name: 'salus-render-on-server',
    apply: 'serve',
    transformIndexHtml: {
      order: 'post',
      async handler(html, { server }) {
        if (!server) return html
        const { render } = (await server.ssrLoadModule('/src/entry-server.tsx')) as ServerEntry
        return injectApp(html, render())
      },
    },
  }
}

export default defineConfig({
  plugins: [react(), renderOnServerInDev()],
  test: {
    environment: 'jsdom',
  },
})
