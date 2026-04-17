# Console Upstream Git Sync Docs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Record the new upstream Git source strategy for `console/`, add an AI-readable sync contract, and update the READMEs so future sync work preserves local Gateway adaptations.

**Architecture:** This docs-only iteration adds a detailed strategy spec under `docs/`, a machine-readable sync manifest under `console/`, and README guidance that distinguishes the upstream mirror layer from local bridge/host/gateway protection zones. Runtime wiring is intentionally left unchanged in this iteration.

**Tech Stack:** Markdown, JSON

---

### Task 1: Write the detailed strategy and seam inventory

**Files:**
- Create: `docs/superpowers/specs/2026-04-18-console-upstream-git-sync-design.md`
- Create: `docs/superpowers/plans/2026-04-18-console-upstream-git-sync.md`

- [ ] Draft the strategy spec covering source-of-truth changes, layer boundaries, seam ownership, Gateway reuse inventory, protected paths, and the AI sync workflow.
- [ ] Save this implementation plan alongside the spec so later execution can reference exact doc targets.

### Task 2: Add the machine-readable sync contract and bridge scaffold

**Files:**
- Create: `console/upstream-sync.manifest.json`
- Create: `console/src/design-bridge/README.md`

- [ ] Add a JSON manifest describing the upstream repository, path mappings, protected local directories, critical seam components, and verification commands.
- [ ] Add a lightweight `design-bridge` directory README that marks the folder as the future home for stable local UI adaptation work.

### Task 3: Rewrite the Console READMEs to match the new strategy

**Files:**
- Modify: `console/README.md`
- Modify: `console/README.zh-CN.md`

- [ ] Replace the old Figma/design-export language with the new upstream Git source rules.
- [ ] Document the four-layer strategy: `design-source`, `design-bridge`, `design-host`, and `gateway`.
- [ ] Add an explicit AI sync contract, protected-path rules, and deprecate the old `design-source-sync` / `scale` guidance.

### Task 4: Verify the docs iteration

**Files:**
- Verify only

- [ ] Run `node -e "JSON.parse(require('fs').readFileSync('console/upstream-sync.manifest.json', 'utf8'))"` from repo root and verify the manifest parses successfully.
- [ ] Run `git diff --check` from repo root and verify there are no patch formatting issues.
- [ ] Review the updated README and spec together to confirm they describe the same source repository, allowed write areas, and sync flow.
