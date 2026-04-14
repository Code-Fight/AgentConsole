import { Route, Routes, useLocation, useNavigate, useParams } from "react-router-dom";
import App from "../design-source/App";
import { type AppPage, useConsoleHost } from "./use-console-host";

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

function ConsoleHostEntry() {
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

export function ConsoleHostRouter() {
  return (
    <Routes>
      <Route path="/" element={<ConsoleHostEntry />} />
      <Route path="/overview" element={<ConsoleHostEntry />} />
      <Route path="/threads/:threadId" element={<ConsoleHostEntry />} />
      <Route path="/machines" element={<ConsoleHostEntry />} />
      <Route path="/environment" element={<ConsoleHostEntry />} />
      <Route path="/settings" element={<ConsoleHostEntry />} />
      <Route path="*" element={<ConsoleHostEntry />} />
    </Routes>
  );
}
