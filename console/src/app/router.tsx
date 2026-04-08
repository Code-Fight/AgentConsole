import { createBrowserRouter } from "react-router-dom";
import { AppShell } from "./shell";
import { EnvironmentPage } from "../pages/environment-page";
import { MachinesPage } from "../pages/machines-page";
import { OverviewPage } from "../pages/overview-page";

export const appRouter = createBrowserRouter([
  {
    path: "/",
    element: <AppShell />,
    children: [
      {
        index: true,
        element: <OverviewPage />
      },
      {
        path: "machines",
        element: <MachinesPage />
      },
      {
        path: "environment",
        element: <EnvironmentPage />
      }
    ]
  }
]);
