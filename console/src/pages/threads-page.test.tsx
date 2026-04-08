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
