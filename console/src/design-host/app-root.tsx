import { BrowserRouter } from "react-router-dom";
import { ConsoleHostRouter } from "./console-host-router";

export function DesignSourceAppRoot() {
  return (
    <div className="dark fixed inset-0 overflow-hidden">
      <BrowserRouter>
        <ConsoleHostRouter />
      </BrowserRouter>
    </div>
  );
}
