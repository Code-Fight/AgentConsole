# Protocol schema package design

## 1. Motivation

- Task 2 of the Code Agent Gateway V1 effort needs a shared protocol package so future readers can import the same command, event and resource schema definitions from a single place.
- The repo already uses NodeNext resolution, so the package must build to `dist/*.js`, export with `.js` suffixes, and stay private to the workspace.
- The shared types cover the Gateway <-> Machine Client channel described in the Task instructions: commands such as `startThread`, `startTurn`, `toggleSkill`; resource snapshots for `skill/mcp/plugin`; and a handful of events used by snapshot listeners.

## 2. Requirements

1. Provide a new `@cag/protocol` workspace package with `zod`-powered validators exported from `src/index.ts`.
2. Implement `commandEnvelopeSchema`, `resourceSnapshotSchema`, and `eventEnvelopeSchema` in their own module files plus re-export them from `src/index.ts` (using `.js` suffixes to satisfy NodeNext).
3. Wire up package metadata (`package.json`, `tsconfig.json`) so `pnpm corepack` builds/tests succeed and `src/**/*.test.ts` stay out of production builds.
4. Add a Vitest test under `packages/protocol/src/schemas.test.ts` that first fails because the module is missing, then passes after the schema implementations exist.
5. Ensure everything lives under `packages/protocol`, with tests able to import by relative paths, and keep the package private with a `main` pointing at `./dist/index.js`.

## 3. Constraints

- `tsconfig.base.json` already exposes `@cag/protocol` pointing at `packages/protocol/src/index.ts`; no change is needed, but we must respect the existing alias.
- NodeNext + ES modules require `.js` suffixes for relative exports/re-exports in `src/*.ts` files, even though `tsc` strips them for runtime.
- Tests should be invoked via `COREPACK_HOME=/tmp/corepack corepack pnpm vitest packages/protocol/src/schemas.test.ts` to match the project-wide convention of using Corepack in CI.
- The new package should follow the minimal structure shown in Task 2, but leave room for future commands/events if the clarified question about extra types ever arrives.

## 4. Approach options

### Option A — Mirror the provided schema list verbatim (recommended)

- Implement the exact discriminated unions and enums described in the task: `command.startThread`, `command.startTurn`, `command.toggleSkill` for commands; a resource snapshot schema that captures the shared fields and enumerated `scope`, `status`, `source`; and three events (`commandAccepted`, `turn.delta`, `resource.changed`).
- Pros: smallest surface area, directly matches the acceptance test, no speculative types.
- Cons: adding new commands/events requires editing the schema file, but that is acceptable for the current scope.

### Option B — Introduce a registry-style `content` map for commands/events

- Build a map of payload builders keyed by `type` and derive `commandEnvelopeSchema` from `z.discriminatedUnion` built from the registry (similar for events).
- Pros: easier to add new message types without touching the union aggregate.
- Cons: more ceremony for the one-off commands, and the acceptance test only needs the minimal set.

### Option C — Define a base schema and allow `z.intersection` for future extensions

- Author a lean base command envelope schema and `extend` it per command via `z.intersection`, leaving `unknown` fields for future third-party data.
- Pros: extremely flexible and works well for optional metadata.
- Cons: the acceptance test benefits from the precise checks in Option A, and our goals favour clarity over extensibility right now.

## 5. Selected design

We will implement Option A as it directly satisfies the acceptance tests and keeps the package focused on the known protocol surface. The implementation details are:

1. **package.json**
   - `name`: `@cag/protocol`, `version`: `0.0.0`, `private`: `true`.
   - `type`: `module`, `main`: `./dist/index.js`.
   - Dependencies: `zod` at `^4.0.0`.
   - Scripts: `build` runs `tsc -p tsconfig.json`, `test` runs `vitest run src/schemas.test.ts`.

2. **tsconfig.json**
   - Extends `../../tsconfig.base.json`, sets `outDir` to `dist`, `rootDir` to `src`, `include`: "src".
   - Excludes test files via `exclude`: "src/**/*.test.ts" so they never ship in `dist`.

3. **src/commands.ts**
   - `commandEnvelopeSchema` is a `z.discriminatedUnion("type", [...])` listing three command objects with precise shapes for each `payload`.

4. **src/resources.ts**
   - `resourceKindSchema` enumerates `skill`, `mcp`, `plugin`.
   - `resourceSnapshotSchema` validates the shared snapshot fields plus enumerated `scope`, `status`, `source`, a boolean `restartRequired`, and ISO `lastObservedAt`.

5. **src/events.ts**
   - `eventEnvelopeSchema` is a `z.discriminatedUnion("type", [...])` containing the three event objects defined in the task.

6. **src/index.ts**
   - Re-export `commandEnvelopeSchema`, `resourceSnapshotSchema`, and `eventEnvelopeSchema` via `export * from "./commands.js";` etc.

7. **src/schemas.test.ts**
   - Imports `commandEnvelopeSchema` and `resourceSnapshotSchema` from `./index` and asserts the provided fixtures parse successfully and keep their `type`/`kind`.

## 6. Testing & verification

- Step 1 (red): run `COREPACK_HOME=/tmp/corepack corepack pnpm vitest packages/protocol/src/schemas.test.ts` before creating the package to confirm Vitest complains about the missing `./index` module.
- Step 2 (green): after implementing the modules and schema files, rerun the same command to confirm the test passes.
- Keep the `test` script in `package.json` in sync with this command so CI and developers can run the same check without typing the full path.

## 7. Next steps

1. Create the workspace files under `packages/protocol` as described above.
2. Commit the spec to `docs/superpowers/specs/2026-04-07-protocol-schemas-design.md` and share it for review.
3. After the spec is reviewed and accepted, invoke the `writing-plans` skill to build an implementation plan aligned with this design.
4. Once the plan is approved, implement the schema files, run the tests (failing then passing), and make the final commit on this branch with the message `feat: add shared protocol schemas`.
