import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { Button, Input, Modal, Select, Table, Tag } from "antd";
import { SearchOutlined } from "@ant-design/icons";
import { useRuntimeStatus, useRuntimeTraffic, useRuntimeConnections, useRuntimeLogs } from "../hooks/useRuntime";
import { useSubscriptions } from "../hooks/useSubscriptions";
import { useNodes } from "../hooks/useNodes";
import { useForwardingSummary, useRoutingSummary } from "../hooks/useProxySettings";
import { ErrorState } from "../components/common/ErrorState";
import { formatDateTime } from "../utils/datetime";
import type { ColumnsType } from "antd/es/table";
import type { RuntimeConnection, RuntimeLogItem } from "../api/types";
import { useI18n } from "../i18n/context";

export default function Dashboard() {
  const { tr } = useI18n();
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
  const { data: logsData, isFetching: logsFetching } = useRuntimeLogs({ level: "all", limit: 12 });
  const [logsModalOpen, setLogsModalOpen] = useState(false);
  const [logsLevel, setLogsLevel] = useState("all");
  const [logsQuery, setLogsQuery] = useState("");
  const {
    data: logsModalData,
    isLoading: logsModalLoading,
    isFetching: logsModalFetching,
  } = useRuntimeLogs({
    level: logsLevel,
    q: logsQuery,
    limit: 200,
    enabled: logsModalOpen,
    refetchIntervalMs: 8000,
  });
  const [connQuery, setConnQuery] = useState("");
  const {
    data: connectionsData,
    isLoading: connectionsLoading,
    isFetching: connectionsFetching,
  } = useRuntimeConnections(connQuery);

  const runtimeStateKey = runtimeLoading
    ? "loading"
    : runtimeError
      ? "offline"
      : runtime
        ? "online"
        : "unknown";
  const runtimeState = tr(`dashboard.runtime.state.${runtimeStateKey}`, runtimeStateKey.toUpperCase());
  const runtimeTone = runtimeError
    ? "danger"
    : runtimeLoading
      ? "warning"
      : runtime
        ? "success"
        : "muted";
  const configHash = runtime?.config_hash ? runtime.config_hash.slice(0, 8) : "--";
  const trafficSourceMeta = getTrafficSourceMeta(traffic?.source, tr);

  const forwardingTone =
    forwardingSummary?.status === "running"
      ? "success"
      : forwardingSummary?.status === "error"
        ? "danger"
        : "muted";
  const forwardingStatus = forwardingSummary?.status || "stopped";
  const forwardingStatusLabel = tr(`app.proxy.runtime.${forwardingStatus}`, forwardingStatus.toUpperCase());

  const connections = connectionsData?.items ?? [];
  const recentLogs = logsData?.items?.slice(0, 8) ?? [];
  const recentLogsTotal = logsData?.items?.length ?? 0;
  const connectionColumns: ColumnsType<RuntimeConnection> = useMemo(
    () => [
      {
        title: tr("dashboard.connections.node", "Node"),
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
        title: tr("dashboard.connections.target", "Target"),
        dataIndex: "target",
        key: "target",
        className: "bp-table-mono",
      },
      {
        title: tr("dashboard.connections.status", "Status"),
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
        title: tr("dashboard.connections.latency", "Latency"),
        dataIndex: "latency_ms",
        key: "latency_ms",
        width: 120,
        sorter: (a, b) => (a.latency_ms ?? Number.MAX_SAFE_INTEGER) - (b.latency_ms ?? Number.MAX_SAFE_INTEGER),
        render: (v: number | null | undefined, row) => (
          <span className={`bp-latency-badge bp-latency-badge-${latencyTone(v, row.status)}`}>
            {formatLatency(v)}
          </span>
        ),
      },
      {
        title: tr("dashboard.connections.last_test", "Last Test"),
        dataIndex: "last_test_at",
        key: "last_test_at",
        className: "bp-table-mono",
        sorter: (a, b) => (a.last_test_at || "").localeCompare(b.last_test_at || ""),
        render: (v: string | null | undefined) => (v ? formatDateTime(v) : "-"),
      },
    ],
    [tr]
  );

  return (
    <div className="bp-page bp-dashboard">
      <section className="bp-dashboard-hero">
        <div>
          <p className="bp-eyebrow">BoxPilot</p>
          <h1 className="bp-page-title">{tr("nav.dashboard", "Dashboard")}</h1>
          <p className="bp-subtitle">{tr("dashboard.subtitle", "Runtime overview, forwarding status, and live diagnostics.")}</p>
        </div>
        <div className="bp-hero-actions">
          <span className={`bp-badge bp-badge--${runtimeTone}`}>{tr("dashboard.runtime", "Runtime")}: {runtimeState}</span>
          <span className={`bp-badge bp-badge--${forwardingTone}`}>
            {tr("dashboard.forwarding", "Forwarding")}: {forwardingStatusLabel}
          </span>
        </div>
      </section>

      <div className="bp-dashboard-grid">
        <div className="bp-card bp-dashboard-card bp-dashboard-card--wide">
          <div className="bp-card-header">
            <div>
              <p className="bp-card-kicker">{tr("dashboard.kicker.system", "System")}</p>
              <h2 className="bp-card-title">{tr("dashboard.runtime", "Runtime")}</h2>
            </div>
            <span className={`bp-badge bp-badge--${runtimeTone}`}>{runtimeState}</span>
          </div>

          {runtimeLoading && !runtime && (
            <p className="bp-muted">{tr("dashboard.runtime.loading", "Loading runtime status...")}</p>
          )}
          {runtimeError && (
            <ErrorState
              message={tr("dashboard.runtime.error", "Failed to load runtime status: {message}", {
                message: (runtimeError as Error).message,
              })}
            />
          )}
          {runtime && !runtimeError && (
            <div className="bp-runtime-grid">
              <div className="bp-runtime-item">
                <span className="bp-runtime-label">{tr("dashboard.runtime.config_version", "Config version")}</span>
                <span className="bp-runtime-value">{runtime.config_version}</span>
              </div>
              <div className="bp-runtime-item">
                <span className="bp-runtime-label">{tr("dashboard.runtime.hash", "Hash")}</span>
                <span className="bp-runtime-value bp-mono">{configHash}</span>
              </div>
              <div className="bp-runtime-item">
                <span className="bp-runtime-label">{tr("dashboard.runtime.http_port", "HTTP Port")}</span>
                <span className="bp-runtime-value">{runtime.ports.http}</span>
              </div>
              <div className="bp-runtime-item">
                <span className="bp-runtime-label">{tr("dashboard.runtime.socks_port", "SOCKS Port")}</span>
                <span className="bp-runtime-value">{runtime.ports.socks}</span>
              </div>
              <div className="bp-runtime-item">
                <span className="bp-runtime-label">{tr("dashboard.runtime.last_reload", "Last Reload")}</span>
                <span className="bp-runtime-value bp-mono">
                  {runtime.last_reload_at ? formatDateTime(runtime.last_reload_at) : "-"}
                </span>
              </div>
              <div className="bp-runtime-item">
                <span className="bp-runtime-label">{tr("dashboard.runtime.last_error", "Last Error")}</span>
                <span className="bp-runtime-value">{runtime.last_reload_error || "-"}</span>
              </div>
            </div>
          )}
        </div>

        <div className="bp-card bp-dashboard-card">
          <div className="bp-card-header">
            <div>
              <p className="bp-card-kicker">{tr("dashboard.kicker.forwarding", "Forwarding")}</p>
              <h2 className="bp-card-title">{tr("dashboard.traffic", "Traffic")}</h2>
            </div>
            <span
              className={`bp-link-pill bp-source-pill bp-source-pill--${trafficSourceMeta.tone}`}
              title={traffic?.source || tr("common.unknown", "unknown")}
            >
              {trafficSourceMeta.label}
            </span>
          </div>
          <p className="bp-muted">{trafficSourceMeta.description}</p>
          <div className="bp-runtime-grid">
            <div className="bp-runtime-item">
              <span className="bp-runtime-label">{tr("dashboard.traffic.download", "Download Rate")}</span>
              <span className="bp-runtime-value">{formatRate(traffic?.rx_rate_bps ?? 0)}</span>
            </div>
            <div className="bp-runtime-item">
              <span className="bp-runtime-label">{tr("dashboard.traffic.upload", "Upload Rate")}</span>
              <span className="bp-runtime-value">{formatRate(traffic?.tx_rate_bps ?? 0)}</span>
            </div>
            <div className="bp-runtime-item">
              <span className="bp-runtime-label">{tr("dashboard.traffic.rx_total", "RX Total")}</span>
              <span className="bp-runtime-value">{formatBytes(traffic?.rx_total_bytes ?? 0)}</span>
            </div>
            <div className="bp-runtime-item">
              <span className="bp-runtime-label">{tr("dashboard.traffic.tx_total", "TX Total")}</span>
              <span className="bp-runtime-value">{formatBytes(traffic?.tx_total_bytes ?? 0)}</span>
            </div>
          </div>
        </div>

        <div className="bp-card bp-dashboard-card">
          <div className="bp-card-header">
            <div>
              <p className="bp-card-kicker">{tr("dashboard.kicker.inventory", "Inventory")}</p>
              <h2 className="bp-card-title">{tr("nav.subscriptions", "Subscriptions")}</h2>
            </div>
            <Link to="/subscriptions" className="bp-link-pill">
              {tr("dashboard.view_all", "View All")}
            </Link>
          </div>
          <div className="bp-stat">
            <span className="bp-stat-value">{subs?.length ?? 0}</span>
            <span className="bp-stat-label">{tr("dashboard.subs.total", "Total active subscriptions")}</span>
          </div>
          <p className="bp-muted">{tr("dashboard.subs.desc", "Track usage and refresh subscriptions in one place.")}</p>
        </div>

        <div className="bp-card bp-dashboard-card">
          <div className="bp-card-header">
            <div>
              <p className="bp-card-kicker">{tr("dashboard.kicker.topology", "Topology")}</p>
              <h2 className="bp-card-title">{tr("nav.nodes", "Nodes")}</h2>
            </div>
            <Link to="/nodes" className="bp-link-pill">
              {tr("dashboard.view_all", "View All")}
            </Link>
          </div>
          <div className="bp-stat">
            <span className="bp-stat-value">{nodes?.length ?? 0}</span>
            <span className="bp-stat-label">{tr("dashboard.nodes.total", "Nodes configured")}</span>
          </div>
          <p className="bp-muted">
            {tr("dashboard.nodes.selected", "Forwarding selected")}: {forwardingSummary?.selected_nodes_count ?? 0}
          </p>
        </div>

        <div className="bp-card bp-dashboard-card bp-dashboard-card--wide">
          <div className="bp-card-header">
            <div>
              <p className="bp-card-kicker">{tr("dashboard.kicker.routing", "Routing")}</p>
              <h2 className="bp-card-title">{tr("dashboard.routing.title", "Routing / Geo")}</h2>
            </div>
            <Link to="/settings" className="bp-link-pill">
              {tr("dashboard.routing.edit", "Edit Rules")}
            </Link>
          </div>
          <div className="bp-runtime-grid">
            <div className="bp-runtime-item">
              <span className="bp-runtime-label">{tr("dashboard.routing.private", "Bypass Private")}</span>
              <span className="bp-runtime-value">{routingSummary?.bypass_private_enabled ? tr("nodes.status.enabled", "Enabled") : tr("nodes.status.disabled", "Disabled")}</span>
            </div>
            <div className="bp-runtime-item">
              <span className="bp-runtime-label">{tr("dashboard.routing.domains", "Bypass Domains")}</span>
              <span className="bp-runtime-value">{routingSummary?.bypass_domains_count ?? 0}</span>
            </div>
            <div className="bp-runtime-item">
              <span className="bp-runtime-label">{tr("dashboard.routing.cidrs", "Bypass CIDRs")}</span>
              <span className="bp-runtime-value">{routingSummary?.bypass_cidrs_count ?? 0}</span>
            </div>
            <div className="bp-runtime-item">
              <span className="bp-runtime-label">{tr("dashboard.routing.geo", "GeoIP / GeoSite")}</span>
              <span className="bp-runtime-value">
                {(routingSummary?.geoip_status || tr("common.unknown", "unknown")).toUpperCase()} / {(routingSummary?.geosite_status || tr("common.unknown", "unknown")).toUpperCase()}
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
              <p className="bp-card-kicker">{tr("dashboard.kicker.diagnostics", "Diagnostics")}</p>
              <h2 className="bp-card-title">{tr("dashboard.logs.title", "Runtime Logs")}</h2>
            </div>
            <div className="bp-card-header-meta">
              <span className="bp-metric-pill bp-metric-pill-neutral">
                {tr("dashboard.logs.showing", "Showing {shown}/{total}", {
                  shown: recentLogs.length,
                  total: recentLogsTotal,
                })}
              </span>
              <span className={logsFetching ? "bp-sync-indicator bp-sync-indicator-fetching" : "bp-sync-indicator"}>
                <span className="bp-sync-indicator-dot" />
                {tr("dashboard.sync", "Sync")}
              </span>
              <Button size="small" onClick={() => setLogsModalOpen(true)}>
                {tr("dashboard.logs.more", "View More")}
              </Button>
            </div>
          </div>
          {recentLogs.length ? (
            <div className="bp-log-list">
              {recentLogs.map((item) => (
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
            <p className="bp-muted">{logsFetching ? tr("dashboard.logs.loading", "Loading runtime logs...") : tr("dashboard.logs.empty", "No runtime logs yet.")}</p>
          )}
        </div>

        <div className="bp-card bp-dashboard-card bp-dashboard-card--full">
          <div className="bp-card-header">
            <div>
              <p className="bp-card-kicker">{tr("dashboard.kicker.diagnostics", "Diagnostics")}</p>
              <h2 className="bp-card-title">{tr("dashboard.connections.title", "Connections")}</h2>
            </div>
            <div className="bp-card-header-meta">
              <span className="bp-metric-pill bp-metric-pill-active">
                {tr("dashboard.connections.active", "Active {count}", {
                  count: connectionsData?.active_count ?? 0,
                })}
              </span>
              <span className={connectionsFetching ? "bp-sync-indicator bp-sync-indicator-fetching" : "bp-sync-indicator"}>
                <span className="bp-sync-indicator-dot" />
                {tr("dashboard.sync", "Sync")}
              </span>
            </div>
          </div>
          <div className="bp-dashboard-table-toolbar">
            <Input
              className="bp-input bp-search-input"
              value={connQuery}
              onChange={(e) => setConnQuery(e.target.value)}
              allowClear
              prefix={<SearchOutlined style={{ color: "#94a3b8" }} />}
              placeholder={tr("dashboard.connections.filter", "Filter connections by node/target/status")}
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
      <Modal
        title={tr("dashboard.logs.title", "Runtime Logs")}
        open={logsModalOpen}
        onCancel={() => setLogsModalOpen(false)}
        footer={null}
        width={980}
        destroyOnClose
      >
        <div className="bp-log-modal-toolbar">
          <Select
            value={logsLevel}
            onChange={setLogsLevel}
            style={{ width: 140 }}
            options={[
              { value: "all", label: tr("dashboard.logs.level.all", "All Levels") },
              { value: "info", label: tr("dashboard.logs.level.info", "Info") },
              { value: "warn", label: tr("dashboard.logs.level.warn", "Warn") },
              { value: "error", label: tr("dashboard.logs.level.error", "Error") },
            ]}
          />
          <Input
            className="bp-input"
            allowClear
            value={logsQuery}
            onChange={(e) => setLogsQuery(e.target.value)}
            placeholder={tr("dashboard.logs.search", "Search source/message")}
            prefix={<SearchOutlined style={{ color: "#94a3b8" }} />}
          />
        </div>
        <Table<RuntimeLogItem>
          rowKey={(item) => `${item.timestamp}-${item.level}-${item.source}-${item.message}`}
          size="small"
          loading={logsModalLoading || logsModalFetching}
          dataSource={logsModalData?.items || []}
          columns={[
            {
              title: tr("dashboard.logs.col.time", "Time"),
              dataIndex: "timestamp",
              key: "timestamp",
              width: 210,
              className: "bp-table-mono",
              sorter: (a, b) => a.timestamp.localeCompare(b.timestamp),
              render: (value: string) => formatDateTime(value),
            },
            {
              title: tr("dashboard.logs.col.level", "Level"),
              dataIndex: "level",
              key: "level",
              width: 100,
              sorter: (a, b) => a.level.localeCompare(b.level),
              render: (value: string) => (
                <Tag color={value === "error" ? "error" : value === "warn" ? "warning" : "processing"}>
                  {value.toUpperCase()}
                </Tag>
              ),
            },
            {
              title: tr("dashboard.logs.col.source", "Source"),
              dataIndex: "source",
              key: "source",
              width: 120,
              className: "bp-table-mono",
            },
            {
              title: tr("dashboard.logs.col.message", "Message"),
              dataIndex: "message",
              key: "message",
            },
          ]}
          pagination={{ pageSize: 12, showSizeChanger: true }}
        />
      </Modal>
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

type LatencyTone =
  | "excellent"
  | "good"
  | "medium"
  | "slow"
  | "poor"
  | "error"
  | "warn"
  | "unknown";

function latencyTone(latencyMs?: number | null, testStatus?: string | null): LatencyTone {
  const status = (testStatus || "").toLowerCase();
  if (status === "error") {
    return "error";
  }
  if (latencyMs === null || latencyMs === undefined) {
    return status === "warn" ? "warn" : "unknown";
  }
  if (latencyMs <= 80) {
    return "excellent";
  }
  if (latencyMs <= 150) {
    return "good";
  }
  if (latencyMs <= 300) {
    return "medium";
  }
  if (latencyMs <= 600) {
    return "slow";
  }
  return "poor";
}

function formatLatency(latencyMs?: number | null): string {
  if (latencyMs === null || latencyMs === undefined) {
    return "-";
  }
  return `${latencyMs} ms`;
}

type TrafficSourceTone = "success" | "warning" | "danger" | "muted";

function getTrafficSourceMeta(
  source: string | null | undefined,
  tr: (key: string, fallback?: string, params?: Record<string, string | number | boolean | null | undefined>) => string
): {
  label: string;
  description: string;
  tone: TrafficSourceTone;
} {
  const normalized = (source || "").trim().toLowerCase();
  if (normalized === "singbox_clash_api") {
    return {
      label: tr("dashboard.traffic.source.proxy_only", "Proxy Only"),
      description: tr("dashboard.traffic.source.proxy_only_desc", "Only traffic forwarded by sing-box is counted."),
      tone: "success",
    };
  }
  if (normalized === "singbox_clash_api_disabled") {
    return {
      label: tr("dashboard.traffic.source.disabled", "Disabled"),
      description: tr("dashboard.traffic.source.disabled_desc", "Proxy traffic metrics are disabled by config."),
      tone: "warning",
    };
  }
  if (normalized === "singbox_clash_api_unavailable") {
    return {
      label: tr("dashboard.traffic.source.unavailable", "Unavailable"),
      description: tr("dashboard.traffic.source.unavailable_desc", "Cannot reach sing-box Clash API, metrics are temporarily unavailable."),
      tone: "danger",
    };
  }
  return {
    label: tr("dashboard.traffic.source.unknown", "Unknown"),
    description: tr("dashboard.traffic.source.unknown_desc", "Traffic source is not recognized."),
    tone: "muted",
  };
}
