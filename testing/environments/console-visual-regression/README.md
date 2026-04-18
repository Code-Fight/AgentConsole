# Console Visual Regression Environment

This environment runs deterministic Playwright screenshot tests for the console shell using
`console/playwright.visual.config.ts` and centralized specs in `testing/playwright`.

## What It Covers

- Thread hub home (`/`)
- Thread workspace (`/threads/thread-1`)
- Settings (`/settings`)
- Environment (`/environment`)
- Machines (`/machines`)

The suite seeds gateway cookies and stubs gateway API responses so visual snapshots are stable
across runs.

The visual environment is deterministic by design:
- It always starts its own server on `127.0.0.1:4174`.
- It must not reuse an arbitrary existing server process.
- Baselines are stored with stable names (`thread-hub.png`, `thread-workspace.png`, etc.)
  under `testing/playwright/console-visual-regression.spec.ts-snapshots/`.
- `IBM Plex Sans` must be available on the machine running the suite.
- The suite intentionally fails if `IBM Plex Sans` is unavailable, rather than silently
  rendering with fallback fonts.

## Run

From repo root:

```bash
./testing/environments/console-visual-regression/run.sh
```

Or directly:

```bash
cd console
corepack pnpm e2e:visual
```

## Regenerate Baselines

```bash
cd console
corepack pnpm e2e:visual --update-snapshots
```

Commit updated snapshot files when intentional UI changes are accepted.
