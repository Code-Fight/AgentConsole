import "@testing-library/jest-dom/vitest";
import { act, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

const connectConsoleSocketMock = vi.fn();

vi.mock("../common/api/ws", () => ({
  connectConsoleSocket: (
    threadId: string | undefined,
    onMessage: (event: MessageEvent<string>) => void,
  ) => connectConsoleSocketMock(threadId, onMessage)
}));

import { OverviewPage } from "./overview-page";

afterEach(() => {
  connectConsoleSocketMock.mockReset();
  vi.unstubAllGlobals();
});

test("renders derived machine counts from the machines response", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () =>
      new Response(
        JSON.stringify({
          items: [
            { id: "machine-1", name: "Alpha", status: "online" },
            { id: "machine-2", name: "Bravo", status: "offline" },
            { id: "machine-3", name: "Charlie", status: "reconnecting" },
            { id: "machine-4", name: "Delta", status: "online" }
          ]
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json"
          }
        },
      ),
    ),
  );

  render(<OverviewPage />);

  expect(await screen.findByText("4")).toBeInTheDocument();
  expect(screen.getByText("Total machines")).toBeInTheDocument();
  expect(screen.getByText("Online")).toBeInTheDocument();
  expect(screen.getByText("Offline")).toBeInTheDocument();
  expect(screen.getByText("Reconnecting")).toBeInTheDocument();
  expect(screen.getByText("2")).toBeInTheDocument();
  expect(screen.getAllByText("1")).toHaveLength(2);
});

test("refreshes when machine.updated arrives", async () => {
  const fetchMock = vi.fn<
    (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>
  >();
  fetchMock
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          items: [{ id: "machine-1", name: "Alpha", status: "online" }]
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json"
          }
        },
      ),
    )
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          items: [
            { id: "machine-1", name: "Alpha", status: "online" },
            { id: "machine-2", name: "Bravo", status: "online" }
          ]
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json"
          }
        },
      ),
    );
  vi.stubGlobal("fetch", fetchMock);
  connectConsoleSocketMock.mockReturnValue(() => {});

  render(<OverviewPage />);

  expect(await screen.findByText("Total machines")).toBeInTheDocument();
  await screen.findAllByText("1");

  const onMessage = connectConsoleSocketMock.mock.calls[0]?.[1] as
    | ((event: MessageEvent<string>) => void)
    | undefined;
  expect(onMessage).toBeTypeOf("function");

  await act(async () => {
    onMessage?.({
      data: JSON.stringify({
        category: "event",
        name: "machine.updated",
        timestamp: "2026-04-09T12:00:00Z",
        payload: {
          machine: { id: "machine-2", name: "Bravo", status: "online" }
        },
        version: "v1"
      })
    } as MessageEvent<string>);
  });

  await screen.findAllByText("2");
  expect(fetchMock).toHaveBeenCalledTimes(2);
});
