# Console

The `console/` app now follows an `upstream-source first` strategy.

The source of truth for the upstream UI is no longer a Figma export workflow. It is the Git repository:

- `git@github.com:Code-Fight/Agentconsolewebsite.git`

`src/design-source/` remains the mirror of upstream UI code, but its origin is now the upstream Git repository instead of a direct design-export sync. The local project keeps Gateway integration, routing, host control, and future seam-stabilization work outside that mirror so upstream UI refreshes can continue without wiping out local behavior.

## Source Of Truth

- Upstream UI source: `git@github.com:Code-Fight/Agentconsolewebsite.git`
- Machine-readable sync contract: `console/upstream-sync.manifest.json`
- Detailed strategy and seam inventory: `docs/superpowers/specs/2026-04-18-console-upstream-git-sync-design.md`

The older Figma-export assumptions, `design-source-sync` workflow, and any `scale`-based strategy are obsolete for this console and must not be used for future sync work.

## Architecture

The console now has four layers:

1. `src/design-source/`
   The upstream mirror layer. This directory is the local mirror of the upstream UI repository and is expected to be replaceable as a package.
2. `src/design-bridge/`
   The protected local seam layer. This is where stable local adapters should live when upstream components need real props, normalized view-models, or controlled callbacks that must survive mirror replacement.
3. `src/design-host/`
   The host control layer. Routing, entrypoint gating, top-level mount logic, and other runtime host concerns belong here.
4. `src/gateway/`
   The Gateway integration layer. HTTP, WebSocket, capability policy, transport helpers, and page view-model assembly belong here.

`src/common/api/` and `src/pages/` remain local implementation areas. They are not part of the upstream mirror.

## Current Status

The current runtime still mounts the active console through `src/design-host/` and reuses local Gateway hooks. This strategy iteration introduces `src/design-bridge/` as the protected long-term seam layer, but the runtime is not fully migrated into it yet. Future adaptation work should move in that direction instead of adding more direct business logic inside `src/design-source/`.

## Allowed Write Areas

Upstream sync work may replace files only inside the mirror targets declared in `console/upstream-sync.manifest.json`.

Local adaptation work belongs in:

- `src/design-bridge/`
- `src/design-host/`
- `src/gateway/`
- `src/common/api/`
- `src/pages/`
- tests

## Protected Paths

Do not overwrite these paths when syncing upstream UI code:

- `console/src/design-bridge/`
- `console/src/design-host/`
- `console/src/gateway/`
- `console/src/common/api/`
- `console/src/pages/`
- `console/tests/`
- `testing/`
- `testenv/`

## AI Sync Contract

Any AI agent updating upstream UI code must follow this process:

1. Read this README and `console/upstream-sync.manifest.json` before making changes.
2. Clone or fetch `git@github.com:Code-Fight/Agentconsolewebsite.git` into a temporary working directory.
3. Check out the intended branch or commit in that temporary directory.
4. Sync only the path mappings declared in `console/upstream-sync.manifest.json`.
5. Do not widen the overwrite scope beyond those mappings.
6. After syncing, check the critical seam components listed in the manifest:
   - `App`
   - `SetupWizard`
   - `Settings`
   - `MachinePanel`
   - `ThreadItem`
   - `SessionChat`
   - `Machines`
   - `Environment`
7. If the upstream structure changed and the local runtime no longer fits:
   - fix `src/design-bridge/` first
   - then fix `src/design-host/`
   - only update `src/gateway/` if the required local view-model contract truly changed
8. Do not patch Gateway logic, mock removal, connection flow, or routing behavior directly into `src/design-source/` unless the change is purely mirror maintenance and cannot be expressed elsewhere.

## Do Not Edit

When working on sync or integration, do not:

- use the old Figma-export sync workflow
- rely on the deprecated `design-source-sync` skill for this console
- use any `scale` strategy
- write Gateway HTTP / WebSocket logic into `src/design-source/`
- keep upstream mock-only runtime state on the active runtime path when a local Gateway-backed seam already exists
- overwrite protected local directories during mirror replacement

## Upstream Sync Flow

The intended sync flow is:

`fetch upstream repo -> replace only mirror targets -> inspect seam components -> repair bridge/host seams if needed -> rerun verification`

That is the core reason the mirror layer and local protected layers are kept separate.

## Verification

After upstream sync work, run at least:

```bash
cd console
corepack pnpm test
corepack pnpm build
```

If the sync affects interactive flows or key management pages, also run:

```bash
cd console
corepack pnpm e2e
cd ..
./testing/environments/settings-e2e/run.sh
```

## Manifest

`console/upstream-sync.manifest.json` is the machine-readable contract for:

- upstream repository location
- path mappings from upstream source to local mirror targets
- protected local paths
- critical seam components
- required verification commands

AI agents should treat the manifest as the execution companion to this README.
