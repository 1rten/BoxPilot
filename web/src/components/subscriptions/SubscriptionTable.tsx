import type { Subscription } from "../../api/types";
import { formatDateTime } from "../../utils/datetime";
import { Button, Progress, Table, Tag, Tooltip } from "antd";
import { DeleteOutlined, EditOutlined, LoadingOutlined, ReloadOutlined } from "@ant-design/icons";
import type { ColumnsType, TablePaginationConfig } from "antd/es/table";
import { useI18n } from "../../i18n/context";

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
  const { tr } = useI18n();
  if (!list.length) return null;

  const columns: ColumnsType<Subscription> = [
    {
      title: tr("subs.table.name", "Name"),
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
      title: tr("subs.table.status", "Status"),
      dataIndex: "status",
      key: "status",
      width: 90,
      render: (_value, record) => renderStatusTag(record, tr),
    },
    {
      title: tr("subs.table.auto", "Auto Update"),
      dataIndex: "auto_update_enabled",
      key: "auto_update_enabled",
      width: 130,
      render: (value: boolean, record) => (
        <Tag color={value ? "blue" : "default"}>
          {value ? `${record.refresh_interval_sec}s` : tr("subs.table.off", "Off")}
        </Tag>
      ),
    },
    {
      title: tr("subs.table.plan", "Plan"),
      key: "plan",
      width: 230,
      render: (_value, record) => renderPlanCell(record, tr),
    },
    {
      title: (
        <span
          style={{ cursor: onToggleSort ? "pointer" : "default" }}
          onClick={onToggleSort}
        >
          {tr("subs.table.updated", "Updated at")}{" "}
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
      title: tr("subs.table.last_error", "Last error"),
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
      title: tr("subs.table.actions", "Actions"),
      key: "actions",
      align: "right",
      width: 140,
      render: (_value, record) => {
        const refreshing = rowRefreshingId === record.id;
        return (
          <div className="bp-row-actions">
            <Tooltip title={tr("subs.table.action.edit", "Edit")}>
              <Button
                type="text"
                className="bp-row-action-btn"
                aria-label="Edit subscription"
                icon={<EditOutlined />}
                onClick={() => onEdit(record)}
              />
            </Tooltip>
            <Tooltip title={tr("subs.table.action.delete", "Delete")}>
              <Button
                type="text"
                danger
                className="bp-row-action-btn"
                aria-label="Delete subscription"
                icon={<DeleteOutlined />}
                onClick={() => onDelete(record)}
              />
            </Tooltip>
            <Tooltip title={refreshing ? tr("subs.table.action.refreshing", "Refreshing...") : tr("common.refresh", "Refresh")}>
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
      scroll={{ x: 1380 }}
      dataSource={list}
      columns={columns}
      pagination={pagination}
      loading={loading}
    />
  );
}

function renderPlanCell(
  s: Subscription,
  tr: (key: string, fallback?: string, params?: Record<string, string | number | boolean | null | undefined>) => string
): JSX.Element {
  const hasQuota = typeof s.total_bytes === "number" && s.total_bytes > 0 && typeof s.used_bytes === "number";
  const percent = hasQuota
    ? typeof s.usage_percent === "number"
      ? Math.max(0, Math.min(100, s.usage_percent))
      : Math.max(0, Math.min(100, (s.used_bytes || 0) * 100 / (s.total_bytes || 1)))
    : 0;
  const expireText = s.expire_at ? formatDateTime(s.expire_at) : "-";

  if (!hasQuota && !s.expire_at && !s.profile_web_page) {
    return <span style={{ color: "#64748B" }}>{tr("subs.plan.unavailable", "Unavailable")}</span>;
  }

  return (
    <div style={{ display: "grid", gap: 6 }}>
      {hasQuota ? (
        <>
          <div className="bp-table-mono" style={{ fontSize: 12 }}>
            {tr("subs.plan.used_total", "Used / Total")}: {formatBytes(s.used_bytes || 0)} / {formatBytes(s.total_bytes || 0)}
          </div>
          <Progress size="small" percent={percent} showInfo={false} />
        </>
      ) : null}
      {s.expire_at ? (
        <div style={{ color: "#64748B", fontSize: 12 }}>
          {tr("subs.plan.expire", "Expire")}: <span className="bp-table-mono">{expireText}</span>
        </div>
      ) : null}
      {s.profile_web_page ? (
        <a
          href={s.profile_web_page}
          target="_blank"
          rel="noreferrer"
          style={{ color: "#2563eb", fontSize: 12, fontWeight: 600 }}
        >
          {tr("subs.plan.portal", "Portal")}
        </a>
      ) : null}
    </div>
  );
}

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value < 0) {
    return "-";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let n = value;
  let idx = 0;
  while (n >= 1024 && idx < units.length - 1) {
    n /= 1024;
    idx++;
  }
  const digits = idx === 0 ? 0 : 1;
  return `${n.toFixed(digits)} ${units[idx]}`;
}

function truncate(s: string, max = 40): string {
  return s.length > max ? s.slice(0, max - 3) + "..." : s;
}

function renderStatusTag(
  s: Subscription,
  tr: (key: string, fallback?: string, params?: Record<string, string | number | boolean | null | undefined>) => string
): JSX.Element {
  const hasError = !!s.last_error;
  const paused = !s.enabled && !hasError;
  const label = hasError
    ? tr("subs.status.error", "Error")
    : paused
      ? tr("subs.status.paused", "Paused")
      : tr("subs.status.active", "Active");
  return <Tag color={hasError ? "error" : paused ? "warning" : "success"}>{label}</Tag>;
}
