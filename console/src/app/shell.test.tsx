import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { AppShell } from "./shell";

test("renders the console shell", () => {
  render(
    <MemoryRouter>
      <AppShell />
    </MemoryRouter>,
  );

  expect(screen.getByText("Overview")).toBeInTheDocument();
  expect(screen.getByText("Machines")).toBeInTheDocument();
  const threadsLink = screen.getByRole("link", { name: "Threads" });
  expect(screen.getByText("Environment")).toBeInTheDocument();
  expect(threadsLink).toBeInTheDocument();
  expect(threadsLink).toHaveAttribute("href", "/threads");
});
