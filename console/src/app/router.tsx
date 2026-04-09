import { createBrowserRouter } from "react-router-dom";
import { AppShell } from "./shell";
import { EnvironmentPage } from "../pages/environment-page";
import { MachinesPage } from "../pages/machines-page";
import { OverviewPage } from "../pages/overview-page";
import { SettingsPage } from "../pages/settings-page";
import { ThreadWorkspacePage } from "../pages/thread-workspace-page";
import { ThreadsPage } from "../pages/threads-page";

export const router = createBrowserRouter([
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
        path: "threads",
        element: <ThreadsPage />
      },
      {
        path: "threads/:threadId",
        element: <ThreadWorkspacePage />
      },
      {
        path: "environment",
        element: <EnvironmentPage />
      },
      {
        path: "settings",
        element: <SettingsPage />
      }
    ]
  }
]);

export const appRouter = router;
