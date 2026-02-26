import { useRuntimeStatus, useRuntimeReload } from "../hooks/useRuntime";
import { useSubscriptions } from "../hooks/useSubscriptions";
import { useNodes } from "../hooks/useNodes";
import { ErrorState } from "../components/common/ErrorState";
import { Link } from "react-router-dom";

export default function Dashboard() {
  const {
    data: runtime,
    isLoading: runtimeLoading,
    error: runtimeError,
  } = useRuntimeStatus();
  const reload = useRuntimeReload();
  const { data: subs } = useSubscriptions();
  const { data: nodes } = useNodes({});
  const runtimeState = runtimeLoading
    ? "Loading"
    : runtimeError
      ? "Offline"
      : runtime
        ? "Online"
        : "Unknown";
  const runtimeTone = runtimeError
    ? "danger"
    : runtimeLoading
      ? "warning"
      : runtime
        ? "success"
        : "muted";
  const configHash = runtime?.config_hash ? runtime.config_hash.slice(0, 8) : "--";

  return (
    <div className="bp-page bp-dashboard">
      <section className="bp-dashboard-hero">
        <div>
          <p className="bp-eyebrow">BoxPilot</p>
          <h1 className="bp-page-title">Dashboard</h1>
          <p className="bp-subtitle">Runtime overview, subscriptions, and node health.</p>
        </div>
        <div className="bp-hero-actions">
          <span className={`bp-badge bp-badge--${runtimeTone}`}>Status: {runtimeState}</span>
          <button
            className="bp-btn-primary bp-btn-primary--wide"
            onClick={() => reload.mutate()}
            disabled={reload.isPending || runtimeLoading}
          >
            {reload.isPending ? "Reloading..." : "Reload runtime"}
          </button>
        </div>
      </section>

      <div className="bp-dashboard-grid">
        <div className="bp-card bp-dashboard-card bp-dashboard-card--wide">
          <div className="bp-card-header">
            <div>
              <p className="bp-card-kicker">System</p>
              <h2 className="bp-card-title">Runtime</h2>
            </div>
            <span className={`bp-badge bp-badge--${runtimeTone}`}>{runtimeState}</span>
          </div>

          {runtimeLoading && !runtime && (
            <p className="bp-muted">Loading runtime status...</p>
          )}
          {runtimeError && (
            <ErrorState
              message={`Failed to load runtime status: ${(runtimeError as Error).message}`}
            />
          )}
          {runtime && !runtimeError && (
            <div className="bp-runtime-grid">
              <div className="bp-runtime-item">
                <span className="bp-runtime-label">Config version</span>
                <span className="bp-runtime-value">{runtime.config_version}</span>
              </div>
              <div className="bp-runtime-item">
                <span className="bp-runtime-label">Hash</span>
                <span className="bp-runtime-value bp-mono">{configHash}</span>
              </div>
              <div className="bp-runtime-item">
                <span className="bp-runtime-label">HTTP Port</span>
                <span className="bp-runtime-value">{runtime.ports.http}</span>
              </div>
              <div className="bp-runtime-item">
                <span className="bp-runtime-label">SOCKS Port</span>
                <span className="bp-runtime-value">{runtime.ports.socks}</span>
              </div>
            </div>
          )}
        </div>

        <div className="bp-card bp-dashboard-card">
          <div className="bp-card-header">
            <div>
              <p className="bp-card-kicker">Inventory</p>
              <h2 className="bp-card-title">Subscriptions</h2>
            </div>
            <Link to="/subscriptions" className="bp-link-pill">
              View All
            </Link>
          </div>
          <div className="bp-stat">
            <span className="bp-stat-value">{subs?.length ?? 0}</span>
            <span className="bp-stat-label">Total active subscriptions</span>
          </div>
          <p className="bp-muted">Track usage and refresh subscriptions in one place.</p>
        </div>

        <div className="bp-card bp-dashboard-card">
          <div className="bp-card-header">
            <div>
              <p className="bp-card-kicker">Topology</p>
              <h2 className="bp-card-title">Nodes</h2>
            </div>
            <Link to="/nodes" className="bp-link-pill">
              View All
            </Link>
          </div>
          <div className="bp-stat">
            <span className="bp-stat-value">{nodes?.length ?? 0}</span>
            <span className="bp-stat-label">Nodes configured</span>
          </div>
          <p className="bp-muted">Quickly validate node health and routing.</p>
        </div>
      </div>
    </div>
  );
}
