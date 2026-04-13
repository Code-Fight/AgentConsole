import { createBrowserRouter } from "react-router-dom";
import {
  DesignAppShell,
  EnvironmentPageView,
  MachinesPageView,
  SettingsPageView
} from "../design";
import { ThreadWorkspacePage } from "../pages/thread-workspace-page";
import { ThreadsPage } from "../pages/threads-page";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <DesignAppShell />,
    children: [
      {
        index: true,
        element: <ThreadsPage />
      },
      {
        path: "machines",
        element: <MachinesPageView />
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
        element: <EnvironmentPageView />
      },
      {
        path: "settings",
        element: <SettingsPageView />
      }
    ]
  }
]);

export const appRouter = router;
