import { describe, expect, it } from "vitest";
import { buildGatewayApp } from "./app.js";
import { readGatewayConfig } from "./config.js";

describe("gateway app", () => {
  it("serves health and an empty machine list", async () => {
    const app = buildGatewayApp();

    const health = await app.inject({ method: "GET", url: "/health" });
    const machines = await app.inject({ method: "GET", url: "/machines" });

    expect(health.statusCode).toBe(200);
    expect(JSON.parse(health.body)).toEqual({ ok: true });
    expect(JSON.parse(machines.body)).toEqual({ items: [] });
  });

  it("serves empty thread and environment lists", async () => {
    const app = buildGatewayApp();

    const threads = await app.inject({ method: "GET", url: "/threads" });
    const skills = await app.inject({ method: "GET", url: "/skills" });
    const mcps = await app.inject({ method: "GET", url: "/mcps" });
    const plugins = await app.inject({ method: "GET", url: "/plugins" });

    expect(threads.statusCode).toBe(200);
    expect(skills.statusCode).toBe(200);
    expect(mcps.statusCode).toBe(200);
    expect(plugins.statusCode).toBe(200);

    expect(JSON.parse(threads.body)).toEqual({ items: [] });
    expect(JSON.parse(skills.body)).toEqual({ items: [] });
    expect(JSON.parse(mcps.body)).toEqual({ items: [] });
    expect(JSON.parse(plugins.body)).toEqual({ items: [] });
  });
});

describe("readGatewayConfig", () => {
  it("falls back to defaults for blank values and fails on invalid port", () => {
    const initialPort = process.env.PORT;
    const initialHost = process.env.HOST;

    try {
      process.env.PORT = "   ";
      process.env.HOST = "";
      expect(readGatewayConfig()).toEqual({ port: 3000, host: "0.0.0.0" });

      process.env.PORT = "not-a-number";
      expect(() => readGatewayConfig()).toThrowError(/invalid port/i);
    } finally {
      if (initialPort === undefined) {
        delete process.env.PORT;
      } else {
        process.env.PORT = initialPort;
      }

      if (initialHost === undefined) {
        delete process.env.HOST;
      } else {
        process.env.HOST = initialHost;
      }
    }
  });
});
