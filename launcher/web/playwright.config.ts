import { defineConfig, devices } from '@playwright/test';

/**
 * The browser tests run against the fake launcher: the real Connect handlers,
 * the real WebSocket and the real form model, with no game server, no bridge
 * and no multiworld behind them. See launcher/cmd/fakelauncher.
 *
 * The fake is started here rather than by hand so `npm run test:e2e` is one
 * command on a laptop and one step in CI. It builds the app first, because what
 * is under test is what the launcher would embed.
 */
const port = 8471;

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env['CI'],
  retries: process.env['CI'] ? 1 : 0,
  // The HTML report is written in CI too. It used to be list-only there, so a
  // failing run uploaded an empty artifact and the only way to see what broke
  // was to wait for the whole run to finish and read the raw log.
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: `http://127.0.0.1:${port}`,
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  // The real-launcher tests talk to a launcher somebody started; the rest bring
  // the fake up themselves. Both live in e2e/, and the real ones skip
  // themselves when TF2AP_REAL is unset.
  webServer: [
    {
      // The frontend is fenced out of the Go module by launcher/web/go.mod, so
      // the fake is run from the repository root rather than from here.
      command: `npm run build && cd ../.. && go run ./launcher/cmd/fakelauncher -addr 127.0.0.1:${port}`,
      url: `http://127.0.0.1:${port}/`,
      reuseExistingServer: !process.env['CI'],
      timeout: 240_000,
    },
    {
      // The public tracker is a separate Angular entry, not a launcher route.
      command:
        'mkdir -p src/tracker/data && cp ../../apworld/tf2_mvm/data/missions.json ../../apworld/tf2_mvm/data/weapon_classes.json ../../apworld/tf2_mvm/data/class_loadouts.json src/tracker/data/ && npm run build:tracker && node e2e/tracker-server.cjs',
      url: 'http://127.0.0.1:8472/',
      reuseExistingServer: !process.env['CI'],
      timeout: 240_000,
    },
  ],
});
