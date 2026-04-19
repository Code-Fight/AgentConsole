import {
  createBrowserRouter,
  createMemoryRouter,
  Navigate,
} from "react-router-dom";
import { EnvironmentPage } from "../../features/environment/pages/environment-page";
import { MachinesPage } from "../../features/machines/pages/machines-page";
import { OverviewPage } from "../../features/overview/pages/overview-page";
import { SettingsPage } from "../../features/settings/pages/settings-page";
import { ThreadWorkspacePage } from "../../features/threads/pages/thread-workspace-page";
import { ThreadsPage } from "../../features/threads/pages/threads-page";
import { AppShell } from "../layout/app-shell";

type AppRouterOptions = {
  initialEntries?: string[];
};

const routes = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      {
        index: true,
        element: <ThreadsPage />,
      },
      {
        path: "threads",
        element: <ThreadsPage />,
      },
      {
        path: "threads/:threadId",
        element: <ThreadWorkspacePage />,
      },
      {
        path: "machines",
        element: <MachinesPage />,
      },
      {
        path: "environment",
        element: <EnvironmentPage />,
      },
      {
        path: "settings/*",
        element: <SettingsPage />,
      },
      {
        path: "overview",
        element: <OverviewPage />,
      },
      {
        path: "*",
        element: <Navigate replace to="/" />,
      },
    ],
  },
];

export function createAppRouter(
  options: AppRouterOptions = {},
): ReturnType<typeof createBrowserRouter> {
  if (options.initialEntries) {
    return createMemoryRouter(routes, { initialEntries: options.initialEntries });
  }
  return createBrowserRouter(routes);
}
