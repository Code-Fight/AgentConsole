import { NavLink, Outlet } from "react-router-dom";

export function AppShell() {
  return (
    <div className="shell">
      <aside className="left-nav">
        <NavLink to="/">Overview</NavLink>
        <NavLink to="/machines">Machines</NavLink>
        <NavLink to="/environment">Environment</NavLink>
      </aside>
      <main className="center-pane">
        <Outlet />
      </main>
      <aside className="right-pane">Inspector</aside>
    </div>
  );
}
