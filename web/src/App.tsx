import { BrowserRouter, Routes, Route, Link } from "react-router-dom";
import Dashboard from "./pages/Dashboard";
import Subscriptions from "./pages/Subscriptions";
import Nodes from "./pages/Nodes";

export default function App() {
  return (
    <BrowserRouter>
      <nav style={{ padding: "1rem", borderBottom: "1px solid #eee" }}>
        <Link to="/" style={{ marginRight: "1rem" }}>Dashboard</Link>
        <Link to="/subscriptions" style={{ marginRight: "1rem" }}>Subscriptions</Link>
        <Link to="/nodes">Nodes</Link>
      </nav>
      <main style={{ padding: "1rem" }}>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/subscriptions" element={<Subscriptions />} />
          <Route path="/nodes" element={<Nodes />} />
        </Routes>
      </main>
    </BrowserRouter>
  );
}
