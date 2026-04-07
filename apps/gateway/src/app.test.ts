import { describe, expect, it } from "vitest";
import { buildGatewayApp } from "./app.js";

describe("gateway app", () => {
  it("serves health and an empty machine list", async () => {
    const app = buildGatewayApp();

    const health = await app.inject({ method: "GET", url: "/health" });
    const machines = await app.inject({ method: "GET", url: "/machines" });

    expect(health.statusCode).toBe(200);
    expect(JSON.parse(health.body)).toEqual({ ok: true });
    expect(JSON.parse(machines.body)).toEqual({ items: [] });
  });
});
