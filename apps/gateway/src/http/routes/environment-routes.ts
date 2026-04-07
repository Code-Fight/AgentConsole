import type { FastifyInstance } from "fastify";

export function registerEnvironmentRoutes(app: FastifyInstance): void {
  app.get("/skills", async () => ({ items: [] }));
  app.get("/mcps", async () => ({ items: [] }));
  app.get("/plugins", async () => ({ items: [] }));
}
