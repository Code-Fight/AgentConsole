import "@testing-library/jest-dom/vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, expect, test, vi } from "vitest";
import { ThreadsPage } from "./threads-page";

afterEach(() => {
  vi.unstubAllGlobals();
});

test("shows the live load error without inventing a fallback thread", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => {
      throw new Error("network failure");
    }),
  );

  render(
    <MemoryRouter>
      <ThreadsPage />
    </MemoryRouter>,
  );

  expect(await screen.findByText("Unable to load live threads.")).toBeInTheDocument();

  await waitFor(() => {
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  expect(screen.getByText("No threads available.")).toBeInTheDocument();
});

test("renders the thread status so stale threads remain explicit in the list", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () =>
      new Response(
        JSON.stringify({
          items: [
            {
              threadId: "thread-1",
              machineId: "machine-1",
              status: "unknown",
              title: "Investigate flaky test"
            }
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

  render(
    <MemoryRouter>
      <ThreadsPage />
    </MemoryRouter>,
  );

  expect(await screen.findByRole("link", { name: "Investigate flaky test" })).toBeInTheDocument();
  expect(screen.getByText("unknown")).toBeInTheDocument();
});
