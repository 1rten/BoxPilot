import type { Subscription } from "../../api/types";
import { formatDateTime } from "../../utils/datetime";
import { Button, Table, Tag } from "antd";
import type { ColumnsType, TablePaginationConfig } from "antd/es/table";

export interface SubscriptionTableProps {
  list: Subscription[];
  loading: boolean;
  rowRefreshingId: string | null;
  onEdit: (row: Subscription) => void;
  onDelete: (row: Subscription) => void;
  onRefreshRow: (row: Subscription) => void;
  sortOrder?: "asc" | "desc";
  onToggleSort?: () => void;
  pagination?: TablePaginationConfig | false;
}

export function SubscriptionTable({
  list,
  loading,
  rowRefreshingId,
  onEdit,
  onDelete,
  onRefreshRow,
  sortOrder,
  onToggleSort,
  pagination,
}: SubscriptionTableProps) {
  if (!list.length) return null;

  const columns: ColumnsType<Subscription> = [
    {
      title: "Name",
      dataIndex: "name",
      key: "name",
      render: (_value, record) => record.name || record.url,
    },
    {
      title: "URL",
      dataIndex: "url",
      key: "url",
      render: (value: string) => (
        <span className="bp-table-mono" title={value}>
          {truncate(value)}
        </span>
      ),
    },
    {
      title: "Status",
      dataIndex: "status",
      key: "status",
      render: (_value, record) => renderStatusTag(record),
    },
    {
      title: (
        <span
          style={{ cursor: onToggleSort ? "pointer" : "default" }}
          onClick={onToggleSort}
        >
          Updated at{" "}
          {sortOrder === "asc" ? "↑" : sortOrder === "desc" ? "↓" : ""}
        </span>
      ),
      dataIndex: "updated_at",
      key: "updated_at",
      render: (value: string) => (
        <span className="bp-table-mono">{formatDateTime(value)}</span>
      ),
    },
    {
      title: "Last error",
      dataIndex: "last_error",
      key: "last_error",
      render: (value: string | null | undefined) => (
        <span style={{ color: value ? "#DC2626" : "#64748B" }}>
          {value || "-"}
        </span>
      ),
    },
    {
      title: "Actions",
      key: "actions",
      align: "right",
      render: (_value, record) => {
        const refreshing = rowRefreshingId === record.id && loading;
        return (
          <div className="bp-row-actions">
            <Button type="link" onClick={() => onEdit(record)}>
              Edit
            </Button>
            <Button
              type="link"
              danger
              onClick={() => onDelete(record)}
              style={{ paddingLeft: 0 }}
            >
              Delete
            </Button>
            <Button
              type="link"
              onClick={() => onRefreshRow(record)}
              disabled={refreshing}
            >
              {refreshing ? "Refreshing..." : "Refresh"}
            </Button>
          </div>
        );
      },
    },
  ];

  return (
    <Table<Subscription>
      rowKey="id"
      dataSource={list}
      columns={columns}
      pagination={pagination}
      loading={loading}
    />
  );
}

function truncate(s: string, max = 40): string {
  return s.length > max ? s.slice(0, max - 3) + "..." : s;
}

function renderStatusTag(s: Subscription): JSX.Element {
  const hasError = !!s.last_error;
  const paused = !s.enabled && !hasError;
  const label = hasError ? "Error" : paused ? "Paused" : "Active";
  return <Tag color={hasError ? "error" : paused ? "warning" : "success"}>{label}</Tag>;
}
