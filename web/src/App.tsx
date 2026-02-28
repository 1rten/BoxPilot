import { BrowserRouter, Routes, Route, NavLink } from "react-router-dom";
import Dashboard from "./pages/Dashboard";
import Subscriptions from "./pages/Subscriptions";
import Nodes from "./pages/Nodes";
import Settings from "./pages/Settings";
import type { ForwardingSummaryNode } from "./api/types";
import {
  useForwardingSummary,
  useStartForwardingRuntime,
  useStopForwardingRuntime,
} from "./hooks/useProxySettings";
import { Popover } from "antd";

export default function App() {
  const { data: summary, refetch: refetchForwardingSummary } = useForwardingSummary();
  const startForwarding = useStartForwardingRuntime();
  const stopForwarding = useStopForwardingRuntime();
  const toggling = startForwarding.isPending || stopForwarding.isPending;
  const isRunning = !!summary?.running;
  const runtimeStatus = summary?.status ?? "stopped";
  const nodeList = summary?.nodes ?? [];

  return (
    <BrowserRouter>
      <div className="bp-shell">
        <nav className="bp-nav">
          <div className="bp-nav-left">
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
            </div>
          </div>
          <div className="bp-nav-right">
            <Popover
              placement="bottomRight"
              trigger={["hover"]}
              onOpenChange={(open) => {
                if (open) {
                  void refetchForwardingSummary();
                }
              }}
              content={
                <div className="bp-forwarding-popover">
                  <div className="bp-forwarding-popover-head">
                    <span className="bp-forwarding-popover-title">Proxy</span>
                    <span className={`bp-runtime-dot bp-runtime-dot-${runtimeStatus}`} />
                  </div>
                  <p className="bp-forwarding-popover-meta">
                    Status: {runtimeStatus.toUpperCase()} · Selected: {summary?.selected_nodes_count ?? 0}
                  </p>
                  {summary?.error_message && (
                    <p className="bp-forwarding-popover-error">{summary.error_message}</p>
                  )}
                  {nodeList.length > 0 ? (
                    <div className="bp-forwarding-popover-list">
                      {nodeList.map((node: ForwardingSummaryNode) => (
                        <div key={node.id} className="bp-forwarding-popover-item">
                          <span className="bp-forwarding-popover-name">{node.name || node.tag}</span>
                          <div className="bp-forwarding-popover-side">
                            <span>{node.type.toUpperCase()}</span>
                            {node.last_status ? (
                              <span className={`bp-forwarding-popover-status bp-forwarding-popover-status-${node.last_status.toLowerCase()}`}>
                                {node.last_status.toUpperCase()}
                              </span>
                            ) : (
                              <span className="bp-forwarding-popover-status">UNTESTED</span>
                            )}
                            {node.last_latency_ms !== null && node.last_latency_ms !== undefined
                              ? <span>{node.last_latency_ms}ms</span>
                              : null}
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="bp-forwarding-popover-empty">No forwarding nodes selected.</p>
                  )}
                </div>
              }
            >
              <button
                type="button"
                className={
                  runtimeStatus === "running"
                    ? "bp-forwarding-toggle bp-forwarding-toggle-running"
                    : runtimeStatus === "error"
                    ? "bp-forwarding-toggle bp-forwarding-toggle-error"
                    : "bp-forwarding-toggle"
                }
                disabled={toggling}
                onClick={() => {
                  if (isRunning) {
                    stopForwarding.mutate();
                  } else {
                    startForwarding.mutate();
                  }
                }}
                title={summary?.error_message || "Forwarding runtime control"}
              >
                <span className={`bp-runtime-dot bp-runtime-dot-${runtimeStatus}`} />
                {toggling ? "Applying..." : "Proxy"}
              </button>
            </Popover>
            <NavLink
              to="/settings"
              className={({ isActive }) =>
                isActive ? "bp-settings-link bp-settings-link-active" : "bp-settings-link"
              }
              aria-label="Settings"
              title="Settings"
            >
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M19.14 12.94a7.8 7.8 0 0 0 .05-.94 7.8 7.8 0 0 0-.05-.94l2.03-1.58a.5.5 0 0 0 .12-.65l-1.92-3.32a.5.5 0 0 0-.61-.22l-2.39.96a7.4 7.4 0 0 0-1.62-.94l-.36-2.54a.5.5 0 0 0-.5-.43h-3.84a.5.5 0 0 0-.5.43l-.36 2.54a7.4 7.4 0 0 0-1.62.94l-2.39-.96a.5.5 0 0 0-.61.22L2.71 8.83a.5.5 0 0 0 .12.65l2.03 1.58a7.8 7.8 0 0 0-.05.94 7.8 7.8 0 0 0 .05.94l-2.03 1.58a.5.5 0 0 0-.12.65l1.92 3.32a.5.5 0 0 0 .61.22l2.39-.96c.5.39 1.05.71 1.62.94l.36 2.54a.5.5 0 0 0 .5.43h3.84a.5.5 0 0 0 .5-.43l.36-2.54c.57-.23 1.12-.55 1.62-.94l2.39.96a.5.5 0 0 0 .61-.22l1.92-3.32a.5.5 0 0 0-.12-.65zM12 15.2a3.2 3.2 0 1 1 0-6.4 3.2 3.2 0 0 1 0 6.4z" />
              </svg>
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
