import { useEffect, useMemo, useState } from "react";
import { useBatchForwarding, useNodes, useUpdateNode, useTestNodes } from "../hooks/useNodes";
import { useSubscriptions } from "../hooks/useSubscriptions";
import { ErrorState } from "../components/common/ErrorState";
import { EmptyState } from "../components/common/EmptyState";
import { formatDateTime } from "../utils/datetime";
import {
  EyeOutlined,
  LoadingOutlined,
  MoreOutlined,
  PoweroffOutlined,
  SearchOutlined,
  SwapOutlined,
  ThunderboltOutlined,
} from "@ant-design/icons";
import { Button, Card, Drawer, Dropdown, Input, Popconfirm, Table, Tag, Tooltip } from "antd";
import type { MenuProps } from "antd";
import type { ColumnsType, TableRowSelection } from "antd/es/table/interface";
import type { Node } from "../api/types";
import { useI18n } from "../i18n/context";

export default function Nodes() {
  const { tr } = useI18n();
  const { data: list, isLoading, error, refetch } = useNodes({});
  const { data: subscriptions } = useSubscriptions();
  const update = useUpdateNode();
  const testNodes = useTestNodes();
  const batchForwarding = useBatchForwarding();
  const [search, setSearch] = useState("");
  const [detailOpen, setDetailOpen] = useState(false);
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);
  const [testMode, setTestMode] = useState<"ping" | "http">("ping");
  const [rowTestingId, setRowTestingId] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const filtered = useMemo(() => {
    if (!list) return list;
    const q = search.trim().toLowerCase();
    if (!q) return list;
    return list.filter(
      (n) =>
        n.name.toLowerCase().includes(q) ||
        n.tag.toLowerCase().includes(q) ||
        n.type.toLowerCase().includes(q) ||
        (n.server || "").toLowerCase().includes(q)
    );
  }, [list, search]);

  const rowSelection: TableRowSelection<Node> = {
    selectedRowKeys,
    onChange: (keys) => setSelectedRowKeys(keys.map((k) => String(k))),
  };

  useEffect(() => {
    if (!list) return;
    const validKeys = new Set(list.map((item) => item.id));
    setSelectedRowKeys((prev) => prev.filter((key) => validKeys.has(key)));
  }, [list]);

  useEffect(() => {
    setPage(1);
  }, [search, list?.length]);

  const selectedCount = selectedRowKeys.length;
  const testModeLabel = testMode.toUpperCase();
  const boundSubName = useMemo(() => {
    if (!selectedNode || !subscriptions) return null;
    const found = subscriptions.find((s) => s.id === selectedNode.sub_id);
    return found?.name || null;
  }, [selectedNode, subscriptions]);
  const forwardingMenu: MenuProps = {
    items: [
      { key: "enable", label: tr("nodes.forwarding.enable", "Enable Forwarding") },
      { key: "disable", label: tr("nodes.forwarding.disable", "Disable Forwarding") },
    ],
    onClick: ({ key }) => {
      if (key === "enable") {
        void batchSetForwarding(true);
      }
      if (key === "disable") {
        void batchSetForwarding(false);
      }
    },
  };
  const testMenu: MenuProps = {
    items: [
      { key: "ping", label: "PING" },
      { key: "http", label: "HTTP" },
    ],
    onClick: ({ key }) => {
      const mode = key === "http" ? "http" : "ping";
      setTestMode(mode);
    },
  };

  const virtualEnabled = (filtered?.length ?? 0) > 200;

  return (
    <div className="bp-page">
      <div className="bp-page-header">
        <div>
          <h1 className="bp-page-title">{tr("nodes.title", "Nodes")}</h1>
          <p className="bp-page-subtitle">
            {tr("nodes.subtitle", "Select forwarding nodes and run connectivity tests.")}
          </p>
        </div>
        <div className="bp-page-actions">
          <Dropdown menu={testMenu} trigger={["click"]}>
            <Button className="bp-btn-fixed bp-btn-test-mode">
              {tr("nodes.test.mode", "Mode")}: {testModeLabel}
            </Button>
          </Dropdown>
          <Button
            className="bp-btn-fixed bp-btn-test-selected"
            disabled={selectedCount === 0}
            loading={testNodes.isPending}
            onClick={() => void runSelectedTest(testMode)}
          >
            {tr("nodes.test.selected", "Test Selected")}
          </Button>
          <Button className="bp-btn-fixed" onClick={() => refetch()} loading={isLoading}>
            {tr("common.refresh", "Refresh")}
          </Button>
        </div>
      </div>

      <Card className="bp-data-card">
        <div className="bp-toolbar-inline bp-nodes-toolbar">
          <Input
            className="bp-input bp-search-input bp-toolbar-search bp-nodes-search"
            prefix={<SearchOutlined style={{ color: "#94a3b8" }} />}
            allowClear
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={tr("nodes.search.placeholder", "Search by name or address")}
          />
          <div className="bp-toolbar-actions-fixed bp-nodes-toolbar-actions">
            <span className="bp-selection-pill bp-selection-pill-static">
              {tr("nodes.selected", "Selected {count}", { count: selectedCount })}
            </span>
            <Dropdown menu={forwardingMenu} disabled={selectedCount === 0} trigger={["click"]}>
              <Button
                className="bp-batch-forwarding-btn bp-btn-fixed"
                disabled={selectedCount === 0}
                loading={batchForwarding.isPending}
                icon={<MoreOutlined />}
              >
                {tr("nodes.forwarding.batch", "Batch Forwarding")}
              </Button>
            </Dropdown>
          </div>
        </div>

        {error && (
          <ErrorState
            message={tr("nodes.error.load", "Failed to load nodes: {message}", { message: (error as Error).message })}
            onRetry={() => refetch()}
          />
        )}

        {filtered && filtered.length > 0 ? (
          <Table<Node>
            rowKey="id"
            rowSelection={rowSelection}
            size="middle"
            dataSource={filtered}
            loading={isLoading}
            onRow={(record) => ({
              onClick: (event) => {
                const target = event.target as HTMLElement;
                if (target.closest(".ant-table-selection-column") || target.closest(".bp-row-actions")) {
                  return;
                }
                openDetails(record);
              },
              className: "bp-clickable-row",
            })}
            pagination={{
              current: page,
              pageSize,
              showSizeChanger: true,
              pageSizeOptions: [10, 20, 50, 100],
              onChange: (nextPage, nextPageSize) => {
                const resolvedPageSize = nextPageSize || pageSize;
                if (resolvedPageSize !== pageSize) {
                  setPageSize(resolvedPageSize);
                  setPage(1);
                  return;
                }
                setPage(nextPage);
              },
            }}
            columns={buildColumns({
              tr,
              onToggleForwarding: (row) =>
                update.mutate({ id: row.id, forwarding_enabled: !row.forwarding_enabled }),
              onToggleEnabled: (row) =>
                update.mutate({
                  id: row.id,
                  enabled: !row.enabled,
                }),
              onTest: (row) => runSingleNodeTest(row.id),
              onShowDetails: (row) => openDetails(row),
              updating: update.isPending,
              rowTestingId,
            })}
            tableLayout="fixed"
            virtual={virtualEnabled}
            scroll={virtualEnabled ? { y: 560 } : undefined}
          />
        ) : (
          !isLoading && (
            <EmptyState
              title={list && list.length > 0 ? tr("nodes.empty.search.title", "No results") : tr("nodes.empty.base.title", "No nodes yet")}
              description={
                list && list.length > 0
                  ? tr("nodes.empty.search.desc", "Try adjusting your search keywords.")
                  : tr("nodes.empty.base.desc", "Add a subscription and refresh to import nodes.")
              }
            />
          )
        )}
      </Card>

      <Drawer
        className="bp-drawer"
        width={520}
        getContainer={false}
        rootStyle={{ position: "fixed" }}
        onClose={() => setDetailOpen(false)}
        open={detailOpen}
        title={selectedNode ? selectedNode.name || selectedNode.tag : tr("nodes.drawer.title", "Node Details")}
      >
        {selectedNode && (
          <>
            <div className="bp-node-drawer-header">
              <Tag className="bp-node-status-pill" color={selectedNode.enabled ? "success" : "default"}>
                {selectedNode.enabled ? tr("nodes.online", "Online") : tr("nodes.offline", "Offline")}
              </Tag>
            </div>

            <div className="bp-node-drawer-section">
              <h3 className="bp-drawer-section-title">{tr("nodes.drawer.type_block", "Type")}</h3>
              <div className="bp-node-info-list">
                <div className="bp-node-info-row">
                  <span>{tr("nodes.col.type", "Type")}</span>
                  <strong>{selectedNode.type.toUpperCase()}</strong>
                </div>
                <div className="bp-node-info-row">
                  <span>{tr("nodes.ip", "IP")}</span>
                  <strong className="bp-table-mono">{selectedNode.server || "-"}</strong>
                </div>
                <div className="bp-node-info-row">
                  <span>{tr("settings.proxy.port", "Port")}</span>
                  <strong className="bp-table-mono">{selectedNode.server_port ?? "-"}</strong>
                </div>
                <div className="bp-node-info-row">
                  <span>{tr("nodes.created_at", "Created At")}</span>
                  <strong className="bp-table-mono">{formatDateTime(selectedNode.created_at)}</strong>
                </div>
                <div className="bp-node-info-row">
                  <span>{tr("nodes.last_seen", "Last Seen")}</span>
                  <strong className="bp-table-mono">
                    {selectedNode.last_test_at ? formatDateTime(selectedNode.last_test_at) : "-"}
                  </strong>
                </div>
              </div>
            </div>

            <div className="bp-node-drawer-section">
              <div className="bp-node-section-header">
                <h3 className="bp-drawer-section-title">{tr("nodes.drawer.ports", "Ports")}</h3>
              </div>
              <div className="bp-node-ports">
                <div className="bp-node-ports-head">
                  <span>{tr("settings.proxy.port", "Port")}</span>
                  <span>{tr("nodes.drawer.protocol", "Protocol")}</span>
                  <span>{tr("subs.table.status", "Status")}</span>
                </div>
                <div className="bp-node-ports-row">
                  <span>{selectedNode.server_port ?? "-"}</span>
                  <span>{selectedNode.type.toUpperCase()}</span>
                  <span>
                    <Tag
                      color={
                        selectedNode.last_test_status
                          ? statusColor(selectedNode.last_test_status)
                          : selectedNode.enabled
                          ? "success"
                          : "default"
                      }
                    >
                      {selectedNode.last_test_status
                        ? selectedNode.last_test_status.toUpperCase()
                        : selectedNode.enabled
                        ? tr("nodes.drawer.active", "ACTIVE")
                        : tr("nodes.drawer.inactive", "INACTIVE")}
                    </Tag>
                  </span>
                </div>
              </div>
              {selectedNode.last_test_error && (
                <p className="bp-text-danger" style={{ marginTop: 10 }}>
                  {selectedNode.last_test_error}
                </p>
              )}
            </div>

            <div className="bp-node-drawer-section">
              <h3 className="bp-drawer-section-title">{tr("nodes.drawer.bound_subs", "Bound Subscriptions")}</h3>
              <ul className="bp-node-bound-list">
                <li>{boundSubName || selectedNode.sub_id}</li>
              </ul>
            </div>

            <div className="bp-node-drawer-footer">
              <Button
                type="primary"
                className="bp-btn-fixed"
                loading={testNodes.isPending}
                onClick={() => testNodes.mutate({ node_ids: [selectedNode.id], mode: testMode })}
              >
                {tr("nodes.test.node", "Test Node")} ({testModeLabel})
              </Button>
              <Button
                className="bp-btn-fixed"
                danger={selectedNode.enabled}
                onClick={() => update.mutate({ id: selectedNode.id, enabled: !selectedNode.enabled })}
                disabled={update.isPending}
              >
                {selectedNode.enabled ? tr("nodes.disable", "Disable Node") : tr("nodes.enable", "Enable Node")}
              </Button>
            </div>
          </>
        )}
      </Drawer>
    </div>
  );

  function openDetails(row: Node) {
    setSelectedNode(row);
    setDetailOpen(true);
  }

  async function batchSetForwarding(enabled: boolean) {
    if (selectedRowKeys.length === 0) return;
    await batchForwarding.mutateAsync({
      node_ids: selectedRowKeys,
      forwarding_enabled: enabled,
    });
  }

  async function runSelectedTest(mode: "ping" | "http" = testMode) {
    if (selectedRowKeys.length === 0) return;
    setRowTestingId(null);
    await testNodes.mutateAsync({
      node_ids: selectedRowKeys,
      mode,
    });
  }

  async function runSingleNodeTest(nodeID: string) {
    setRowTestingId(nodeID);
    try {
      await testNodes.mutateAsync({ node_ids: [nodeID], mode: testMode });
    } finally {
      setRowTestingId(null);
    }
  }
}

