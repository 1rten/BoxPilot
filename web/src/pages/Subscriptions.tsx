import {
  useSubscriptions,
  useCreateSubscription,
  useUpdateSubscription,
  useDeleteSubscription,
  useRefreshSubscription,
} from "../hooks/useSubscriptions";
import { useEffect, useMemo, useState } from "react";
import type { Subscription } from "../api/types";
import { ErrorState } from "../components/common/ErrorState";
import { EmptyState } from "../components/common/EmptyState";
import { SubscriptionTable } from "../components/subscriptions/SubscriptionTable";
import {
  SubscriptionModal,
  type SubscriptionModalMode,
} from "../components/subscriptions/SubscriptionModal";
import { SearchOutlined } from "@ant-design/icons";
import { Button, Card, Input, Modal, Skeleton, Switch } from "antd";
import { useI18n } from "../i18n/context";

export default function Subscriptions() {
  const { tr } = useI18n();
  const {
    data: list,
    isLoading,
    error,
    refetch,
    isFetching,
  } = useSubscriptions();
  const create = useCreateSubscription();
  const update = useUpdateSubscription();
  const del = useDeleteSubscription();
  const refresh = useRefreshSubscription();

  const [modalOpen, setModalOpen] = useState(false);
  const [modalMode, setModalMode] = useState<SubscriptionModalMode>("edit");
  const [editingSub, setEditingSub] = useState<Subscription | null>(null);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [rowRefreshingId, setRowRefreshingId] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  useEffect(() => {
    if (!autoRefresh) return;
    const id = window.setInterval(() => {
      refetch();
    }, 30000);
    return () => window.clearInterval(id);
  }, [autoRefresh, refetch]);

  useEffect(() => {
    setPage(1);
  }, [search, sortOrder, list?.length]);

  const filteredList = useMemo(() => {
    if (!list) return list;
    const q = search.trim().toLowerCase();
    const base = !q
      ? list
      : list.filter(
      (s) =>
        s.name.toLowerCase().includes(q) ||
        s.url.toLowerCase().includes(q)
        );
    return [...base].sort((a, b) => {
      const va = a.updated_at;
      const vb = b.updated_at;
      if (va === vb) return 0;
      const cmp = va > vb ? 1 : -1;
      return sortOrder === "asc" ? cmp : -cmp;
    });
  }, [list, search, sortOrder]);

  const handleCreate = () => {
    setModalMode("create");
    setEditingSub(null);
    setModalOpen(true);
  };

  return (
    <div className="bp-page">
      <div className="bp-page-header">
        <div>
          <h1 className="bp-page-title">{tr("subs.title", "Subscriptions")}</h1>
          <p className="bp-page-subtitle">
            {tr("subs.subtitle", "Manage source feeds, refresh cadence, and sync health.")}
          </p>
        </div>
        <div className="bp-page-actions">
          <Button type="primary" onClick={handleCreate} loading={create.isPending}>
            {tr("subs.new", "New Subscription")}
          </Button>
          <Button onClick={() => refetch()} loading={isFetching}>
            {tr("common.refresh", "Refresh")}
          </Button>
        </div>
      </div>

      <Card className="bp-data-card">
        <div className="bp-toolbar-inline bp-list-toolbar bp-subscriptions-toolbar">
          <Input
            className="bp-input bp-search-input bp-toolbar-search bp-list-toolbar-search"
            prefix={<SearchOutlined style={{ color: "#94a3b8" }} />}
            allowClear
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={tr("subs.search.placeholder", "Search by name or URL")}
          />
          <div className="bp-toolbar-actions-fixed bp-list-toolbar-actions bp-subscriptions-toolbar-actions">
            <span className="bp-inline-control bp-subscriptions-autopoll">
              <Switch
                size="small"
                checked={autoRefresh}
                onChange={(checked) => setAutoRefresh(checked)}
              />
              {tr("subs.autopoll", "Auto poll list every 30s")}
            </span>
          </div>
        </div>

        {isLoading && !list && <Skeleton active paragraph={{ rows: 4 }} />}
        {error && (
          <ErrorState
            message={tr("subs.error.load", "Failed to load subscriptions: {message}", { message: (error as Error).message })}
            onRetry={() => refetch()}
          />
        )}

        {filteredList && filteredList.length > 0 ? (
          <SubscriptionTable
            list={filteredList}
            loading={isFetching}
            rowRefreshingId={rowRefreshingId}
            sortOrder={sortOrder}
            onToggleSort={() =>
              setSortOrder((prev) => (prev === "asc" ? "desc" : "asc"))
            }
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
            onEdit={(row) => {
              setEditingSub(row);
              setModalMode("edit");
              setModalOpen(true);
            }}
            onDelete={(row) => setDeleteId(row.id)}
            onRefreshRow={(row) => {
              setRowRefreshingId(row.id);
              refresh.mutate(row.id, {
                onSettled: () => setRowRefreshingId(null),
              });
            }}
          />
        ) : (
          !isLoading && (
            <EmptyState
              title={list && list.length > 0 ? tr("subs.empty.search.title", "No results") : tr("subs.empty.base.title", "No subscriptions yet")}
              description={
                list && list.length > 0
                  ? tr("subs.empty.search", "Try adjusting your search keywords.")
                  : tr("subs.empty.base", "Create your first subscription to start syncing.")
              }
              actionLabel={list && list.length > 0 ? undefined : tr("subs.new", "New Subscription")}
              onActionClick={list && list.length > 0 ? undefined : handleCreate}
            />
          )
        )}
      </Card>

      {/* Edit / Create modal */}
      <SubscriptionModal
        open={modalOpen}
        mode={modalMode}
        initialValues={
          editingSub
            ? {
                name: editingSub.name || editingSub.url,
                url: editingSub.url,
                auto_update_enabled: editingSub.auto_update_enabled ?? false,
                refresh_interval_sec: editingSub.refresh_interval_sec,
              }
            : undefined
        }
        submitting={
          modalMode === "create" ? create.isPending : update.isPending
        }
        onCancel={() => {
          setModalOpen(false);
          setEditingSub(null);
        }}
        onSubmit={(values) => {
          if (modalMode === "create") {
            if (!values.url) return;
            create.mutate(
              {
                url: values.url,
                name: values.name || undefined,
                auto_update_enabled: values.auto_update_enabled,
                refresh_interval_sec: values.refresh_interval_sec,
              },
              {
                onSuccess: () => {
                  setModalOpen(false);
                },
              }
            );
          } else {
            if (!editingSub) return;
            update.mutate(
              {
                id: editingSub.id,
                name: values.name,
                url: values.url,
                auto_update_enabled: values.auto_update_enabled,
                refresh_interval_sec: values.refresh_interval_sec,
              },
              {
                onSuccess: () => {
                  setModalOpen(false);
                  setEditingSub(null);
                },
              }
            );
          }
        }}
      />

      {/* Delete confirm */}
      {deleteId && (
        <Modal
          open={!!deleteId}
          title={tr("subs.delete.title", "Delete Subscription")}
          okText={tr("subs.delete.confirm", "Delete")}
          okButtonProps={{ danger: true, loading: del.isPending }}
          cancelText={tr("common.cancel", "Cancel")}
          onCancel={() => setDeleteId(null)}
          onOk={() => {
            if (!deleteId) return;
            del.mutate(deleteId, {
              onSuccess: () => setDeleteId(null),
            });
          }}
        >
          <p className="bp-text-danger" style={{ fontSize: 14 }}>
            {tr("subs.delete.desc", "This will remove the subscription and its nodes from DB. Continue?")}
          </p>
        </Modal>
      )}
    </div>
  );
}
