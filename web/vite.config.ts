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
      '/binaries': apiUrl, // the files of questionnaire answers: status and content
      '/specialties': apiUrl,
      '/terminology': apiUrl, // the cheatsheet's search (no screen of the app is under /terminology)
      // /questionnaires and /questionnaires/<id> are also screens of the app: only their writes go to the API.
      '/questionnaires': { target: apiUrl, bypass: (req) => (req.method === 'GET' ? '/index.html' : undefined) },
      '/fhir': apiUrl,
      // The consultation's live events (a WebSocket) and its transcript; no screen of the app is under /encounters.
      '/encounters': { target: apiUrl, ws: true },
    },
  },
  test: {
    environment: 'jsdom',
    // app.css is read as text by the tests that check a spacing against it (Builder.test.tsx).
    css: { include: [/app\.css/] },
  },
})
