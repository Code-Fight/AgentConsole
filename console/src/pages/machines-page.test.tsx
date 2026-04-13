import "@testing-library/jest-dom/vitest";
import { act, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

const connectConsoleSocketMock = vi.fn();

vi.mock("../common/api/ws", () => ({
  connectConsoleSocket: (
    threadId: string | undefined,
    onMessage: (event: MessageEvent<string>) => void,
  ) => connectConsoleSocketMock(threadId, onMessage),
}));

import { MachinesPage } from "./machines-page";

afterEach(() => {
  connectConsoleSocketMock.mockReset();
  vi.unstubAllGlobals();
});

test("renders the design machines surface with disabled unsupported lifecycle actions", async () => {
  connectConsoleSocketMock.mockReturnValue(() => {});
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const path = typeof input === "string" ? input : input.toString();

      if (path === "/machines") {
        return new Response(
          JSON.stringify({
            items: [
              {
                id: "machine-01",
                name: "Machine 01",
                status: "online",
                runtimeStatus: "running",
              },
            ],
          }),
          {
            status: 200,
            headers: {
              "Content-Type": "application/json",
            },
          },
        );
      }

      throw new Error(`Unexpected request: ${path}`);
    }),
  );

  render(<MachinesPage />);

  expect(await screen.findByRole("heading", { name: "Machines" })).toBeInTheDocument();
  expect(await screen.findByText("Machine 01")).toBeInTheDocument();
  expect(screen.getByText("Runtime: running")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Install agent" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "Remove agent" })).toBeDisabled();
  expect(screen.getAllByText("Not connected")).toHaveLength(2);
});

test("reloads machines after machine.updated websocket events", async () => {
  const socketListeners: Array<(event: MessageEvent<string>) => void> = [];
  connectConsoleSocketMock.mockImplementation(
    (_threadId: string | undefined, onMessage: (event: MessageEvent<string>) => void) => {
      socketListeners.push(onMessage);
      return () => {};
    },
  );

  const fetchMock = vi
    .fn<(input: RequestInfo | URL) => Promise<Response>>()
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          items: [
            {
              id: "machine-01",
              name: "Machine 01",
              status: "online",
              runtimeStatus: "running",
            },
          ],
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json",
          },
        },
      ),
    )
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          items: [
            {
              id: "machine-01",
              name: "Machine 01",
              status: "reconnecting",
              runtimeStatus: "running",
            },
          ],
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json",
          },
        },
      ),
    );
  vi.stubGlobal("fetch", fetchMock);

  render(<MachinesPage />);

  expect(await screen.findByText("online")).toBeInTheDocument();

  await act(async () => {
    socketListeners[0]?.(
      new MessageEvent("message", {
        data: JSON.stringify({
          version: "v1",
          category: "event",
          name: "machine.updated",
          timestamp: "2026-04-13T10:00:00Z",
          payload: {
            machine: {
              id: "machine-01",
              name: "Machine 01",
              status: "reconnecting",
              runtimeStatus: "running",
            },
          },
        }),
      }),
    );
  });

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(screen.getByText("reconnecting")).toBeInTheDocument();
  });
});
