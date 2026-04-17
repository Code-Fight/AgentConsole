# Design Bridge

`src/design-bridge/` is the protected local adaptation layer between the upstream UI mirror in `src/design-source/` and the runtime logic owned by `src/design-host/` and `src/gateway/`.

Use this directory for:

- stable view adapters around upstream components
- prop normalization between upstream UI shape and local Gateway view-models
- local glue that must survive upstream mirror replacement

Do not use this directory for:

- raw Gateway HTTP / WebSocket clients
- route entrypoints
- full-page business logic that belongs in `src/gateway/`
- copying upstream components that should remain mirrored under `src/design-source/`

If an upstream sync changes component structure, fix the seam here before considering edits inside `src/design-source/`.
