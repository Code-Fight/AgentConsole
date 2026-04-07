export function readGatewayConfig() {
  return {
    port: readPort(process.env.PORT),
    host: readHost(process.env.HOST),
  };
}

function readPort(portValue: string | undefined): number {
  const raw = portValue?.trim();
  if (!raw) {
    return 3000;
  }

  const port = Number(raw);
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new Error(`Invalid PORT value: ${portValue}`);
  }

  return port;
}

function readHost(hostValue: string | undefined): string {
  const host = hostValue?.trim();
  if (!host) {
    return "0.0.0.0";
  }

  return host;
}
