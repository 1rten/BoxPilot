import { useEffect, useMemo, useState } from "react";
import { useBatchForwarding, useNodes, useUpdateNode, useTestNodes } from "../hooks/useNodes";
import { useSubscriptions } from "../hooks/useSubscriptions";
import { ErrorState } from "../components/common/ErrorState";
import { EmptyState } from "../components/common/EmptyState";
import { formatDateTime } from "../utils/datetime";
import { EyeOutlined, MoreOutlined, PoweroffOutlined, SwapOutlined, ThunderboltOutlined, SearchOutlined } from "@ant-design/icons";
import { Button, Card, Drawer, Dropdown, Input, Popconfirm, Table, Tag, Tooltip } from "antd";
import type { MenuProps } from "antd";
import type { ColumnsType, TableRowSelection } from "antd/es/table/interface";
import type { Node } from "../api/types";

export default function Nodes() {
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

  const selectedCount = selectedRowKeys.length;
  const testModeLabel = testMode.toUpperCase();
  const boundSubName = useMemo(() => {
    if (!selectedNode || !subscriptions) return null;
    const found = subscriptions.find((s) => s.id === selectedNode.sub_id);
    return found?.name || null;
  }, [selectedNode, subscriptions]);
  const forwardingMenu: MenuProps = {
    items: [
      { key: "enable", label: "Enable Forwarding" },
      { key: "disable", label: "Disable Forwarding" },
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
      { key: "ping", label: "Test Selected via PING" },
      { key: "http", label: "Test Selected via HTTP" },
    ],
    onClick: ({ key }) => {
      const mode = key === "http" ? "http" : "ping";
      setTestMode(mode);
      void runSelectedTest(mode);
    },
  };

  return (
    <div className="bp-page">
      <div className="bp-page-header">
        <div>
          <h1 className="bp-page-title">Nodes</h1>
          <p className="bp-page-subtitle">
            Select forwarding nodes and run connectivity tests.
          </p>
        </div>
        <div className="bp-page-actions">
          <Dropdown
            menu={testMenu}
            disabled={selectedCount === 0}
            trigger={["click"]}
          >
            <Button
              disabled={selectedCount === 0}
              loading={testNodes.isPending}
            >
              Test Selected ({testModeLabel})
            </Button>
          </Dropdown>
          <Button onClick={() => refetch()} loading={isLoading}>
            Refresh
          </Button>
        </div>
      </div>

      <Card className="bp-data-card">
        <div className="bp-toolbar-inline bp-nodes-toolbar">
          <Input
            className="bp-input bp-search-input bp-nodes-search"
            prefix={<SearchOutlined style={{ color: "#94a3b8" }} />}
            allowClear
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search by name or address"
          />
          <div className="bp-page-actions bp-nodes-toolbar-actions">
            <span className="bp-selection-pill bp-selection-pill-static">Selected {selectedCount}</span>
            <Dropdown
              menu={forwardingMenu}
              disabled={selectedCount === 0}
              trigger={["click"]}
            >
              <Button
                className="bp-batch-forwarding-btn"
                disabled={selectedCount === 0}
                loading={batchForwarding.isPending}
                icon={<MoreOutlined />}
              >
                Batch Forwarding
              </Button>
            </Dropdown>
          </div>
        </div>

        {error && (
          <ErrorState
            message={`Failed to load nodes: ${(error as Error).message}`}
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
              onClick: () => openDetails(record),
              className: "bp-clickable-row",
            })}
            pagination={{
              pageSize: 10,
              showSizeChanger: true,
              showTotal: (total, range) =>
                `${range[0]}-${range[1]} of ${total} nodes`,
            }}
            columns={buildColumns({
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
          />
        ) : (
          !isLoading && (
            <EmptyState
              title={list && list.length > 0 ? "No results" : "No nodes yet"}
              description={
                list && list.length > 0
                  ? "Try adjusting your search keywords."
                  : "Add a subscription and refresh to import nodes."
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
        title={
          selectedNode ? (
            <div className="bp-drawer-title">
              <span className="bp-drawer-name">
                {selectedNode.name || selectedNode.tag}
              </span>
            </div>
          ) : (
            "Node Details"
          )
        }
      >
        {selectedNode && (
          <>
            <div className="bp-drawer-section">
              <Tag color={selectedNode.enabled ? "success" : "default"}>
                {selectedNode.enabled ? "Online" : "Offline"}
              </Tag>
              <div className="bp-node-detail-list">
                <div className="bp-node-detail-row"><span>Type</span><strong>{selectedNode.type.toUpperCase()}</strong></div>
                <div className="bp-node-detail-row"><span>IP</span><strong>{selectedNode.server || "-"}</strong></div>
                <div className="bp-node-detail-row"><span>Port</span><strong>{selectedNode.server_port ?? "-"}</strong></div>
                <div className="bp-node-detail-row"><span>Created At</span><strong className="bp-mono">{formatDateTime(selectedNode.created_at)}</strong></div>
                <div className="bp-node-detail-row">
                  <span>Last Seen</span>
                  <strong className="bp-mono">{selectedNode.last_test_at ? formatDateTime(selectedNode.last_test_at) : "-"}</strong>
                </div>
              </div>
            </div>
            <div className="bp-drawer-section">
              <h3 className="bp-card-title">Ports</h3>
              <div className="bp-node-ports">
                <div className="bp-node-ports-head">
                  <span>Port</span>
                  <span>Protocol</span>
                  <span>Status</span>
                </div>
                <div className="bp-node-ports-row">
                  <span>{selectedNode.server_port ?? "-"}</span>
                  <span>{selectedNode.type.toUpperCase()}</span>
                  <span>
                    <Tag color={selectedNode.last_test_status ? statusColor(selectedNode.last_test_status) : selectedNode.enabled ? "success" : "default"}>
                      {selectedNode.last_test_status ? selectedNode.last_test_status.toUpperCase() : selectedNode.enabled ? "ACTIVE" : "INACTIVE"}
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
            <div className="bp-drawer-section">
              <h3 className="bp-card-title">Bound Subscriptions</h3>
              <ul className="bp-node-bound-list">
                <li>{boundSubName || selectedNode.sub_id}</li>
              </ul>
            </div>
            <div className="bp-node-drawer-footer">
              <Button
                type="primary"
                loading={testNodes.isPending}
                onClick={() => testNodes.mutate({ node_ids: [selectedNode.id], mode: testMode })}
              >
                Test Node ({testModeLabel})
              </Button>
              <Button
                danger={selectedNode.enabled}
                onClick={() => update.mutate({ id: selectedNode.id, enabled: !selectedNode.enabled })}
                disabled={update.isPending}
              >
                {selectedNode.enabled ? "Disable Node" : "Enable Node"}
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
  updating,
  rowTestingId,
  onToggleForwarding,
  onToggleEnabled,
  onTest,
  onShowDetails,
}: {
  updating: boolean;
  rowTestingId: string | null;
  onToggleForwarding: (row: Node) => void;
  onToggleEnabled: (row: Node) => void;
  onTest: (row: Node) => void;
  onShowDetails: (row: Node) => void;
}): ColumnsType<Node> {
  return [
    {
      title: "Name",
      dataIndex: "name",
      key: "name",
      width: 180,
      render: (_value, record) => record.name || record.tag,
    },
    { title: "Type", dataIndex: "type", key: "type", width: 110 },
    {
      title: "Forwarding",
      dataIndex: "forwarding_enabled",
      key: "forwarding_enabled",
      width: 130,
      render: (value: boolean) => (
        <Tag color={value ? "blue" : "default"}>{value ? "Enabled" : "Disabled"}</Tag>
      ),
    },
    {
      title: "Node Status",
      dataIndex: "enabled",
      key: "status",
      width: 130,
      render: (value: boolean) => (
        <Tag color={value ? "success" : "default"}>{value ? "Enabled" : "Disabled"}</Tag>
      ),
    },
    {
      title: "Latency",
      dataIndex: "last_latency_ms",
      key: "latency",
      width: 120,
      render: (_value, record) =>
        record.last_latency_ms !== null && record.last_latency_ms !== undefined
          ? `${record.last_latency_ms} ms`
          : "-",
    },
    {
      title: "Last Test",
      dataIndex: "last_test_at",
      key: "last_test_at",
      width: 190,
      render: (value: string | null | undefined) =>
        value ? <span className="bp-table-mono">{formatDateTime(value)}</span> : "-",
    },
    {
      title: "Actions",
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
          <Tooltip title="Details">
            <Button
              type="text"
              icon={<EyeOutlined />}
              onClick={() => onShowDetails(record)}
            />
          </Tooltip>
          <Tooltip title="Test">
            <Button
              type="text"
              icon={<ThunderboltOutlined />}
              loading={rowTestingId === record.id}
              onClick={() => onTest(record)}
            />
          </Tooltip>
          {record.enabled ? (
            <Popconfirm
              title="Disable this node?"
              description="Disabled nodes will be excluded from forwarding and tests."
              okText="Disable"
              cancelText="Cancel"
              onConfirm={() => onToggleEnabled(record)}
            >
              <Tooltip title="Disable">
                <Button type="text" icon={<PoweroffOutlined />} disabled={updating} />
              </Tooltip>
            </Popconfirm>
          ) : (
            <Tooltip title="Enable">
              <Button
                type="text"
                icon={<PoweroffOutlined />}
                onClick={() => onToggleEnabled(record)}
                disabled={updating}
              />
            </Tooltip>
          )}
          <Tooltip title={record.forwarding_enabled ? "Disable forwarding" : "Enable forwarding"}>
            <Button
              type="text"
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
