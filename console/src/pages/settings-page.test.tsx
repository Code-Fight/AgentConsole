import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { SettingsPage } from "./settings-page";

afterEach(() => {
  vi.unstubAllGlobals();
});

test("renders global default and machine override settings", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = typeof input === "string" ? input : input.toString();

    if (path === "/settings/agents") {
      return jsonResponse({
        items: [{ agentType: "codex", displayName: "Codex" }]
      });
    }
    if (path === "/machines") {
      return jsonResponse({
        items: [{ id: "machine-01", name: "Machine 01", status: "online", runtimeStatus: "running" }]
      });
    }
    if (path === "/settings/agents/codex/global") {
      return jsonResponse({
        document: { agentType: "codex", format: "toml", content: "model = \"gpt-5.4\"\n" }
      });
    }
    if (path === "/settings/machines/machine-01/agents/codex") {
      return jsonResponse({
        machineId: "machine-01",
        agentType: "codex",
        globalDefault: { agentType: "codex", format: "toml", content: "model = \"gpt-5.4\"\n" },
        machineOverride: null,
        usesGlobalDefault: true
      });
    }

    throw new Error(`Unexpected request: ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);

  render(<SettingsPage />);

  await waitFor(() => {
    expect(screen.getByLabelText("Global Default TOML")).toHaveValue("model = \"gpt-5.4\"\n");
  });
  expect(await screen.findByText("Using Global Default")).toBeInTheDocument();
  expect(screen.getByText("Codex")).toBeInTheDocument();
  expect(screen.getByText("Machine 01")).toBeInTheDocument();
});

test("saving global default sends put request", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString();

    if (path === "/settings/agents") {
      return jsonResponse({ items: [{ agentType: "codex", displayName: "Codex" }] });
    }
    if (path === "/machines") {
      return jsonResponse({ items: [{ id: "machine-01", name: "Machine 01", status: "online", runtimeStatus: "running" }] });
    }
    if (path === "/settings/agents/codex/global" && (!init?.method || init.method === "GET")) {
      return jsonResponse({ document: null });
    }
    if (path === "/settings/machines/machine-01/agents/codex") {
      return jsonResponse({
        machineId: "machine-01",
        agentType: "codex",
        globalDefault: null,
        machineOverride: null,
        usesGlobalDefault: true
      });
    }
    if (path === "/settings/agents/codex/global" && init?.method === "PUT") {
      return jsonResponse({
        document: { agentType: "codex", format: "toml", content: "model = \"gpt-5.4\"\n" }
      });
    }

    throw new Error(`Unexpected request: ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);

  render(<SettingsPage />);

  fireEvent.change(await screen.findByLabelText("Global Default TOML"), {
    target: { value: "model = \"gpt-5.4\"\n" }
  });
  fireEvent.click(screen.getByRole("button", { name: "Save Global Default" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      "/settings/agents/codex/global",
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({ content: "model = \"gpt-5.4\"\n" })
      })
    );
  });
});

test("saving machine override and applying settings use the machine endpoint", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString();

    if (path === "/settings/agents") {
      return jsonResponse({ items: [{ agentType: "codex", displayName: "Codex" }] });
    }
    if (path === "/machines") {
      return jsonResponse({ items: [{ id: "machine-01", name: "Machine 01", status: "online", runtimeStatus: "running" }] });
    }
    if (path === "/settings/agents/codex/global") {
      return jsonResponse({
        document: { agentType: "codex", format: "toml", content: "model = \"gpt-5.4\"\n" }
      });
    }
    if (path === "/settings/machines/machine-01/agents/codex" && (!init?.method || init.method === "GET")) {
      return jsonResponse({
        machineId: "machine-01",
        agentType: "codex",
        globalDefault: { agentType: "codex", format: "toml", content: "model = \"gpt-5.4\"\n" },
        machineOverride: null,
        usesGlobalDefault: true
      });
    }
    if (path === "/settings/machines/machine-01/agents/codex" && init?.method === "PUT") {
      return jsonResponse({
        document: { agentType: "codex", format: "toml", content: "model = \"gpt-5.2\"\n" }
      });
    }
    if (path === "/settings/machines/machine-01/agents/codex/apply" && init?.method === "POST") {
      return jsonResponse({
        machineId: "machine-01",
        agentType: "codex",
        source: "machine",
        filePath: "/tmp/.codex/config.toml"
      });
    }

    throw new Error(`Unexpected request: ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);

  render(<SettingsPage />);

  fireEvent.change(await screen.findByLabelText("Machine Override TOML"), {
    target: { value: "model = \"gpt-5.2\"\n" }
  });
  fireEvent.click(screen.getByRole("button", { name: "Save Machine Override" }));
  fireEvent.click(await screen.findByRole("button", { name: "Apply To Machine" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      "/settings/machines/machine-01/agents/codex",
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({ content: "model = \"gpt-5.2\"\n" })
      })
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/settings/machines/machine-01/agents/codex/apply",
      expect.objectContaining({
        method: "POST"
      })
    );
  });
});

test("invalid toml blocks saving", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = typeof input === "string" ? input : input.toString();
    if (path === "/settings/agents") {
      return jsonResponse({ items: [{ agentType: "codex", displayName: "Codex" }] });
    }
    if (path === "/machines") {
      return jsonResponse({ items: [{ id: "machine-01", name: "Machine 01", status: "online", runtimeStatus: "running" }] });
    }
    if (path === "/settings/agents/codex/global") {
      return jsonResponse({ document: null });
    }
    if (path === "/settings/machines/machine-01/agents/codex") {
      return jsonResponse({
        machineId: "machine-01",
        agentType: "codex",
        globalDefault: null,
        machineOverride: null,
        usesGlobalDefault: true
      });
    }

    throw new Error(`Unexpected request: ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);

  render(<SettingsPage />);

  fireEvent.change(await screen.findByLabelText("Global Default TOML"), {
    target: { value: "model = [" }
  });
  fireEvent.click(screen.getByRole("button", { name: "Save Global Default" }));

  expect(await screen.findByText("Invalid TOML content.")).toBeInTheDocument();
  expect(fetchMock).not.toHaveBeenCalledWith(
    "/settings/agents/codex/global",
    expect.objectContaining({ method: "PUT" })
  );
});

