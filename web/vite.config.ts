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
      // Only the JSON actions: /appointments/<id> and /appointments/<id>/call are screens of the app.
      '^/appointments/[^/]+/(cancel|status|questionnaires|consultation(/join|/end)?)$': apiUrl,
      '/questionnaire-responses': apiUrl,
      '/specialties': apiUrl,
      // /questionnaires and /questionnaires/<id> are also screens of the app: only their writes go to the API.
      '/questionnaires': { target: apiUrl, bypass: (req) => (req.method === 'GET' ? '/index.html' : undefined) },
      '/fhir': apiUrl,
    },
  },
  test: {
    environment: 'jsdom',
  },
})