function buildColumns({
  tr,
  updating,
  rowTestingId,
  onToggleForwarding,
  onToggleEnabled,
  onTest,
  onShowDetails,
}: {
  tr: (key: string, fallback?: string, params?: Record<string, string | number | boolean | null | undefined>) => string;
  updating: boolean;
  rowTestingId: string | null;
  onToggleForwarding: (row: Node) => void;
  onToggleEnabled: (row: Node) => void;
  onTest: (row: Node) => void;
  onShowDetails: (row: Node) => void;
}): ColumnsType<Node> {
  return [
    {
      title: tr("subs.table.name", "Name"),
      dataIndex: "name",
      key: "name",
      width: 210,
      ellipsis: true,
      render: (_value, record) => record.name || record.tag,
    },
    {
      title: tr("nodes.col.type", "Type"),
      dataIndex: "type",
      key: "type",
      width: 110,
      render: (value: string) => value.toUpperCase(),
    },
    {
      title: tr("nodes.col.forwarding", "Forwarding"),
      dataIndex: "forwarding_enabled",
      key: "forwarding_enabled",
      width: 130,
      render: (value: boolean) => <Tag color={value ? "blue" : "default"}>{value ? tr("nodes.status.enabled", "Enabled") : tr("nodes.status.disabled", "Disabled")}</Tag>,
    },
    {
      title: tr("nodes.col.node_status", "Node Status"),
      dataIndex: "enabled",
      key: "status",
      width: 130,
      render: (value: boolean) => <Tag color={value ? "success" : "default"}>{value ? tr("nodes.status.enabled", "Enabled") : tr("nodes.status.disabled", "Disabled")}</Tag>,
    },
    {
      title: tr("nodes.col.latency", "Latency"),
      dataIndex: "last_latency_ms",
      key: "latency",
      width: 120,
      render: (_value, record) => (
        <span
          className={`bp-latency-badge bp-latency-badge-${latencyTone(
            record.last_latency_ms,
            record.last_test_status
          )}`}
        >
          {formatLatency(record.last_latency_ms)}
        </span>
      ),
    },
    {
      title: tr("nodes.col.last_test", "Last Test"),
      dataIndex: "last_test_at",
      key: "last_test_at",
      width: 190,
      render: (value: string | null | undefined) =>
        value ? <span className="bp-table-mono">{formatDateTime(value)}</span> : "-",
    },
    {
      title: tr("subs.table.actions", "Actions"),
      key: "actions",
      align: "right",
      width: 180,
      render: (_value, record) => (
        <div
          className="bp-row-actions"
          onClick={(event) => {
            event.stopPropagation();
          }}
        >
          <Tooltip title={tr("nodes.action.details", "Details")}>
            <Button
              type="text"
              className="bp-row-action-btn"
              aria-label={tr("nodes.action.details", "Details")}
              icon={<EyeOutlined />}
              onClick={() => onShowDetails(record)}
            />
          </Tooltip>
          <Tooltip title={tr("nodes.action.test", "Test")}>
            <Button
              type="text"
              className="bp-row-action-btn"
              aria-label={tr("nodes.action.test", "Test")}
              icon={rowTestingId === record.id ? <LoadingOutlined spin /> : <ThunderboltOutlined />}
              onClick={() => onTest(record)}
            />
          </Tooltip>
          {record.enabled ? (
            <Popconfirm
              title={tr("nodes.disable.confirm", "Disable this node?")}
              description={tr("nodes.disable.desc", "Disabled nodes will be excluded from forwarding and tests.")}
              okText={tr("nodes.disable.short", "Disable")}
              cancelText={tr("common.cancel", "Cancel")}
              onConfirm={() => onToggleEnabled(record)}
            >
              <Tooltip title={tr("nodes.disable.short", "Disable")}>
                <Button
                  type="text"
                  className="bp-row-action-btn"
                  aria-label={tr("nodes.disable.short", "Disable")}
                  icon={<PoweroffOutlined />}
                  disabled={updating}
                />
              </Tooltip>
            </Popconfirm>
          ) : (
            <Tooltip title={tr("nodes.enable.short", "Enable")}>
              <Button
                type="text"
                className="bp-row-action-btn"
                aria-label={tr("nodes.enable.short", "Enable")}
                icon={<PoweroffOutlined />}
                onClick={() => onToggleEnabled(record)}
                disabled={updating}
              />
            </Tooltip>
          )}
          <Tooltip title={record.forwarding_enabled ? tr("nodes.forwarding.disable", "Disable forwarding") : tr("nodes.forwarding.enable", "Enable forwarding")}>
            <Button
              type="text"
              className="bp-row-action-btn"
              aria-label={record.forwarding_enabled ? tr("nodes.forwarding.disable", "Disable forwarding") : tr("nodes.forwarding.enable", "Enable forwarding")}
              icon={<SwapOutlined />}
              onClick={() => onToggleForwarding(record)}
              disabled={updating}
            />
          </Tooltip>
        </div>
      ),
    },
  ];
}

function statusColor(status: string): string {
  switch (status.toLowerCase()) {
    case "ok":
      return "success";
    case "warn":
      return "warning";
    case "error":
      return "error";
    default:
      return "default";
  }
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
