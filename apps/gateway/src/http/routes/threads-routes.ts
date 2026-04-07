import type { FastifyInstance } from "fastify";
import { ThreadStore } from "../../store/thread-store.js";

export function registerThreadRoutes(
  app: FastifyInstance,
  store: ThreadStore,
): void {
  app.get("/threads", async () => ({ items: store.listThreads() }));
}
