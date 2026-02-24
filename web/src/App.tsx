import { BrowserRouter, Routes, Route, NavLink } from "react-router-dom";
import Dashboard from "./pages/Dashboard";
import Subscriptions from "./pages/Subscriptions";
import Nodes from "./pages/Nodes";
import Settings from "./pages/Settings";

export default function App() {
  return (
    <BrowserRouter>
      <div className="bp-shell">
        <nav className="bp-nav">
          <div className="bp-brand">
            <div className="bp-brand-mark">BP</div>
            <div className="bp-brand-name">BoxPilot</div>
          </div>
          <div className="bp-tabs">
            <NavLink
              to="/"
              end
              className={({ isActive }) =>
                isActive ? "bp-tab bp-tab-active" : "bp-tab"
              }
            >
              Dashboard
            </NavLink>
            <NavLink
              to="/subscriptions"
              className={({ isActive }) =>
                isActive ? "bp-tab bp-tab-active" : "bp-tab"
              }
            >
              Subscriptions
            </NavLink>
            <NavLink
              to="/nodes"
              className={({ isActive }) =>
                isActive ? "bp-tab bp-tab-active" : "bp-tab"
              }
            >
              Nodes
            </NavLink>
            <NavLink
              to="/settings"
              className={({ isActive }) =>
                isActive ? "bp-tab bp-tab-active" : "bp-tab"
              }
            >
              Settings
            </NavLink>
          </div>
        </nav>
        <main className="bp-main">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/subscriptions" element={<Subscriptions />} />
            <Route path="/nodes" element={<Nodes />} />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  );
}
