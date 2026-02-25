import { useMemo, useState } from "react";
import { useNodes, useUpdateNode, useTestNodes } from "../hooks/useNodes";
import { ErrorState } from "../components/common/ErrorState";
import { EmptyState } from "../components/common/EmptyState";
import { formatDateTime } from "../utils/datetime";
import { Button, Card, Drawer, Input, Select, Switch, Table, Tag } from "antd";
import type { ColumnsType, TableRowSelection } from "antd/es/table/interface";
import type { Node } from "../api/types";

export default function Nodes() {
  const { data: list, isLoading, error, refetch } = useNodes({});
  const update = useUpdateNode();
  const testNodes = useTestNodes();
  const [search, setSearch] = useState("");
  const [detailOpen, setDetailOpen] = useState(false);
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);
  const [testMode, setTestMode] = useState<"ping" | "http">("ping");

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
          <Select
            value={testMode}
            onChange={(value: "ping" | "http") => setTestMode(value)}
            options={[
              { value: "ping", label: "PING" },
              { value: "http", label: "HTTP" },
            ]}
            style={{ minWidth: 108 }}
          />
          <Button
            onClick={() =>
              testNodes.mutate({
                node_ids: selectedRowKeys.map((k) => String(k)),
                mode: testMode,
              })
            }
            disabled={selectedRowKeys.length === 0}
            loading={testNodes.isPending}
          >
            Test Selected
          </Button>
          <Button onClick={() => refetch()} loading={isLoading}>
            Refresh
          </Button>
        </div>
      </div>

      <Card className="bp-data-card">
        <div className="bp-card-toolbar">
          <Input
            className="bp-input bp-search-input"
            allowClear
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search nodes by name/tag/type/server"
          />
          <div className="bp-card-toolbar-meta">
            {filtered && (
              <span>
                Showing {filtered.length} of {list?.length ?? 0} nodes
              </span>
            )}
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
              onToggleForwarding: (row, checked) =>
                update.mutate({ id: row.id, forwarding_enabled: checked }),
              onToggleEnabled: (row) =>
                update.mutate({
                  id: row.id,
                  enabled: !row.enabled,
                }),
              onTest: (row) =>
                testNodes.mutate({ node_ids: [row.id], mode: testMode }),
              onShowDetails: (row) => openDetails(row),
              updating: update.isPending,
              testing: testNodes.isPending,
            })}
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
        onClose={() => setDetailOpen(false)}
        open={detailOpen}
        title={
          selectedNode ? (
            <div className="bp-drawer-title">
              <div>
                <span className="bp-drawer-name">
                  {selectedNode.name || selectedNode.tag}
                </span>
                <Tag
                  className="bp-drawer-status"
                  color={selectedNode.enabled ? "success" : "error"}
                >
                  {selectedNode.enabled ? "Online" : "Offline"}
                </Tag>
              </div>
              <span className="bp-muted">Node Details</span>
            </div>
          ) : (
            "Node Details"
          )
        }
      >
        {selectedNode && (
          <>
            <div className="bp-drawer-section">
              <div className="bp-drawer-kv">
                <div>
                  <p className="bp-kv-label">Type</p>
                  <p className="bp-kv-value">{selectedNode.type}</p>
                </div>
                <div>
                  <p className="bp-kv-label">Tag</p>
                  <p className="bp-kv-value bp-mono">{selectedNode.tag}</p>
                </div>
                <div>
                  <p className="bp-kv-label">Server</p>
                  <p className="bp-kv-value bp-mono">
                    {selectedNode.server || "-"}:{selectedNode.server_port || "-"}
                  </p>
                </div>
                <div>
                  <p className="bp-kv-label">Network</p>
                  <p className="bp-kv-value">{selectedNode.network || "-"}</p>
                </div>
                <div>
                  <p className="bp-kv-label">TLS</p>
                  <p className="bp-kv-value">{selectedNode.tls_enabled ? "Enabled" : "Disabled"}</p>
                </div>
                <div>
                  <p className="bp-kv-label">Created</p>
                  <p className="bp-kv-value bp-mono">{formatDateTime(selectedNode.created_at)}</p>
                </div>
              </div>
            </div>
            <div className="bp-drawer-section">
              <h3 className="bp-card-title">Health</h3>
              <div className="bp-drawer-kv">
                <div>
                  <p className="bp-kv-label">Last Status</p>
                  <p className="bp-kv-value">{selectedNode.last_test_status || "-"}</p>
                </div>
                <div>
                  <p className="bp-kv-label">Last Latency</p>
                  <p className="bp-kv-value">
                    {selectedNode.last_latency_ms !== null && selectedNode.last_latency_ms !== undefined
                      ? `${selectedNode.last_latency_ms} ms`
                      : "-"}
                  </p>
                </div>
                <div>
                  <p className="bp-kv-label">Last Test At</p>
                  <p className="bp-kv-value bp-mono">
                    {selectedNode.last_test_at ? formatDateTime(selectedNode.last_test_at) : "-"}
                  </p>
                </div>
              </div>
              {selectedNode.last_test_error && (
                <p className="bp-text-danger" style={{ marginTop: 12 }}>
                  {selectedNode.last_test_error}
                </p>
              )}
              <div className="bp-page-actions" style={{ marginTop: 12 }}>
                <Button
                  loading={testNodes.isPending}
                  onClick={() => testNodes.mutate({ node_ids: [selectedNode.id], mode: testMode })}
                >
                  Test Node ({testMode.toUpperCase()})
                </Button>
              </div>
            </div>
            <div className="bp-drawer-section">
              <h3 className="bp-card-title">Forwarding</h3>
              <p className="bp-muted">
                Forwarding configuration is global in Settings. This node only controls whether it participates in forwarding.
              </p>
              <div className="bp-inline-control" style={{ marginTop: 10 }}>
                <Switch
                  checked={selectedNode.forwarding_enabled}
                  onChange={(checked) => {
                    update.mutate(
                      { id: selectedNode.id, forwarding_enabled: checked },
                      {
                        onSuccess: (updated) => {
                          setSelectedNode(updated);
                        },
                      }
                    );
                  }}
                />
                <span>{selectedNode.forwarding_enabled ? "Forwarding enabled" : "Forwarding disabled"}</span>
              </div>
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
}

function buildColumns({
  updating,
  testing,
  onToggleForwarding,
  onToggleEnabled,
  onTest,
  onShowDetails,
}: {
  updating: boolean;
  testing: boolean;
  onToggleForwarding: (row: Node, checked: boolean) => void;
  onToggleEnabled: (row: Node) => void;
  onTest: (row: Node) => void;
  onShowDetails: (row: Node) => void;
}): ColumnsType<Node> {
  return [
    {
      title: "Name",
      dataIndex: "name",
      key: "name",
      render: (_value, record) => record.name || record.tag,
    },
    { title: "Type", dataIndex: "type", key: "type" },
    {
      title: "Forwarding",
      dataIndex: "forwarding_enabled",
      key: "forwarding_enabled",
      render: (value: boolean, record) => (
        <div
          onClick={(event) => {
            event.stopPropagation();
          }}
        >
          <Switch
            size="small"
            checked={value}
            disabled={updating}
            onChange={(checked) => onToggleForwarding(record, checked)}
          />
        </div>
      ),
    },
    {
      title: "Node Status",
      dataIndex: "enabled",
      key: "status",
      render: (value: boolean) => (
        <Tag color={value ? "success" : "default"}>{value ? "Enabled" : "Disabled"}</Tag>
      ),
    },
    {
      title: "Latency",
      dataIndex: "last_latency_ms",
      key: "latency",
      render: (_value, record) =>
        record.last_latency_ms !== null && record.last_latency_ms !== undefined
          ? `${record.last_latency_ms} ms`
          : "-",
    },
    {
      title: "Last Test",
      dataIndex: "last_test_at",
      key: "last_test_at",
      render: (value: string | null | undefined) =>
        value ? <span className="bp-table-mono">{formatDateTime(value)}</span> : "-",
    },
    {
      title: "Actions",
      key: "actions",
      align: "right",
      render: (_value, record) => (
        <div
          className="bp-row-actions"
          onClick={(event) => {
            event.stopPropagation();
          }}
        >
          <Button type="link" onClick={() => onShowDetails(record)}>
            Details
          </Button>
          <Button
            type="link"
            loading={testing}
            onClick={() => onTest(record)}
          >
            Test
          </Button>
          <Button
            type="link"
            onClick={() => onToggleEnabled(record)}
            disabled={updating}
          >
            {record.enabled ? "Disable" : "Enable"}
          </Button>
        </div>
      ),
    },
  ];
}
