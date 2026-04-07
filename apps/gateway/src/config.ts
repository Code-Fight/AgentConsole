export function readGatewayConfig() {
  return {
    port: Number(process.env.PORT ?? 3000),
    host: process.env.HOST ?? "0.0.0.0",
  };
}
