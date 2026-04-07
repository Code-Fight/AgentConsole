import type { FastifyInstance } from "fastify";
import { RegistryStore } from "../../store/registry-store.js";

export function registerMachineRoutes(
  app: FastifyInstance,
  store: RegistryStore,
): void {
  app.get("/machines", async () => ({ items: store.listMachines() }));
}
