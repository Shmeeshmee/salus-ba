import { defineConfig } from '@playwright/test'

// The screenshots of docs/demo/SCRIPT.md, taken against a stack that `make demo-seed` filled (SCRIPT.md, "Retaking the
// screenshots"). Chromium at 1280 px; the web app is the Vite dev server of web/ on this container's localhost,
// proxying the API on the compose network (as test/e2e/web does). It writes only docs/demo/screenshots/.
export default defineConfig({
  testDir: '.',
  testMatch: 'screenshots.spec.ts',
  outputDir: '/tmp/demo-screenshots-results',
  timeout: 180_000,
  expect: { timeout: 20_000 },
  retries: 0,
  workers: 1,
  reporter: 'list',
  use: { baseURL: 'http://localhost:5173', viewport: { width: 1280, height: 860 }, locale: 'bs-BA', timezoneId: 'Europe/Sarajevo' },
  webServer: {
    command: 'npm run dev -- --host 127.0.0.1 --port 5173 --strictPort',
    cwd: '../../web',
    url: 'http://127.0.0.1:5173',
    reuseExistingServer: false,
    timeout: 60_000,
  },
})
