# Console

The `console/` app now follows a `design source first` strategy.

The immediate goal is not to reinterpret the UI or selectively rebuild it. The goal is to keep the generated design code as close to upstream as possible, so future design iterations can be dropped into this repo quickly and predictably.

## Architecture

The console is split into three layers:

1. `src/design-source/`
   The imported upstream design code. Treat this as near-read-only. It should stay visually faithful to the latest design export and should not absorb Gateway-specific business logic.
2. `src/design-host/`
   The thinnest possible compatibility layer that mounts the design source into this repo. This is the right place for entrypoint-level fixes, wrappers, shims, style imports, or other host concerns required to run the imported design.
3. `src/gateway/`
   The integration layer for later phases. This is where HTTP, WebSocket, capability policy, and view-model mapping should live once Gateway connectivity is reintroduced on top of the imported design source.

`src/app/` remains available for application composition concerns such as routers and providers, but the current 1:1 design phase is intentionally mounted through `src/design-host/` instead of rebuilding the UI through `src/pages/`.

## Where Things Live

- Imported upstream design code: `src/design-source/`
- Host-level mount and compatibility glue: `src/design-host/`
- Future Gateway adapters and policies: `src/gateway/`
- Legacy route-oriented wrappers from earlier iterations still exist in `src/pages/`, but they are not the source of truth for the current rendered UI

## Ground Rules

- Do not hand-edit `src/design-source/` for business logic.
- Only make the minimum compatibility changes required for the imported design to build and render.
- If Gateway integration is needed, add it outside the design source and feed it inward through host/adapters.
- Do not re-implement the same page by hand just because the generated code looks unusual. Preserving update velocity is more important than making the imported layer look locally hand-written.

## Design Source Update Flow

When a new design export arrives, use this flow:

1. Replace the affected files under `src/design-source/`.
2. Reapply only the minimum host-level compatibility edits in `src/design-host/` or the build toolchain.
3. Verify that the imported design still builds without introducing Gateway logic into `src/design-source/`.
4. If the new design changes public props or interaction seams, update the outer integration layer in `src/gateway/`, not the imported design code.
5. Run:
   - `corepack pnpm test`
   - `corepack pnpm build`
   - `corepack pnpm e2e`
   - `./testenv/settings-e2e/run.sh`

The desired workflow is:

`new design export -> replace design-source -> minimal host fixes -> re-run verification`

That is the core reason this structure exists.

## Capability Policy

Gateway capability policy still belongs in [`src/gateway/capabilities.ts`](/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/gateway/capabilities.ts), but it should only control integration behavior outside the design source.

The imported design can expose controls that are not yet wired. When that happens, prefer:

- host/adaptor-level disablement
- explicit “not connected” states
- view-model mapping outside `src/design-source/`

Do not fake local success paths inside the imported design layer.

## Current Phase

This repo is currently in the `1:1 design import` phase:

- the runtime entrypoint renders the imported design source
- the priority is visual fidelity and future re-import speed
- full Gateway reattachment happens in later steps, on top of this imported design base
