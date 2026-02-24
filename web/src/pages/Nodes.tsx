import { useMemo, useState } from "react";
import { useNodes, useUpdateNode } from "../hooks/useNodes";
import { useNodeForwarding, useUpdateNodeForwarding, useRestartNodeForwarding } from "../hooks/useNodeForwarding";
import { ErrorState } from "../components/common/ErrorState";
import { EmptyState } from "../components/common/EmptyState";
import { formatDateTime } from "../utils/datetime";
import { Button, Card, Drawer, Form, Input, InputNumber, Modal, Select, Switch, Table, Tag } from "antd";
import type { ColumnsType } from "antd/es/table";
import type { Node, ProxyType, ProxyConfig } from "../api/types";
import { buildProxyUrl } from "../api/settings";
import { useToast } from "../components/common/ToastContext";

export default function Nodes() {
  const { data: list, isLoading, error, refetch } = useNodes({});
  const update = useUpdateNode();
  const [search, setSearch] = useState("");
  const [detailOpen, setDetailOpen] = useState(false);
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);
  const [overrideOpen, setOverrideOpen] = useState(false);
  const [overrideType, setOverrideType] = useState<ProxyType>("http");
  const [useGlobal, setUseGlobal] = useState(false);
  const [form] = Form.useForm();
  const overrideAuthMode = Form.useWatch("auth_mode", form);
  const { addToast } = useToast();
  const forwarding = useNodeForwarding(selectedNode?.id);
  const updateForwarding = useUpdateNodeForwarding();
  const restartForwarding = useRestartNodeForwarding();

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
      <div className="bp-page-header">
        <h1 className="bp-page-title">Nodes</h1>
        <div className="bp-page-actions">
          <Button type="primary">Add Node</Button>
          <Button onClick={() => refetch()} loading={isLoading}>
            Refresh
          </Button>
        </div>
      </div>

      <Card>
        <div className="bp-card-toolbar">
          <Input
            className="bp-input"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search nodes"
          />
          <div className="bp-card-toolbar-meta">
            {filtered && (
              <span>
                Showing {filtered.length} of {list?.length ?? 0} nodes
              </span>
            )}
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
              }),
              (row) => openDetails(row)
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

      <Drawer
        title={selectedNode ? `Node Details: ${selectedNode.name || selectedNode.tag}` : "Node Details"}
        width={520}
        onClose={() => setDetailOpen(false)}
        open={detailOpen}
      >
        {selectedNode && (
          <div className="bp-drawer-section">
            <p className="bp-muted">Type: {selectedNode.type}</p>
            <p className="bp-muted">Tag: {selectedNode.tag}</p>
            <p className="bp-muted">Created: {formatDateTime(selectedNode.created_at)}</p>
          </div>
        )}

        <div className="bp-drawer-section">
          <h3 className="bp-card-title">Forwarding</h3>
          {forwarding.isLoading && <p className="bp-muted">Loading forwarding...</p>}
          {forwarding.data && (
            <div className="bp-forwarding-grid">
              {renderForwardingCard("HTTP Proxy", "http", forwarding.data.http)}
              {renderForwardingCard("SOCKS5 Proxy", "socks", forwarding.data.socks)}
            </div>
          )}
        </div>
      </Drawer>

      <Modal
        open={overrideOpen}
        title={`Edit ${overrideType.toUpperCase()} Override`}
        onCancel={() => setOverrideOpen(false)}
        okText="Save"
        onOk={() => handleOverrideSave()}
        confirmLoading={updateForwarding.isPending}
      >
        <div style={{ marginBottom: 12 }}>
          <span style={{ marginRight: 8 }}>Use Global Settings</span>
          <Switch checked={useGlobal} onChange={(checked) => setUseGlobal(checked)} />
        </div>
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            enabled: true,
            port: overrideType === "http" ? 7890 : 7891,
            auth_mode: "none",
            username: "",
            password: "",
          }}
        >
          <Form.Item name="enabled" label="Enabled" valuePropName="checked">
            <Switch disabled={useGlobal} />
          </Form.Item>
          <Form.Item
            name="port"
            label="Port"
            rules={[
              { required: true, message: "Port is required" },
              { type: "number", min: 1, max: 65535, message: "Port must be 1-65535" },
            ]}
          >
            <InputNumber min={1} max={65535} style={{ width: "100%" }} disabled={useGlobal} />
          </Form.Item>
          <Form.Item name="auth_mode" label="Auth Mode">
            <Select
              disabled={useGlobal}
              options={[
                { value: "none", label: "None" },
                { value: "basic", label: "Basic" },
              ]}
            />
          </Form.Item>
          <Form.Item
            name="username"
            label="Username"
            rules={[
              {
                required: overrideAuthMode === "basic" && !useGlobal,
                message: "Username is required for Basic auth",
              },
            ]}
          >
            <Input disabled={useGlobal} />
          </Form.Item>
          <Form.Item
            name="password"
            label="Password"
            rules={[
              {
                required: overrideAuthMode === "basic" && !useGlobal,
                message: "Password is required for Basic auth",
              },
            ]}
          >
            <Input.Password disabled={useGlobal} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );

  function openDetails(row: Node) {
    setSelectedNode(row);
    setDetailOpen(true);
  }

  function renderForwardingCard(title: string, type: ProxyType, cfg: ProxyConfig) {
    return (
      <div className="bp-forwarding-card" key={type}>
        <div className="bp-forwarding-header">
          <div>
            <p className="bp-card-kicker">{cfg.source === "override" ? "Override" : "Global"}</p>
            <h4 className="bp-card-title">{title}</h4>
          </div>
          <Tag color={cfg.status === "running" ? "success" : cfg.status === "error" ? "error" : "default"}>
            {cfg.status === "running" ? "Running" : cfg.status === "error" ? "Error" : "Stopped"}
          </Tag>
        </div>
        <div className="bp-forwarding-meta">
          <span>Enabled: {cfg.enabled ? "Yes" : "No"}</span>
          <span>Port: {cfg.port}</span>
          <span>Source: {cfg.source === "override" ? "Override" : "Global"}</span>
        </div>
        {cfg.error_message && <p className="bp-text-danger">{cfg.error_message}</p>}
        <div className="bp-page-actions">
          <Button onClick={() => copyConnection(cfg)}>Copy URL</Button>
          <Button onClick={() => openOverride(type)}>Edit</Button>
          <Button onClick={() => selectedNode && restartForwarding.mutate(selectedNode.id)}>
            Restart
          </Button>
        </div>
      </div>
    );
  }

  function openOverride(type: ProxyType) {
    if (!forwarding.data) return;
    const cfg = type === "http" ? forwarding.data.http : forwarding.data.socks;
    setOverrideType(type);
    setUseGlobal(cfg.source !== "override");
    form.setFieldsValue({
      enabled: cfg.enabled,
      port: cfg.port,
      auth_mode: cfg.auth_mode,
      username: cfg.username || "",
      password: cfg.password || "",
    });
    setOverrideOpen(true);
  }

  async function handleOverrideSave() {
    if (!selectedNode) return;
    if (useGlobal) {
      updateForwarding.mutate({
        node_id: selectedNode.id,
        proxy_type: overrideType,
        use_global: true,
      });
      setOverrideOpen(false);
      return;
    }
    const values = await form.validateFields();
    updateForwarding.mutate({
      node_id: selectedNode.id,
      proxy_type: overrideType,
      enabled: values.enabled,
      port: values.port,
      auth_mode: values.auth_mode,
      username: values.username,
      password: values.password,
    });
    setOverrideOpen(false);
  }

  async function copyConnection(cfg: ProxyConfig) {
    const url = buildProxyUrl(cfg);
    try {
      await navigator.clipboard.writeText(url);
      addToast("success", "Connection string copied");
    } catch {
      addToast("error", "Copy failed");
    }
  }
}

function buildColumns(
  updating: boolean,
  onToggleEnabled: (row: Node) => void,
  onShowDetails: (row: Node) => void
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
        <>
          <Button type="link" onClick={() => onShowDetails(record)}>
            Details
          </Button>
          <Button
            type="link"
            onClick={() => onToggleEnabled(record)}
            disabled={updating}
          >
            {updating ? "Updating..." : record.enabled ? "Disable" : "Enable"}
          </Button>
        </>
      ),
    },
  ];
}
