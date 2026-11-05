import { defineConfig } from '@playwright/test'

// The web journey of `make e2e`, started by test/e2e/consultation_test.go in the playwright service. Chromium gets
// fake camera and microphone devices and allows them without a prompt. The app is the Vite dev server of web/ on
// this container's localhost (a secure context, as on a developer's host), proxying the API on the compose network.
// Nothing is written into the repository.
export default defineConfig({
  testDir: '.',
  outputDir: '/tmp/playwright-results',
  timeout: 150_000,
  expect: { timeout: 20_000 }, // joining a call takes longer than the default 5 s on a slow runner
  retries: 0,
  workers: 1,
  reporter: 'list',
  globalSetup: './livekit-forward.ts',
  use: {
    baseURL: 'http://localhost:5173',
    permissions: ['camera', 'microphone'],
    launchOptions: { args: ['--use-fake-device-for-media-stream', '--use-fake-ui-for-media-stream'] },
  },
  webServer: {
    command: 'npm run dev -- --host 127.0.0.1 --port 5173 --strictPort',
    cwd: '../../../web',
    url: 'http://127.0.0.1:5173',
    reuseExistingServer: false,
    timeout: 60_000,
  },
})
