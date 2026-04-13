---
name: design-source-sync
description: Use when refreshing generated design source code into console/src/design-source, preserving 1:1 UI fidelity while keeping host compatibility fixes and Gateway integration outside the imported design layer.
---

# Design Source Sync

## Overview

Use this skill when the latest generated design code needs to be pulled into the project again.

The core rule is:

- keep `console/src/design-source/` as the upstream mirror
- keep compatibility fixes in `console/src/design-host/`
- keep Gateway-specific logic out of the imported design source

## When To Use

Use this skill when:

- the design source has changed and `console/src/design-source/` must be updated
- you want to test whether the current import flow still supports rapid re-sync
- visual fidelity matters more than refactoring the generated code
- a future integration step needs to build on top of a fresh imported design baseline

Do not use this skill when:

- you are wiring Gateway behavior into the app
- you are redesigning the imported code by hand
- you only need a small product tweak outside the imported design layer

## Project Structure

- `console/src/design-source/`
  Upstream generated design code. Near-read-only.
- `console/src/design-host/`
  Thin host layer for mount points, runtime wrappers, and compatibility fixes.
- `console/src/gateway/`
  Future integration layer for HTTP, WebSocket, view-model mapping, and capability policy.

## Sync Workflow

1. Pull the latest generated design source from the upstream design tool or export.
2. Replace only the affected files under `console/src/design-source/`.
3. Keep the imported code visually faithful. Do not push business logic into it.
4. Reapply only the minimum compatibility fixes needed in:
   - `console/src/design-host/`
   - `console/src/main.tsx`
   - `console/vite.config.ts`
   - `console/package.json`
   - design-source style entry files
5. If the new design changes copy, component names, or layout seams, update the host and test layers rather than rewriting the imported design.
6. Rebuild and run verification.

## Verification

Always run:

```bash
cd console
corepack pnpm test
corepack pnpm build
corepack pnpm e2e
cd ..
./testenv/settings-e2e/run.sh
```

If visual layout is suspicious, restart the local integration stack:

```bash
./testenv/dev-integration/run.sh restart
```

Then inspect:

- `http://localhost:14173`
- `http://localhost:18080`

## Guardrails

- Do not treat generated design code as the place for product logic.
- Do not re-implement design pages by hand if the generated code can be imported directly.
- Do not silently absorb unsupported backend actions into fake local success paths.
- If a design update breaks the build, prefer fixing the host/build layer before modifying the imported design.

## Typical Update Targets

Common files that may need to change during a sync:

- `console/src/design-source/App.tsx`
- `console/src/design-source/components/*`
- `console/src/design-source/data/mockData.ts`
- `console/src/design-source/styles/*`

Common files that may need host-level adjustments after a sync:

- `console/src/design-host/app-root.tsx`
- `console/src/main.tsx`
- `console/package.json`
- `console/pnpm-lock.yaml`
- `console/vite.config.ts`
- `console/tests/console-smoke.spec.ts`
- `console/tests/settings-e2e.spec.ts`

## Success Criteria

A sync is successful when:

- the latest generated design files are present in `console/src/design-source/`
- the app still renders through the design-source entrypoint
- the UI remains visually faithful to the upstream design
- the build and verification commands pass
- future Gateway integration can still be added outside the imported design layer
