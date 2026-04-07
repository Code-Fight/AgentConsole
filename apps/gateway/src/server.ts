import { buildGatewayApp } from "./app.js";
import { readGatewayConfig } from "./config.js";

const app = buildGatewayApp();
const config = readGatewayConfig();

await app.listen({ port: config.port, host: config.host });
