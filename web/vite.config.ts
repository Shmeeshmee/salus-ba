/// <reference types="vitest/config" />
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// The dev server proxies API calls to the api service (API_URL, set by compose), so the app calls the API
// on its own origin.
const apiUrl = process.env.API_URL ?? 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/healthz': apiUrl,
      '/readyz': apiUrl,
      '/auth': apiUrl,
      '/invites': apiUrl,
      '/practitioner-roles': apiUrl,
      // Only the JSON actions: /appointments/<id> itself is a screen of the app.
      '^/appointments/[^/]+/(cancel|status)$': apiUrl,
      '/specialties': apiUrl,
      '/fhir': apiUrl,
    },
  },
  test: {
    environment: 'jsdom',
  },
})
