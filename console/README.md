# Console

The `console/` app is a design-driven React client for the Gateway. The current replacement is organized so design updates can land without reworking Gateway behavior each time.

## Architecture

The console is split into three layers:

1. `src/design/`
   Imported design-source pages, shell components, and design-only styles. Treat this as the upstream visual layer. Keep edits here limited to compatibility fixes that are required to mount the design in this app.
2. `src/gateway/`
   Gateway adapters, hooks, capability policy, and view-model mapping. This layer owns HTTP and WebSocket integration and converts Gateway responses into the props expected by the design layer.
3. `src/app/`
   Application composition: router setup, shell mounting, shared providers, and route-level decisions about which page is shown.

Route pages under `src/pages/` are intentionally thin. They bridge `src/gateway/` hooks into `src/design/` views without embedding Gateway logic into imported design code.

## Where Things Live

- Imported design code lives under `src/design/`.
- Gateway-aware adapters live under `src/gateway/`.
- Shell wiring and route composition live under `src/app/`.
- Route wrappers that connect adapters to design views live under `src/pages/`.

## Capability Policy

Capability policy is defined in [`src/gateway/capabilities.ts`](/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/gateway/capabilities.ts). It is the single place that answers whether a design-visible action is actually connected.

Use capability policy for two cases:

- A Gateway feature is supported and should stay interactive.
- A design exposes an action that the Gateway does not implement in this console; keep the control visible, but disabled or explicitly labeled as not connected.

Do not fake local success for unsupported actions. The design can show the control, but adapter logic must preserve the real Gateway contract.

## Design Source Update Flow

When a new design-source export arrives, use this flow:

1. Replace the affected files under `src/design/`.
2. Reapply only the minimum compatibility edits needed for routing, props, or styling imports.
3. Review `src/gateway/` adapters and view-model mappers for any prop or shape changes introduced by the new design source.
4. Update capability checks if the new design surfaces add, rename, or remove actions.
5. Verify the console with `corepack pnpm test`, `corepack pnpm build`, `corepack pnpm e2e`, and the settings harness at `testenv/settings-e2e/run.sh`.

The goal is to keep design refreshes concentrated in `src/design/`, while Gateway behavior stays isolated in adapters and policy.

## Legacy Path Removal

The design-driven console now lands on the thread hub at `/`. Management pages remain available as dedicated design views under `/machines`, `/environment`, and `/settings`. Legacy console-only paths should not be reintroduced; if a new screen is needed, add it through the same three-layer structure instead of reviving older route trees.
