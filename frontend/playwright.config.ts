import { defineConfig, devices } from "@playwright/test";

// E2E runs against the built app (dist) served by `vite preview`, with the Wails
// bridge (window.go / window.runtime) stubbed in the browser — so it exercises
// the real Svelte UI without a Go backend. Its job is to catch render-time
// breakage (a ReferenceError or a bad template hangs the whole app, and Go-only
// CI never sees it).
export default defineConfig({
  testDir: "./e2e",
  timeout: 60_000,
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  reporter: "list",
  use: {
    baseURL: "http://localhost:4173",
    headless: true,
  },
  webServer: {
    command: "npm run build && npm run preview -- --port 4173 --strictPort",
    url: "http://localhost:4173",
    reuseExistingServer: !process.env.CI,
    timeout: 180_000,
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
