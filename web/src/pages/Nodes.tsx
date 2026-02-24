import { useMemo, useState } from "react";
import { useNodes, useUpdateNode } from "../hooks/useNodes";
import { ErrorState } from "../components/common/ErrorState";
import { EmptyState } from "../components/common/EmptyState";
import { formatDateTime } from "../utils/datetime";
import { Button, Card, Input, Table, Tag } from "antd";
import type { ColumnsType } from "antd/es/table";
import type { Node } from "../api/types";

export default function Nodes() {
  const { data: list, isLoading, error, refetch } = useNodes({});
  const update = useUpdateNode();
  const [search, setSearch] = useState("");

  const filtered = useMemo(() => {
    if (!list) return list;
    const q = search.trim().toLowerCase();
    if (!q) return list;
    return list.filter(
      (n) =>
        n.name.toLowerCase().includes(q) ||
        n.tag.toLowerCase().includes(q) ||
        n.type.toLowerCase().includes(q)
    );
  }, [list, search]);

  return (
    <div>
      <h1 className="bp-page-title">Nodes</h1>

      <Card>
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            marginBottom: 16,
            gap: 12,
          }}
        >
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search nodes"
          />
          <div style={{ display: "flex", gap: 8 }}>
            <Button type="primary">Add Node</Button>
            <Button onClick={() => refetch()} loading={isLoading}>
              Refresh
            </Button>
          </div>
        </div>

        {isLoading && !list && (
          <div className="bp-card">
            <p style={{ color: "#64748B", fontSize: 14 }}>Loading nodes...</p>
          </div>
        )}
        {error && (
          <ErrorState
            message={`Failed to load nodes: ${(error as Error).message}`}
            onRetry={() => {
              refetch();
            }}
          />
        )}

        {filtered && filtered.length > 0 ? (
          <Table<Node>
            rowKey="id"
            size="middle"
            dataSource={filtered}
            loading={isLoading}
            pagination={{
              pageSize: 10,
              showSizeChanger: true,
              showTotal: (total, range) =>
                `${range[0]}-${range[1]} of ${total} nodes`,
            }}
            columns={buildColumns(update.isPending, (row) =>
              update.mutate({
                id: row.id,
                enabled: !row.enabled,
              })
            )}
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
    </div>
  );
}

function buildColumns(
  updating: boolean,
  onToggleEnabled: (row: Node) => void
): ColumnsType<Node> {
  return [
    {
      title: "Name",
      dataIndex: "name",
      key: "name",
      render: (_value, record) => record.name || record.tag,
    },
    { title: "Type", dataIndex: "type", key: "type" },
    {
      title: "Status",
      dataIndex: "enabled",
      key: "status",
      render: (value: boolean) => (
        <Tag color={value ? "success" : "error"}>
          {value ? "Online" : "Offline"}
        </Tag>
      ),
    },
    {
      title: "Created at",
      dataIndex: "created_at",
      key: "created_at",
      render: (value: string) => formatDateTime(value),
    },
    {
      title: "Actions",
      key: "actions",
      align: "right",
      render: (_value, record) => (
        <Button
          type="link"
          onClick={() => onToggleEnabled(record)}
          disabled={updating}
        >
          {updating ? "Updating..." : record.enabled ? "Disable" : "Enable"}
        </Button>
      ),
    },
  ];
}
