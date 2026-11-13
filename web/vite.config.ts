/// <reference types="vitest/config" />
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// The dev server proxies API calls to the api service (API_URL, set by compose), so the app calls the API
// on its own origin. Production routes the same paths in deployments/production/Caddyfile (app.); make lint
// fails when the two differ (scripts/check-caddy-routes.sh).
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
      '^/appointments/[^/]+/(cancel|status|changes|questionnaires|consultation(/join|/end)?)$': apiUrl,
      '/questionnaire-responses': apiUrl,
      '/binaries': apiUrl, // the files of questionnaire answers: status and content
      '/specialties': apiUrl,
      '/terminology': apiUrl, // the cheatsheet's search and the terms' explanations (no screen of the app is under /terminology)
      // /questionnaires and /questionnaires/<id> are also screens of the app: only their writes go to the API.
      '/questionnaires': { target: apiUrl, bypass: (req) => (req.method === 'GET' ? '/index.html' : undefined) },
      '/fhir': apiUrl,
      // The consultation's live events (a WebSocket) and its transcript; no screen of the app is under /encounters.
      '/encounters': { target: apiUrl, ws: true },
      // The documents drafted in the call (SAL-8): the clinic's templates and a draft's sign; no screen is under either.
      '/document-templates': apiUrl,
      '/drafts': apiUrl,
      '/compositions': apiUrl, // a document's explained terms (SAL-13); no screen is under it
      // The chat's JSON API and its socket (SAL-27, SAL-31); the chat's screens are under /chats.
      '/conversations': { target: apiUrl, ws: true },
      '/places': apiUrl, // the places of a clinic's location (SAL-19); no screen is under it
      '/analytics/clinic': apiUrl, // the clinic's figures (SAL-20); the screen is /analytics itself
      '/analytics/platform': apiUrl, // the All of Salus figures (SAL-21); no screen is under it
      '/patients': apiUrl, // a patient's history and pre-visit summary (SAL-40); no screen is under it
      '/clinic/': apiUrl, // the admin's clinic settings, SMS usage and no-show flags (SAL-32, SAL-33, SAL-36); /clinic itself is a screen
      '^/waitlist/.+$': apiUrl, // removing an entry of the waitlist (SAL-35); /waitlist itself is a screen of the app
    },
  },
  test: {
    environment: 'jsdom',
    // app.css and analytics.css are read as text by the tests that check a spacing or the motion against them
    // (Builder.test.tsx, analytics/motion.test.ts).
    css: { include: [/app\.css/, /analytics\.css/] },
  },
})
