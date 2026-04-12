import { useState } from "react";
import { NavLink, Outlet, useLocation, useMatch } from "react-router-dom";
import { ThreadHubPanel } from "../components/thread-hub-panel";

function isThreadSurface(pathname: string): boolean {
  return pathname === "/" || pathname === "/threads" || pathname.startsWith("/threads/");
}

export function DesignAppShell() {
  const location = useLocation();
  const threadMatch = useMatch("/threads/:threadId");
  const [mobilePanelOpen, setMobilePanelOpen] = useState(false);
  const showHub = isThreadSurface(location.pathname);
  const threadId = threadMatch?.params.threadId;

  return (
    <div className="thread-shell">
      <header className="thread-shell-mobile-header">
        {showHub ? (
          <button
            type="button"
            aria-label={mobilePanelOpen ? "Close thread hub" : "Open thread hub"}
            className="thread-shell-icon-button"
            onClick={() => setMobilePanelOpen((current) => !current)}
          >
            {mobilePanelOpen ? "×" : "☰"}
          </button>
        ) : (
          <NavLink to="/threads" aria-label="Back to thread hub" className="thread-shell-icon-button">
            ←
          </NavLink>
        )}
        <div className="thread-shell-mobile-brand">
          <div className="thread-shell-brand-mark">CA</div>
          <div>
            <strong>{showHub ? "Thread Hub" : "Management"}</strong>
            <span>{showHub ? "Design source shell" : "Design source pages"}</span>
          </div>
        </div>
      </header>

      {showHub && mobilePanelOpen ? (
        <div className="thread-shell-mobile-overlay" onClick={() => setMobilePanelOpen(false)}>
          <aside className="thread-shell-mobile-drawer" onClick={(event) => event.stopPropagation()}>
            <ThreadHubPanel activeThreadId={threadId} onNavigate={() => setMobilePanelOpen(false)} />
          </aside>
        </div>
      ) : null}

      <div className={`thread-shell-desktop ${showHub ? "thread-shell-desktop-thread" : "thread-shell-desktop-management"}`}>
        {showHub ? (
          <aside className="thread-shell-panel">
            <ThreadHubPanel activeThreadId={threadId} />
          </aside>
        ) : (
          <aside className="thread-shell-backbar">
            <NavLink to="/threads" aria-label="Back to thread hub" className="thread-shell-back-button">
              ←
            </NavLink>
          </aside>
        )}
        <div className="thread-shell-main">
          <main className="thread-shell-workspace">
            <Outlet />
          </main>
        </div>
      </div>
    </div>
  );
}
