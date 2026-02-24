import { BrowserRouter, Routes, Route, NavLink } from "react-router-dom";
import Dashboard from "./pages/Dashboard";
import Subscriptions from "./pages/Subscriptions";
import Nodes from "./pages/Nodes";

export default function App() {
  return (
    <BrowserRouter>
      <div className="bp-shell">
        <nav className="bp-nav">
          <NavLink
            to="/"
            end
            className={({ isActive }) =>
              isActive ? "bp-nav-link bp-nav-link-active" : "bp-nav-link"
            }
          >
            Dashboard
          </NavLink>
          <NavLink
            to="/subscriptions"
            className={({ isActive }) =>
              isActive ? "bp-nav-link bp-nav-link-active" : "bp-nav-link"
            }
          >
            Subscriptions
          </NavLink>
          <NavLink
            to="/nodes"
            className={({ isActive }) =>
              isActive ? "bp-nav-link bp-nav-link-active" : "bp-nav-link"
            }
          >
            Nodes
          </NavLink>
        </nav>
        <main className="bp-main">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/subscriptions" element={<Subscriptions />} />
            <Route path="/nodes" element={<Nodes />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  );
}
