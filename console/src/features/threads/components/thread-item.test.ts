import { expect, test } from "vitest";
import {
  resolveThreadIndicatorPresentation,
  resolveThreadIndicatorState,
} from "./thread-item";

test("uses active indicator for active sessions", () => {
  expect(resolveThreadIndicatorState("active", false)).toBe("active");
  expect(resolveThreadIndicatorState("active", true)).toBe("active");
});

test("uses unread indicator for non-active sessions with unread updates", () => {
  expect(resolveThreadIndicatorState("idle", true)).toBe("unread");
  expect(resolveThreadIndicatorState("systemError", true)).toBe("unread");
});

test("uses idle indicator for non-active sessions without unread updates", () => {
  expect(resolveThreadIndicatorState("idle", false)).toBe("idle");
  expect(resolveThreadIndicatorState("unknown", false)).toBe("idle");
});

test("matches Figma indicator presentation", () => {
  expect(resolveThreadIndicatorPresentation("active")).toEqual({
    kind: "spinner",
    className: "size-3 text-emerald-400 animate-spin",
  });
  expect(resolveThreadIndicatorPresentation("unread")).toEqual({
    kind: "dot",
    className: "size-1.5 rounded-full bg-emerald-400",
  });
  expect(resolveThreadIndicatorPresentation("idle")).toEqual({ kind: "none" });
});
