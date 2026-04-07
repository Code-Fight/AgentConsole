import Fastify from "fastify";
import { registerEnvironmentRoutes } from "./http/routes/environment-routes.js";
import { registerMachineRoutes } from "./http/routes/machines-routes.js";
import { registerThreadRoutes } from "./http/routes/threads-routes.js";
import { RegistryStore } from "./store/registry-store.js";
import { ThreadStore } from "./store/thread-store.js";

export function buildGatewayApp() {
  const app = Fastify();
  const registryStore = new RegistryStore();
  const threadStore = new ThreadStore();

  app.get("/health", async () => ({ ok: true }));
  registerMachineRoutes(app, registryStore);
  registerThreadRoutes(app, threadStore);
  registerEnvironmentRoutes(app);

  return app;
}
