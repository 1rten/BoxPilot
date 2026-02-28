import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { Input, Table, Tag } from "antd";
import { SearchOutlined } from "@ant-design/icons";
import { useRuntimeStatus, useRuntimeTraffic, useRuntimeConnections, useRuntimeLogs } from "../hooks/useRuntime";
import { useSubscriptions } from "../hooks/useSubscriptions";
import { useNodes } from "../hooks/useNodes";
import { useForwardingSummary, useRoutingSummary } from "../hooks/useProxySettings";
import { ErrorState } from "../components/common/ErrorState";
import { formatDateTime } from "../utils/datetime";
import type { ColumnsType } from "antd/es/table";
import type { RuntimeConnection } from "../api/types";

export default function Dashboard() {
  const {
    data: runtime,
    isLoading: runtimeLoading,
    error: runtimeError,
  } = useRuntimeStatus();
  const { data: traffic } = useRuntimeTraffic();
  const { data: subs } = useSubscriptions();
  const { data: nodes } = useNodes({});
  const { data: forwardingSummary } = useForwardingSummary();
  const { data: routingSummary } = useRoutingSummary();
  const { data: logsData } = useRuntimeLogs({ level: "all", limit: 12 });
  const [connQuery, setConnQuery] = useState("");
  const { data: connectionsData, isLoading: connectionsLoading } = useRuntimeConnections(connQuery);

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

  const forwardingTone =
    forwardingSummary?.status === "running"
      ? "success"
      : forwardingSummary?.status === "error"
        ? "danger"
        : "muted";

  const connections = connectionsData?.items ?? [];
  const connectionColumns: ColumnsType<RuntimeConnection> = useMemo(
    () => [
      {
        title: "Node",
        dataIndex: "node_name",
        key: "node_name",
        sorter: (a, b) => a.node_name.localeCompare(b.node_name),
        render: (name: string, row) => (
          <div>
            <div>{name}</div>
            <span className="bp-muted bp-table-mono">{row.node_type.toUpperCase()}</span>
          </div>
        ),
      },
      {
        title: "Target",
        dataIndex: "target",
        key: "target",
        className: "bp-table-mono",
      },
      {
        title: "Status",
        dataIndex: "status",
        key: "status",
        width: 120,
        sorter: (a, b) => a.status.localeCompare(b.status),
        render: (value: string) => (
          <Tag color={value === "ok" ? "success" : value === "error" ? "error" : "processing"}>
            {value.toUpperCase()}
          </Tag>
        ),
      },
      {
        title: "Latency",
        dataIndex: "latency_ms",
        key: "latency_ms",
        width: 120,
        sorter: (a, b) => (a.latency_ms ?? Number.MAX_SAFE_INTEGER) - (b.latency_ms ?? Number.MAX_SAFE_INTEGER),
        render: (v: number | null | undefined) => (v || v === 0 ? `${v} ms` : "-"),
      },
      {
        title: "Last Test",
        dataIndex: "last_test_at",
        key: "last_test_at",
        className: "bp-table-mono",
        sorter: (a, b) => (a.last_test_at || "").localeCompare(b.last_test_at || ""),
        render: (v: string | null | undefined) => (v ? formatDateTime(v) : "-"),
      },
    ],
    []
  );

  return (
    <div className="bp-page bp-dashboard">
      <section className="bp-dashboard-hero">
        <div>
          <p className="bp-eyebrow">BoxPilot</p>
          <h1 className="bp-page-title">Dashboard</h1>
          <p className="bp-subtitle">Runtime overview, forwarding status, and live diagnostics.</p>
        </div>
        <div className="bp-hero-actions">
          <span className={`bp-badge bp-badge--${runtimeTone}`}>Runtime: {runtimeState}</span>
          <span className={`bp-badge bp-badge--${forwardingTone}`}>
            Forwarding: {(forwardingSummary?.status || "stopped").toUpperCase()}
          </span>
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
              <div className="bp-runtime-item">
                <span className="bp-runtime-label">Last Reload</span>
                <span className="bp-runtime-value bp-mono">
                  {runtime.last_reload_at ? formatDateTime(runtime.last_reload_at) : "-"}
                </span>
              </div>
              <div className="bp-runtime-item">
                <span className="bp-runtime-label">Last Error</span>
                <span className="bp-runtime-value">{runtime.last_reload_error || "-"}</span>
              </div>
            </div>
          )}
        </div>

        <div className="bp-card bp-dashboard-card">
          <div className="bp-card-header">
            <div>
              <p className="bp-card-kicker">Forwarding</p>
              <h2 className="bp-card-title">Traffic</h2>
            </div>
            <span className="bp-link-pill">{traffic?.source || "unknown"}</span>
          </div>
          <div className="bp-runtime-grid">
            <div className="bp-runtime-item">
              <span className="bp-runtime-label">Download Rate</span>
              <span className="bp-runtime-value">{formatRate(traffic?.rx_rate_bps ?? 0)}</span>
            </div>
            <div className="bp-runtime-item">
              <span className="bp-runtime-label">Upload Rate</span>
              <span className="bp-runtime-value">{formatRate(traffic?.tx_rate_bps ?? 0)}</span>
            </div>
            <div className="bp-runtime-item">
              <span className="bp-runtime-label">RX Total</span>
              <span className="bp-runtime-value">{formatBytes(traffic?.rx_total_bytes ?? 0)}</span>
            </div>
            <div className="bp-runtime-item">
              <span className="bp-runtime-label">TX Total</span>
              <span className="bp-runtime-value">{formatBytes(traffic?.tx_total_bytes ?? 0)}</span>
            </div>
          </div>
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
          <p className="bp-muted">
            Forwarding selected: {forwardingSummary?.selected_nodes_count ?? 0}
          </p>
        </div>

        <div className="bp-card bp-dashboard-card bp-dashboard-card--wide">
          <div className="bp-card-header">
            <div>
              <p className="bp-card-kicker">Routing</p>
              <h2 className="bp-card-title">Routing / Geo</h2>
            </div>
            <Link to="/settings" className="bp-link-pill">
              Edit Rules
            </Link>
          </div>
          <div className="bp-runtime-grid">
            <div className="bp-runtime-item">
              <span className="bp-runtime-label">Bypass Private</span>
              <span className="bp-runtime-value">
                {routingSummary?.bypass_private_enabled ? "Enabled" : "Disabled"}
              </span>
            </div>
            <div className="bp-runtime-item">
              <span className="bp-runtime-label">Bypass Domains</span>
              <span className="bp-runtime-value">{routingSummary?.bypass_domains_count ?? 0}</span>
            </div>
            <div className="bp-runtime-item">
              <span className="bp-runtime-label">Bypass CIDRs</span>
              <span className="bp-runtime-value">{routingSummary?.bypass_cidrs_count ?? 0}</span>
            </div>
            <div className="bp-runtime-item">
              <span className="bp-runtime-label">GeoIP / GeoSite</span>
              <span className="bp-runtime-value">
                {routingSummary?.geoip_status || "unknown"} / {routingSummary?.geosite_status || "unknown"}
              </span>
            </div>
          </div>
          {routingSummary?.notes?.length ? (
            <div className="bp-list-compact" style={{ marginTop: 12 }}>
              {routingSummary.notes.slice(0, 2).map((note) => (
                <p key={note} className="bp-muted">
                  - {note}
                </p>
              ))}
            </div>
          ) : null}
        </div>

        <div className="bp-card bp-dashboard-card bp-dashboard-card--wide">
          <div className="bp-card-header">
            <div>
              <p className="bp-card-kicker">Diagnostics</p>
              <h2 className="bp-card-title">Runtime Logs</h2>
            </div>
            <span className="bp-muted">Recent {Math.min(logsData?.items?.length ?? 0, 12)}</span>
          </div>
          {logsData?.items?.length ? (
            <div className="bp-log-list">
              {logsData.items.slice(0, 8).map((item) => (
                <div
                  key={`${item.timestamp}-${item.level}-${item.source}-${item.message}`}
                  className="bp-log-item"
                >
                  <div className="bp-log-head">
                    <span className="bp-table-mono">{formatDateTime(item.timestamp)}</span>
                    <Tag color={item.level === "error" ? "error" : item.level === "warn" ? "warning" : "processing"}>
                      {item.level.toUpperCase()}
                    </Tag>
                  </div>
                  <p className="bp-log-message">
                    <span className="bp-table-mono">[{item.source}]</span> {item.message}
                  </p>
                </div>
              ))}
            </div>
          ) : (
            <p className="bp-muted">No runtime logs yet.</p>
          )}
        </div>

        <div className="bp-card bp-dashboard-card bp-dashboard-card--full">
          <div className="bp-card-header">
            <div>
              <p className="bp-card-kicker">Diagnostics</p>
              <h2 className="bp-card-title">Connections</h2>
            </div>
            <span className="bp-muted">Active {connectionsData?.active_count ?? 0}</span>
          </div>
          <div className="bp-dashboard-table-toolbar">
            <Input
              className="bp-input bp-search-input"
              value={connQuery}
              onChange={(e) => setConnQuery(e.target.value)}
              allowClear
              prefix={<SearchOutlined style={{ color: "#94a3b8" }} />}
              placeholder="Filter connections by node/target/status"
            />
          </div>
          <Table<RuntimeConnection>
            rowKey="id"
            size="small"
            loading={connectionsLoading}
            dataSource={connections}
            columns={connectionColumns}
            pagination={{ pageSize: 5, showSizeChanger: false }}
          />
        </div>
      </div>
    </div>
  );
}

function formatRate(value: number): string {
  return `${formatBytes(value)}/s`;
}

function formatBytes(value: number): string {
  const units = ["B", "KB", "MB", "GB", "TB"];
  let v = Math.max(0, value);
  let idx = 0;
  while (v >= 1024 && idx < units.length - 1) {
    v /= 1024;
    idx++;
  }
  const formatted = idx === 0 ? v.toFixed(0) : v.toFixed(1);
  return `${formatted} ${units[idx]}`;
}