test("empty toml blocks saving", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = typeof input === "string" ? input : input.toString();
    if (path === "/settings/agents") {
      return jsonResponse({ items: [{ agentType: "codex", displayName: "Codex" }] });
    }
    if (path === "/machines") {
      return jsonResponse({ items: [{ id: "machine-01", name: "Machine 01", status: "online", runtimeStatus: "running" }] });
    }
    if (path === "/settings/agents/codex/global") {
      return jsonResponse({ document: null });
    }
    if (path === "/settings/machines/machine-01/agents/codex") {
      return jsonResponse({
        machineId: "machine-01",
        agentType: "codex",
        globalDefault: null,
        machineOverride: null,
        usesGlobalDefault: true
      });
    }

    throw new Error(`Unexpected request: ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);

  render(<SettingsPage />);

  fireEvent.change(await screen.findByLabelText("Global Default TOML"), {
    target: { value: "   " }
  });
  fireEvent.click(screen.getByRole("button", { name: "Save Global Default" }));

  expect(await screen.findByText("Invalid TOML content.")).toBeInTheDocument();
});

test("deleting machine override falls back to global default", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input.toString();
    if (path === "/settings/agents") {
      return jsonResponse({ items: [{ agentType: "codex", displayName: "Codex" }] });
    }
    if (path === "/machines") {
      return jsonResponse({ items: [{ id: "machine-01", name: "Machine 01", status: "online", runtimeStatus: "running" }] });
    }
    if (path === "/settings/agents/codex/global") {
      return jsonResponse({
        document: { agentType: "codex", format: "toml", content: "model = \"gpt-5.4\"\n" }
      });
    }
    if (path === "/settings/machines/machine-01/agents/codex" && (!init?.method || init.method === "GET")) {
      return jsonResponse({
        machineId: "machine-01",
        agentType: "codex",
        globalDefault: { agentType: "codex", format: "toml", content: "model = \"gpt-5.4\"\n" },
        machineOverride: { agentType: "codex", format: "toml", content: "model = \"gpt-5.2\"\n" },
        usesGlobalDefault: false
      });
    }
    if (path === "/settings/machines/machine-01/agents/codex" && init?.method === "DELETE") {
      return new Response(null, { status: 204 });
    }

    throw new Error(`Unexpected request: ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);

  render(<SettingsPage />);

  fireEvent.click(await screen.findByRole("button", { name: "Delete Machine Override" }));

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      "/settings/machines/machine-01/agents/codex",
      expect.objectContaining({
        method: "DELETE"
      })
    );
  });
  expect(await screen.findByText("Machine override deleted.")).toBeInTheDocument();
  expect(screen.getByText("Using Global Default")).toBeInTheDocument();
});

test("shows load error when settings bootstrap fails", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = typeof input === "string" ? input : input.toString();
    if (path === "/settings/agents") {
      throw new Error("boom");
    }
    throw new Error(`Unexpected request: ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);

  render(<SettingsPage />);

  expect(await screen.findByText("Unable to load settings.")).toBeInTheDocument();
});

test("shows machine settings error when machine fetch fails", async () => {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = typeof input === "string" ? input : input.toString();
    if (path === "/settings/agents") {
      return jsonResponse({ items: [{ agentType: "codex", displayName: "Codex" }] });
    }
    if (path === "/machines") {
      return jsonResponse({ items: [{ id: "machine-01", name: "Machine 01", status: "online", runtimeStatus: "running" }] });
    }
    if (path === "/settings/agents/codex/global") {
      return jsonResponse({ document: null });
    }
    if (path === "/settings/machines/machine-01/agents/codex") {
      throw new Error("boom");
    }
    throw new Error(`Unexpected request: ${path}`);
  });
  vi.stubGlobal("fetch", fetchMock);

  render(<SettingsPage />);

  expect(await screen.findByText("Unable to load machine settings.")).toBeInTheDocument();
});

function jsonResponse(value: unknown): Response {
  return new Response(JSON.stringify(value), {
    status: 200,
    headers: {
      "Content-Type": "application/json"
    }
  });
}
