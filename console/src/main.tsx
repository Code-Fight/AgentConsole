import { renderApp } from "./app/entry/main";
import "./common/ui/console.css";

const rootElement = document.getElementById("root");

if (!rootElement) {
  throw new Error("Root element not found");
}

renderApp(rootElement);
