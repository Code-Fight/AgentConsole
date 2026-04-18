import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { AppProviders } from "../providers/index";
import { createAppRouter } from "../router/index";

export function renderApp(rootElement: HTMLElement) {
  createRoot(rootElement).render(
    <StrictMode>
      <AppProviders router={createAppRouter()} />
    </StrictMode>,
  );
}
