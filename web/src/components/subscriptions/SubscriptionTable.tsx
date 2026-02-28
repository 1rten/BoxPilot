import type { Subscription } from "../../api/types";
import { formatDateTime } from "../../utils/datetime";
import { Button, Table, Tag, Tooltip } from "antd";
import { DeleteOutlined, EditOutlined, LoadingOutlined, ReloadOutlined } from "@ant-design/icons";
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
      width: 140,
      ellipsis: true,
      render: (_value, record) => record.name || record.url,
    },
    {
      title: "URL",
      dataIndex: "url",
      key: "url",
      width: 260,
      ellipsis: true,
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
      width: 90,
      render: (_value, record) => renderStatusTag(record),
    },
    {
      title: "Auto Update",
      dataIndex: "auto_update_enabled",
      key: "auto_update_enabled",
      width: 130,
      render: (value: boolean, record) => (
        <Tag color={value ? "blue" : "default"}>
          {value ? `${record.refresh_interval_sec}s` : "Off"}
        </Tag>
      ),
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
      width: 160,
      render: (value: string) => (
        <span className="bp-table-mono">{formatDateTime(value)}</span>
      ),
    },
    {
      title: "Last error",
      dataIndex: "last_error",
      key: "last_error",
      width: 180,
      ellipsis: true,
      render: (value: string | null | undefined) => (
        <span style={{ color: value ? "#DC2626" : "#64748B" }} title={value ?? undefined}>
          {value || "-"}
        </span>
      ),
    },
    {
      title: "Actions",
      key: "actions",
      align: "right",
      width: 140,
      render: (_value, record) => {
        const refreshing = rowRefreshingId === record.id;
        return (
          <div className="bp-row-actions">
            <Tooltip title="Edit">
              <Button
                type="text"
                className="bp-row-action-btn"
                aria-label="Edit subscription"
                icon={<EditOutlined />}
                onClick={() => onEdit(record)}
              />
            </Tooltip>
            <Tooltip title="Delete">
              <Button
                type="text"
                danger
                className="bp-row-action-btn"
                aria-label="Delete subscription"
                icon={<DeleteOutlined />}
                onClick={() => onDelete(record)}
              />
            </Tooltip>
            <Tooltip title={refreshing ? "Refreshing..." : "Refresh"}>
              <Button
                type="text"
                className="bp-row-action-btn"
                aria-label="Refresh subscription"
                icon={refreshing ? <LoadingOutlined spin /> : <ReloadOutlined />}
                onClick={() => onRefreshRow(record)}
                disabled={refreshing}
              />
            </Tooltip>
          </div>
        );
      },
    },
  ];

  return (
    <Table<Subscription>
      rowKey="id"
      scroll={{ x: 1160 }}
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
