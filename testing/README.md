# Testing

This directory is the project-level home for centralized verification assets.

- `playwright/` contains browser-driven end-to-end scenarios
- `environments/` contains multi-service Docker or shell harnesses
- future integration suites should be added here instead of `console/tests/` or `testenv/`

Canonical entry points:

```bash
# from console/
corepack pnpm exec playwright test --config playwright.config.ts --list
corepack pnpm exec playwright test --config playwright.settings.config.ts --list

# from repo root
./testing/environments/settings-e2e/run.sh
./testing/environments/dev-integration/run.sh restart
```
