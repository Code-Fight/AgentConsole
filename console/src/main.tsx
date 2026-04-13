import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { DesignSourceAppRoot } from "./design-host/app-root";
import "./design-source/styles/index.css";

const rootElement = document.getElementById("root");

if (!rootElement) {
  throw new Error("Root element not found");
}

createRoot(rootElement).render(
  <StrictMode>
    <DesignSourceAppRoot />
  </StrictMode>,
);
