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
  expect(screen.getByText("Threads")).toBeInTheDocument();
  expect(screen.getByText("Environment")).toBeInTheDocument();
  expect(screen.getByRole("link", { name: "Threads" })).toBeInTheDocument();
});
