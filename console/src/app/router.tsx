import { createBrowserRouter } from "react-router-dom";
import {
  DesignAppShell,
  EnvironmentPageView,
  MachinesPageView,
  SettingsPageView,
  ThreadHubPage,
  ThreadWorkspacePageView
} from "../design";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <DesignAppShell />,
    children: [
      {
        index: true,
        element: <ThreadHubPage />
      },
      {
        path: "machines",
        element: <MachinesPageView />
      },
      {
        path: "threads",
        element: <ThreadHubPage />
      },
      {
        path: "threads/:threadId",
        element: <ThreadWorkspacePageView />
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
