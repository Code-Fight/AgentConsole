import {
  createBrowserRouter,
  createMemoryRouter,
  useLocation,
  useNavigate,
  useParams,
} from "react-router-dom";
import App from "../../design-source/App";
import { type AppPage, useConsoleHost } from "../../design-host/use-console-host";
import { ThreadWorkspacePage } from "../../features/threads/pages/thread-workspace-page";
import { ThreadsPage } from "../../features/threads/pages/threads-page";
import { AppShell } from "../layout/app-shell";

type AppRouterOptions = {
  initialEntries?: string[];
};

function resolveActivePage(pathname: string): AppPage {
  if (pathname.startsWith("/overview")) {
    return "overview";
  }
  if (pathname.startsWith("/machines")) {
    return "machines";
  }
  if (pathname.startsWith("/environment")) {
    return "environment";
  }
  if (pathname.startsWith("/settings")) {
    return "settings";
  }
  return "threads";
}

function LegacyConsolePage() {
  const location = useLocation();
  const navigate = useNavigate();
  const { threadId } = useParams<{ threadId?: string }>();
  const activePage = resolveActivePage(location.pathname);
  const host = useConsoleHost({
    activePage,
    threadId: threadId ?? null,
    navigate,
  });

  return <App {...host} />;
}

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
        element: <LegacyConsolePage />,
      },
      {
        path: "environment",
        element: <LegacyConsolePage />,
      },
      {
        path: "settings/*",
        element: <LegacyConsolePage />,
      },
      {
        path: "overview",
        element: <LegacyConsolePage />,
      },
      {
        path: "*",
        element: <LegacyConsolePage />,
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
